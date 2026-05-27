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
	// Spotify 第三方音频源配置
	Spotify struct {
		ClientID     string `json:",optional"`
		ClientSecret string `json:",optional"`
		BaseURL      string `json:",optional"` // Spotify API 基础地址
		MockMode     bool   `json:",optional"` // 是否使用模拟模式（开发测试）
	} `json:",optional"`
	// DRMPass 版权/DRM 头透传与可选平台侧防篡改 RSA-PSS 签名（不解密载荷、不涉及内容密钥）。
	DRMPass struct {
		AttestSigning  bool   `json:",optional"` // true 且在私钥可读时写入 X-DRM-Platform-Signature
		Issuer         string `json:",optional"` // X-DRM-Platform-Issuer（非秘密）
		SigningKeyPath string `json:",optional"` // PEM 私钥路径
		SigningKeyPEM  string `json:",optional"` // PEM 私钥正文（不推荐提交到仓库）
	} `json:",optional"`
	// ThirdPartyMusic 网易云音乐 IoT 开放平台配置
	ThirdPartyMusic struct {
		BaseURL    string `json:",optional"` // API 基础地址（https://music.163.com）
		AppID      string `json:",optional"` // 应用 ID
		AppSecret  string `json:",optional"` // 应用密钥（用于 HMAC-SHA256 签名）
		PrivateKey string `json:",optional"` // RSA 私钥（PEM 格式，用于签名）
		TimeoutSec int    `json:",optional"` // 请求超时时间（秒），默认10
		EnableMock bool   `json:",optional"` // 测试开关：true 时不调用真实第三方
	} `json:",optional"`
}
