package config

// Cancellation 账号注销：冷静期天数、文案长度、是否要求已实名（依赖业务前置校验时可打开）。
type Cancellation struct {
	CoolingDays     int  `json:",optional"` // 默认 7，范围 7–30
	MaxReasonRunes  int  `json:",optional"` // 注销原因最大长度（按 rune），默认 500
	RequireRealName bool `json:",optional"` // 为 true 时仅 real_name_status=1 可申请
	// WithdrawRateLimitWindowSec / WithdrawRateLimitMaxPerUser：撤销注销申请按用户限流
	WithdrawRateLimitWindowSec  int `json:",optional"`
	WithdrawRateLimitMaxPerUser int `json:",optional"`
	// WithdrawIPRateLimitWindowMin / WithdrawIPRateLimitMaxPerIP：按客户端 IP 限流（防刷）
	WithdrawIPRateLimitWindowMin int `json:",optional"`
	WithdrawIPRateLimitMaxPerIP  int `json:",optional"`
	
	// 定时任务配置
	CleanupCronExpr string `json:",optional"` // cron 表达式，默认 "0 0 2 * * *" (每天凌晨 2 点)
	BatchSize       int   `json:",optional"` // 每批次处理用户数，默认 100
}

// EffectiveCleanupCronExpr 获取有效的 cron 表达式
func (c Cancellation) EffectiveCleanupCronExpr() string {
	if c.CleanupCronExpr == "" {
		return "0 0 2 * * *" // 默认每天凌晨 2 点
	}
	return c.CleanupCronExpr
}

// EffectiveBatchSize 获取有效的批次大小
func (c Cancellation) EffectiveBatchSize() int {
	if c.BatchSize <= 0 {
		return 100
	}
	if c.BatchSize > 1000 {
		return 1000
	}
	return c.BatchSize
}

// EffectiveCoolingDays 冷静期天数：未配置或非法时默认 7，最大 30，最小 7。
func (c Cancellation) EffectiveCoolingDays() int {
	d := c.CoolingDays
	if d <= 0 {
		d = 7
	}
	if d < 7 {
		d = 7
	}
	if d > 30 {
		d = 30
	}
	return d
}

// EffectiveMaxReasonRunes 注销原因长度上限。
func (c Cancellation) EffectiveMaxReasonRunes() int {
	if c.MaxReasonRunes <= 0 {
		return 500
	}
	if c.MaxReasonRunes > 2000 {
		return 2000
	}
	return c.MaxReasonRunes
}

// EffectiveWithdrawRateLimitWindowSec 撤销注销：单用户计数窗口（秒），默认 300。
func (c Cancellation) EffectiveWithdrawRateLimitWindowSec() int {
	if c.WithdrawRateLimitWindowSec <= 0 {
		return 300
	}
	if c.WithdrawRateLimitWindowSec > 3600 {
		return 3600
	}
	return c.WithdrawRateLimitWindowSec
}

// EffectiveWithdrawRateLimitMaxPerUser 窗口内单用户最大撤销次数，默认 10。
func (c Cancellation) EffectiveWithdrawRateLimitMaxPerUser() int {
	if c.WithdrawRateLimitMaxPerUser <= 0 {
		return 10
	}
	if c.WithdrawRateLimitMaxPerUser > 100 {
		return 100
	}
	return c.WithdrawRateLimitMaxPerUser
}

// EffectiveWithdrawIPRateLimitWindowMin 撤销注销：单 IP 计数窗口（分钟），默认 1。
func (c Cancellation) EffectiveWithdrawIPRateLimitWindowMin() int {
	if c.WithdrawIPRateLimitWindowMin <= 0 {
		return 1
	}
	if c.WithdrawIPRateLimitWindowMin > 1440 {
		return 1440
	}
	return c.WithdrawIPRateLimitWindowMin
}

// EffectiveWithdrawIPRateLimitMaxPerIP 窗口内单 IP 最大撤销次数，默认 30。
func (c Cancellation) EffectiveWithdrawIPRateLimitMaxPerIP() int {
	if c.WithdrawIPRateLimitMaxPerIP <= 0 {
		return 30
	}
	if c.WithdrawIPRateLimitMaxPerIP > 1000 {
		return 1000
	}
	return c.WithdrawIPRateLimitMaxPerIP
}
