package config

import (
	apicors "github.com/jacklau/audio-ai-platform/common/cors"
	"github.com/zeromicro/go-zero/rest"
)

type Config struct {
	rest.RestConf
	Postgres struct {
		DataSource string
	}
	Redis struct {
		Addr     string
		Password string
		DB       int
		Optional bool
	}
	Auth struct {
		AccessSecret string
	}
	// DeviceService 若配置 BaseURL，则点播 / 跳转请求将转发至设备服务，
	// 由设备服务写入 device_instruction 并通过既有 WebSocket 下发（与同服务 pause 一致）。
	DeviceService struct {
		BaseURL string `json:",optional"` // 例：http://127.0.0.1:8002
	} `json:",optional"`
	Stream struct {
		RTMPBaseURL       string
		FLVBaseURL        string
		DefaultSecretKey  string
		DefaultConfigID   string
		DefaultExpiresSec int64
	}
	SQS struct {
		Enabled  bool
		Endpoint string
		QueueURL string
		Region   string
	}
	CORS apicors.Config `json:",optional"`
}
