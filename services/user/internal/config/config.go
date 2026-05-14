package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	Auth struct {
		AccessSecret string
		AccessExpire int64
	}
	Register      Register
	ResetPassword ResetPassword `json:",optional"`
	UpdateProfile UpdateProfile `json:",optional"`
	RebindContact RebindContact `json:",optional"`
	Security      Security
	Notify        Notify
	Login         Login
	Postgres      struct {
		DataSource string
	}
	Redis struct {
		Addr     string
		Password string
		DB       int
	}
	OAuth struct {
		WeChat struct {
			AppId     string
			AppSecret string
			// RedirectURL 微信 OAuth2 回调完整地址；为空则按 Host/Port 拼 /api/v1/user/oauth/wechat/callback
			RedirectURL string `json:",optional"`
		}
		Google struct {
			ClientID     string
			ClientSecret string
			// RedirectURL OAuth 回调完整地址；为空则使用 http://{Host}:{Port}/api/v1/user/oauth/google/callback（Host 为 0.0.0.0 时用 localhost）
			RedirectURL string `json:",optional"`
		}
	}
	VerifyCode  VerifyCodeConfig
	DeviceShare DeviceShareWorker `json:",optional"`
	// 设备绑定配置
	MaxDeviceBinds int `json:",default=10"` // 用户最大绑定设备数
	// 文件上传配置
	Upload UploadConfig `json:",optional"`
}

// UploadConfig 文件上传配置。
type UploadConfig struct {
	// MaxFileSize 上传文件大小限制（MB），默认 10MB
	MaxFileSize int64 `json:",default=10"`
	// SavePath 本地存储根目录，默认 ./uploads
	SavePath string `json:",default=./uploads"`
	// AllowedExtensions 允许的文件扩展名，默认 jpg,jpeg,png,gif,webp
	AllowedExtensions []string `json:",optional"`
}

type DeviceShareWorker struct {
	ExpireCronExpr string `json:",optional"`
	BatchSize      int    `json:",optional"`
}

// VerifyCodeConfig 验证码：过期、频控、黑名单等（供 util 与配置加载共用类型名）。
type VerifyCodeConfig struct {
	ExpireSeconds int // 验证码过期秒数，默认 180
	MaxPerMinute  int // 每分钟最多发送条数，默认 3
	BlockMinutes  int // 超过后禁止发送的分钟数，默认 3
	// 敏感号段/邮箱片段（子串匹配，不区分大小写）
	BlacklistMobiles []string `json:",optional"`
	BlacklistEmails  []string `json:",optional"`
	SendLockSeconds  int      `json:",optional"` // 发码分布式锁 TTL（秒）
	DeliveryRetry    int      `json:",optional"` // 投递失败额外重试次数
}
