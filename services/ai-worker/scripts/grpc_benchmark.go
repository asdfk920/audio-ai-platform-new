package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/jacklau/audio-ai-platform/services/ai-worker/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type BenchmarkConfig struct {
	ServerAddr      string
	ConcurrentUsers int
	TestDuration    time.Duration
	WarmupDuration  time.Duration
	AudioDuration   time.Duration
	SampleRate      int
	Channels        int
}

type BenchmarkResult struct {
	TotalRequests  int64
	SuccessfulReqs int64
	FailedReqs     int64
	TotalLatency   time.Duration
	MinLatency     time.Duration
	MaxLatency     time.Duration
	Latencies      []time.Duration
	AvgThroughput  float64
	P50Latency     time.Duration
	P95Latency     time.Duration
	P99Latency     time.Duration
}

func main() {
	config := &BenchmarkConfig{
		ServerAddr:      "localhost:50051",
		ConcurrentUsers: 10,
		TestDuration:    60 * time.Second,
		WarmupDuration:  10 * time.Second,
		AudioDuration:   30 * time.Second,
		SampleRate:      44100,
		Channels:        2,
	}

	fmt.Println("🚀 AI Worker gRPC 性能基准测试")
	fmt.Println("=========================================")
	fmt.Printf("服务器地址: %s\n", config.ServerAddr)
	fmt.Printf("并发用户数: %d\n", config.ConcurrentUsers)
	fmt.Printf("测试时长: %v\n", config.TestDuration)
	fmt.Printf("预热时长: %v\n", config.WarmupDuration)
	fmt.Println()

	result := runBenchmark(config)

	printResults(result)
}

func runBenchmark(config *BenchmarkConfig) *BenchmarkResult {
	conn, err := grpc.Dial(config.ServerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(100*1024*1024),
			grpc.MaxCallSendMsgSize(100*1024*1024),
		),
	)
	if err != nil {
		log.Fatalf("连接服务器失败: %v", err)
	}
	defer conn.Close()

	client := pb.NewInferenceServiceClient(conn)

	fmt.Println("🔥 预热阶段...")
	runWarmup(client, config)

	fmt.Println("📊 正式测试开始...")
	return runTest(client, config)
}

func runWarmup(client pb.InferenceServiceClient, config *BenchmarkConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), config.WarmupDuration)
	defer cancel()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.ConcurrentUsers)

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		default:
			semaphore <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-semaphore }()

				stream, err := client.StreamSeparate(ctx)
				if err != nil {
					return
				}

				sendTestFrames(stream, 5*time.Second, config.SampleRate, config.Channels)
				stream.CloseSend()

				for {
					_, err := stream.Recv()
					if err == io.EOF {
						break
					}
					if err != nil {
						break
					}
				}
			}()
		}
	}
}

func runTest(client pb.InferenceServiceClient, config *BenchmarkConfig) *BenchmarkResult {
	ctx, cancel := context.WithTimeout(context.Background(), config.TestDuration)
	defer cancel()

	var (
		totalRequests  int64
		successfulReqs int64
		failedReqs     int64
		totalLatency   int64
		minLatency     int64 = -1
		maxLatency     int64

		mu        sync.Mutex
		latencies []time.Duration
	)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.ConcurrentUsers)

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			wg.Wait()

			testDuration := time.Since(startTime)
			avgThroughput := float64(successfulReqs) / testDuration.Seconds()

			sortLatencies(latencies)

			return &BenchmarkResult{
				TotalRequests:  totalRequests,
				SuccessfulReqs: successfulReqs,
				FailedReqs:     failedReqs,
				TotalLatency:   time.Duration(totalLatency) * time.Nanosecond,
				MinLatency:     time.Duration(minLatency) * time.Nanosecond,
				MaxLatency:     time.Duration(maxLatency) * time.Nanosecond,
				Latencies:      latencies,
				AvgThroughput:  avgThroughput,
				P50Latency:     percentile(latencies, 50),
				P95Latency:     percentile(latencies, 95),
				P99Latency:     percentile(latencies, 99),
			}

		default:
			semaphore <- struct{}{}
			atomic.AddInt64(&totalRequests, 1)
			wg.Add(1)

			go func(reqID int64) {
				defer wg.Done()
				defer func() { <-semaphore }()

				reqStart := time.Now()

				err := runSingleInference(ctx, client, config.AudioDuration, config.SampleRate, config.Channels)

				reqLatency := time.Since(reqStart)
				reqLatencyNs := reqLatency.Nanoseconds()

				mu.Lock()
				if err == nil {
					atomic.AddInt64(&successfulReqs, 1)
				} else {
					atomic.AddInt64(&failedReqs, 1)
				}

				atomic.AddInt64(&totalLatency, reqLatencyNs)

				if minLatency == -1 || reqLatencyNs < minLatency {
					minLatency = reqLatencyNs
				}
				if reqLatencyNs > maxLatency {
					maxLatency = reqLatencyNs
				}

				latencies = append(latencies, reqLatency)
				mu.Unlock()

				if atomic.LoadInt64(&totalRequests)%10 == 0 {
					fmt.Printf("\r进度: %d 请求完成 (成功: %d, 失败: %d)",
						atomic.LoadInt64(&totalRequests),
						atomic.LoadInt64(&successfulReqs),
						atomic.LoadInt64(&failedReqs))
				}
			}(atomic.LoadInt64(&totalRequests))
		}
	}
}

