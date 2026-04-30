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
	VerifyCode          VerifyCodeConfig
	RealName            RealName                  `json:",optional"`
	Cancellation        Cancellation              `json:",optional"`
	DeviceShare         DeviceShareWorker         `json:",optional"`
	MemberAutoRenew     MemberAutoRenewWorker     `json:",optional"`
	DownloadRecordClean DownloadRecordCleanWorker `json:",optional"`
	// Payment 会员订单支付：MockCallbackSecret 非空时，回调 JSON 验签 HMAC-SHA256(secret, order_no+"|"+trade_no)
	Payment Payment `json:",optional"`
	// 设备绑定配置
	MaxDeviceBinds int `json:",default=10"` // 用户最大绑定设备数
}

// Payment 配置（微信/支付宝正式对接可扩展字段）。
type Payment struct {
	MockCallbackSecret string `json:",optional"`
}

type DeviceShareWorker struct {
	ExpireCronExpr string `json:",optional"`
	BatchSize      int    `json:",optional"`
}

// MemberAutoRenewWorker 自动续费占位扫描：仅打日志，不真实扣款。
type MemberAutoRenewWorker struct {
	CronExpr               string `json:",optional"` // 默认每日一次
	WithinDaysBeforeExpire int    `json:",optional"` // 到期前窗口（天）
	BatchSize              int    `json:",optional"`
}

// DownloadRecordCleanWorker 下载记录清理定时任务配置
type DownloadRecordCleanWorker struct {
	CronExpr              string `json:",optional"` // cron 表达式，默认每天凌晨 3 点
	FreeRetentionDays     int    `json:",optional"` // 免费版保留天数，默认 7 天
	StandardRetentionDays int    `json:",optional"` // 标准版保留天数，默认 365 天
	BatchSize             int    `json:",optional"` // 每批次清理数量，默认 1000
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
