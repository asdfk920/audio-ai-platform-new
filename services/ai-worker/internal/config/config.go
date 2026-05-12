package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	// 认证配置
	Auth struct {
		AccessSecret string `json:",default=audio-ai-platform-jwt-secret"`
	}
	// 数据库配置
	Database struct {
		DataSource string `json:",optional"`
		Host       string `json:",default=localhost"`
		Port       int    `json:",default=5432"`
		User       string `json:",default=postgres"`
		Password   string `json:",default=postgres"`
		DBName     string `json:",default=audio_platform"`
	}
	// Redis 配置
	CacheRedis []redis.RedisConf `json:",optional"`
	// 对象存储配置
	Storage struct {
		Driver    string `json:",default=local"` // local | s3
		Region    string `json:",optional"`
		Endpoint  string `json:",optional"`
		AccessKey string `json:",optional"`
		SecretKey string `json:",optional"`
		Bucket    string `json:",optional"`
		Local     struct {
			Root string `json:",default=./data/ai-worker-objects"`
		}
	}
	// SQS 队列配置
	SQS struct {
		Endpoint  string `json:",optional"`
		Region    string `json:",default=us-east-1"`
		QueueURL  string `json:",optional"`
		AccessKey string `json:",optional"`
		SecretKey string `json:",optional"`
	}
	// AI 模型配置
	AI struct {
		ModelType         string   `json:",default=bsroformer"`                // 默认模型类型：BSRoformer
		ModelName         string   `json:",default=bss_roformer"`              // 模型名称
		ModelPath         string   `json:",default=/app/models/bsroformer"`    // 模型路径
		GPUDevice         int      `json:",default=0"`                         // GPU 设备 ID
		GPUID             int      `json:",default=0"`                         // 兼容旧字段
		SegmentSize       float64  `json:",default=10"`                        // 音频分段长度 (秒)
		Overlap           float64  `json:",default=0.25"`                      // 分段重叠比例
		UseFP16           bool     `json:",default=true"`                      // 使用 FP16 半精度
		UseINT8           bool     `json:",default=false"`                     // 使用 INT8 量化
		BatchSize         int      `json:",default=1"`                         // 批量推理大小
		OutputDir         string   `json:",default=/app/output"`               // 输出目录
		CacheDir          string   `json:",default=/app/cache"`                // 缓存目录
		MaxConcurrentJobs int      `json:",default=3"`                         // 最大并发任务数
		DefaultStems      []string `json:",default=[vocals,drums,bass,other]"` // 默认输出音轨
		// 性能优化配置
		UseCUDA       bool `json:",default=true"`  // 使用 CUDA 加速
		CUDABenchmark bool `json:",default=false"` // CUDA 性能模式
		NumThreads    int  `json:",default=4"`     // CPU 线程数
		InterThreads  int  `json:",default=2"`     // 线程间并行数
	}
	// gRPC 配置
	GRPC struct {
		Port           int    `json:",default=50051"`
		TLSCertFile    string `json:",optional"`
		TLSKeyFile     string `json:",optional"`
		MaxRecvMsgSize int    `json:",default=104857600"` // 100MB
		MaxSendMsgSize int    `json:",default=104857600"` // 100MB
		EnableTLS      bool   `json:",default=false"`
	}
	// 监控配置
	Monitoring struct {
		PrometheusPort int  `json:",default=9090"`
		MetricsEnabled bool `json:",default=true"`
	}
	// 回调配置
	Callback struct {
		TimeoutSec int `json:",default=30"`
		RetryCount int `json:",default=3"`
	}
}
