package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	internalconfig "github.com/jacklau/audio-ai-platform/services/media-processing/internal/config"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/handler"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var (
	configFile = flag.String("f", "etc/media-processing.yaml", "the config file")
)

type MediaTask struct {
	RawContentID int64  `json:"raw_content_id"`
	UserID       int64  `json:"user_id"`
	FileKey      string `json:"file_key"`
	FileSize     int64  `json:"file_size"`
}

func main() {
	flag.Parse()

	var c internalconfig.Config
	conf.MustLoad(*configFile, &c)

	ctx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(err)
	}
	defer func() {
		if ctx.DB != nil {
			_ = ctx.DB.Close()
		}
	}()

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	handler.RegisterHandlers(server, ctx)

	// 可选：继续跑原有 SQS worker（与 HTTP server 同进程）
	if c.SQS.Enabled && c.SQS.QueueURL != "" {
		awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(c.SQS.Region))
		if err == nil {
			sqsClient := sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
				// 本地 localstack endpoint 可通过环境/配置注入；此处仅保留旧逻辑扩展点
				_ = c.SQS.Endpoint
			})
			go processMessages(context.Background(), sqsClient, c.SQS.QueueURL)
		}
	}

	server.Start()
}

func processMessages(ctx context.Context, client *sqs.Client, queueURL string) {
	fmt.Printf("Starting message processing (queue: %s)...\n", queueURL)
	_ = client // TODO: use client to receive SQS messages

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// TODO: 从 SQS 接收消息并解析为 MediaTask，目前模拟处理
			time.Sleep(5 * time.Second)
			fmt.Printf("[%s] Waiting for media tasks...\n", time.Now().Format("15:04:05"))
			_ = handleMediaTask
		}
	}
}

func handleMediaTask(task MediaTask) error {
	fmt.Printf("Processing media task: RawContentID=%d, FileKey=%s\n",
		task.RawContentID, task.FileKey)

	// TODO: 实现实际的媒体处理逻辑
	// 1. 从 S3 下载原始文件
	// 2. 验证文件格式
	// 3. 发送到 AI Worker 队列

	return nil
}