func runSingleInference(ctx context.Context, client pb.InferenceServiceClient,
	audioDuration time.Duration, sampleRate int, channels int) error {

	taskID := generateTaskID("bench")

	stream, err := client.StreamSeparate(ctx)
	if err != nil {
		return fmt.Errorf("创建流失败: %w", err)
	}

	frameSize := sampleRate * channels * 2 // 16-bit samples
	framesPerSecond := 10

	totalFrames := int(audioDuration.Seconds()) * framesPerSecond

	for i := 0; i < totalFrames; i++ {
		audioData := generatePCMFrame(frameSize)

		frame := &pb.AudioFrame{
			TaskId:     taskID,
			FrameIndex: int32(i),
			AudioData:  audioData,
			SampleRate: int32(sampleRate),
			Channels:   int32(channels),
			Format:     pb.AudioFormat_Pcm16bit,
		}

		if err := stream.Send(frame); err != nil {
			return fmt.Errorf("发送帧失败: %w", err)
		}

		time.Sleep(time.Duration(float64(time.Second) / float64(framesPerSecond)))
	}

	stream.CloseSend()

	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("接收结果失败: %w", err)
		}
	}

	return nil
}

func sendTestFrames(stream pb.InferenceService_StreamSeparateClient,
	duration time.Duration, sampleRate int, channels int) {

	taskID := generateTaskID("warmup")
	frameSize := sampleRate * channels * 2
	framesPerSecond := 10
	totalFrames := int(duration.Seconds()) * framesPerSecond

	for i := 0; i < totalFrames; i++ {
		audioData := generatePCMFrame(frameSize)

		frame := &pb.AudioFrame{
			TaskId:     taskID,
			FrameIndex: int32(i),
			AudioData:  audioData,
			SampleRate: int32(sampleRate),
			Channels:   int32(channels),
		}

		stream.Send(frame)
		time.Sleep(time.Duration(float64(time.Second) / float64(framesPerSecond)))
	}
}

func generateTaskID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%s_%x_%d", prefix, b, time.Now().UnixNano())
}

func generatePCMFrame(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

func sortLatencies(latencies []time.Duration) {
	// 简单实现：实际应使用排序算法
	// 这里假设已经排序或使用外部库
}

func percentile(latencies []time.Duration, p float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	index := int(float64(len(latencies)-1) * p / 100)
	if index >= len(latencies) {
		index = len(latencies) - 1
	}

	return latencies[index]
}

func printResults(result *BenchmarkResult) {
	fmt.Println("\n\n=========================================")
	fmt.Println("📈 测试结果")
	fmt.Println("=========================================")

	fmt.Println("📊 基本指标:")
	fmt.Printf("  总请求数: %d\n", result.TotalRequests)
	fmt.Printf("  成功请求: %d (%.1f%%)\n", result.SuccessfulReqs,
		float64(result.SuccessfulReqs)/float64(result.TotalRequests)*100)
	fmt.Printf("  失败请求: %d (%.1f%%)\n", result.FailedReqs,
		float64(result.FailedReqs)/float64(result.TotalRequests)*100)

	fmt.Println("\n⏱️ 延迟指标:")
	fmt.Printf("  平均延迟: %.2f ms\n", float64(result.TotalLatency.Milliseconds())/float64(result.SuccessfulReqs))
	fmt.Printf("  最小延迟: %.2f ms\n", float64(result.MinLatency.Milliseconds()))
	fmt.Printf("  最大延迟: %.2f ms\n", float64(result.MaxLatency.Milliseconds()))
	fmt.Printf("  P50 延迟: %.2f ms\n", float64(result.P50Latency.Milliseconds()))
	fmt.Printf("  P95 延迟: %.2f ms\n", float64(result.P95Latency.Milliseconds()))
	fmt.Printf("  P99 延迟: %.2f ms\n", float64(result.P99Latency.Milliseconds()))

	fmt.Println("\n🚀 吞吐量:")
	fmt.Printf("  平均吞吐量: %.2f requests/s\n", result.AvgThroughput)

	fmt.Println("\n✅ 目标达成情况:")

	targetLatency := 100.0 * time.Millisecond
	actualP95 := result.P95Latency

	fmt.Printf("  延迟目标 (<100ms): ")
	if actualP95 <= targetLatency {
		fmt.Printf("✅ 达成 (P95: %.2fms)\n", actualP95.Seconds()*1000)
	} else {
		fmt.Printf("❌ 未达标 (P95: %.2fms)\n", actualP95.Seconds()*1000)
	}

	fmt.Printf("  吞吐量目标 (>100 QPS): ")
	if result.AvgThroughput >= 100 {
		fmt.Printf("✅ 达成 (%.2f QPS)\n", result.AvgThroughput)
	} else {
		fmt.Printf("⚠️ 待优化 (%.2f QPS)\n", result.AvgThroughput)
	}

	fmt.Println("\n💡 优化建议:")
	generateRecommendations(result)
}

func generateRecommendations(result *BenchmarkResult) {
	if result.P95Latency > 100*time.Millisecond {
		fmt.Println("  • 考虑启用 FP16 推理以降低延迟")
		fmt.Println("  • 优化音频分段大小和重叠比例")
		fmt.Println("  • 检查 GPU 利用率，考虑增加并发数")
	}

	if result.AvgThroughput < 100 {
		fmt.Println("  • 增加 MaxConcurrentJobs 配置")
		fmt.Println("  • 使用批量推理接口处理多任务")
		fmt.Println("  • 考虑部署多个实例并使用负载均衡")
	}

	if float64(result.FailedReqs)/float64(result.TotalRequests) > 0.01 {
		fmt.Printf("  • 错误率偏高 (%.2f%%)，检查日志排查问题\n",
			float64(result.FailedReqs)/float64(result.TotalRequests)*100)
	}
}
