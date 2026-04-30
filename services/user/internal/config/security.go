package config

import "time"

// Register 注册相关限流与密码策略入口（具体哈希参数见 Security）。
type Register struct {
	MinPasswordLen        int `json:",optional"` // 默认 6
	RateLimitWindowMin    int `json:",optional"` // 单 IP 计数窗口（分钟），默认 60
	RateLimitMaxPerIP     int `json:",optional"` // 窗口内单 IP 最大注册提交次数，默认 20
	// TxTimeoutSec 注册主流程（Redis/DB）额外超时（秒）；0 表示仅用请求上下文，默认 0
	TxTimeoutSec        int `json:",optional"`
	RegisterLockSeconds int `json:",optional"` // 同账号注册分布式锁 TTL（秒），默认 30
	SubmitDedupSeconds  int `json:",optional"` // 同 IP+账号短时间重复提交拦截（秒），默认 5
}

// EffectiveMinPasswordLen 返回密码最小长度：未配置或小于 6 时按 6（与 common/validate 下限一致）。
func (r Register) EffectiveMinPasswordLen() int {
	if r.MinPasswordLen <= 0 || r.MinPasswordLen < 6 {
		return 6
	}
	return r.MinPasswordLen
}

// EffectiveRateLimitWindowMin 注册按 IP 限流窗口（分钟）。
func (r Register) EffectiveRateLimitWindowMin() int {
	if r.RateLimitWindowMin <= 0 {
		return 60
	}
	return r.RateLimitWindowMin
}

// EffectiveRateLimitMaxPerIP 窗口内允许的单 IP 注册请求次数。
func (r Register) EffectiveRateLimitMaxPerIP() int {
	if r.RateLimitMaxPerIP <= 0 {
		return 20
	}
	return r.RateLimitMaxPerIP
}

// EffectiveRegisterLockSeconds 同一注册目标（邮箱/手机）互斥锁 TTL。
func (r Register) EffectiveRegisterLockSeconds() int {
	if r.RegisterLockSeconds <= 0 {
		return 30
	}
	return r.RegisterLockSeconds
}

// EffectiveTxTimeout 注册逻辑层超时；<=0 不启用；上限 120 秒。
func (r Register) EffectiveTxTimeout() time.Duration {
	if r.TxTimeoutSec <= 0 {
		return 0
	}
	if r.TxTimeoutSec > 120 {
		return 120 * time.Second
	}
	return time.Duration(r.TxTimeoutSec) * time.Second
}

// EffectiveSubmitDedupSeconds 防重复提交窗口上限（避免误伤过长）。
func (r Register) EffectiveSubmitDedupSeconds() int {
	if r.SubmitDedupSeconds <= 0 {
		return 5
	}
	if r.SubmitDedupSeconds > 120 {
		return 120
	}
	return r.SubmitDedupSeconds
}

// UpdateProfile 修改昵称/头像：按用户 ID 限流，防恶意刷接口。
type UpdateProfile struct {
	RateLimitWindowSec  int `json:",optional"` // 计数窗口（秒），默认 60
	RateLimitMaxPerUser int `json:",optional"` // 窗口内单用户最大修改次数，默认 30
}

// EffectiveRateLimitWindowSec 资料修改限流窗口。
func (u UpdateProfile) EffectiveRateLimitWindowSec() int {
	if u.RateLimitWindowSec <= 0 {
		return 60
	}
	if u.RateLimitWindowSec > 3600 {
		return 3600
	}
	return u.RateLimitWindowSec
}

// EffectiveRateLimitMaxPerUser 单用户在窗口内允许的修改次数。
func (u UpdateProfile) EffectiveRateLimitMaxPerUser() int {
	if u.RateLimitMaxPerUser <= 0 {
		return 30
	}
	return u.RateLimitMaxPerUser
}

// Security 密码哈希与服务端校验参数。
type Security struct {
	PasswordHashAlgo string `json:",optional"` // bcrypt_concat | argon2id，默认 bcrypt_concat
	BcryptCost       int    `json:",optional"` // 默认沿用 pkg/passwd.BcryptCost
	Argon2Time       uint32 `json:",optional"`
	Argon2Memory     uint32 `json:",optional"` // 字节，默认 65536
	Argon2Threads    uint8  `json:",optional"`
	Argon2KeyLen     uint32 `json:",optional"`
}

// Notify 注册成功通知（邮件/短信占位，可接第三方）。
type Notify struct {
	RegisterEnabled bool   `json:",optional"`
	EmailSubject    string `json:",optional"`
	// RebindSecurityNotify 为 true 时，换绑成功会记录 [SECURITY_NOTIFY] 结构化日志，便于对接邮件/短信网关向旧联系方式告警。
	RebindSecurityNotify bool `json:",optional"`
}

// RebindContact 换绑邮箱/手机：按用户与 IP 限流、成功后的冷却期，防刷与频繁改绑。
type RebindContact struct {
	RateLimitWindowSec   int `json:",optional"` // 单用户计数窗口（秒），默认 3600
	RateLimitMaxPerUser  int `json:",optional"` // 窗口内单用户最大换绑提交次数，默认 10
	IPRateLimitWindowMin int `json:",optional"` // 单 IP 窗口（分钟），默认 60
	IPRateLimitMaxPerIP  int `json:",optional"` // 窗口内单 IP 最大次数，默认 30
	CooldownHours        int `json:",optional"` // 两次成功换绑之间的最短间隔（小时），0 表示不启用冷却，默认 24
}

// EffectiveRateLimitWindowSec 单用户换绑限流窗口。
func (r RebindContact) EffectiveRateLimitWindowSec() int {
	if r.RateLimitWindowSec <= 0 {
		return 3600
	}
	if r.RateLimitWindowSec > 86400 {
		return 86400
	}
	return r.RateLimitWindowSec
}

// EffectiveRateLimitMaxPerUser 窗口内允许的换绑请求次数。
func (r RebindContact) EffectiveRateLimitMaxPerUser() int {
	if r.RateLimitMaxPerUser <= 0 {
		return 10
	}
	return r.RateLimitMaxPerUser
}

// EffectiveIPRateLimitWindowMin 换绑接口按 IP 限流窗口（分钟）。
func (r RebindContact) EffectiveIPRateLimitWindowMin() int {
	if r.IPRateLimitWindowMin <= 0 {
		return 60
	}
	return r.IPRateLimitWindowMin
}

// EffectiveIPRateLimitMaxPerIP 单 IP 窗口内允许的换绑请求次数。
func (r RebindContact) EffectiveIPRateLimitMaxPerIP() int {
	if r.IPRateLimitMaxPerIP <= 0 {
		return 30
	}
	return r.IPRateLimitMaxPerIP
}

// EffectiveCooldown 两次成功换绑之间的间隔；CooldownHours<=0 时返回 0 表示不启用。
func (r RebindContact) EffectiveCooldown() time.Duration {
	if r.CooldownHours <= 0 {
		return 0
	}
	return time.Duration(r.CooldownHours) * time.Hour
}

// Login 登录：IP 名单、refresh、签发锁、JWT sid 黑名单、用户信息缓存、单会话策略。
type Login struct {
	IPWhitelist          []string `json:",optional"`
	IPBlacklist          []string `json:",optional"`
	RefreshTTLHours      int      `json:",optional"`
	LoginLockSeconds     int      `json:",optional"`
	JWTBlacklistDisabled bool     `json:",optional"`
	UserCacheTTLMinutes  int      `json:",optional"`
	SingleSessionOnLogin bool     `json:",optional"`
	// 刷新 Token 接口按 IP 限流（防暴力猜 refresh）
	RefreshTokenRateLimitWindowSec int `json:",optional"` // 默认 60 秒
	RefreshTokenRateLimitMaxPerIP  int `json:",optional"` // 窗口内单 IP 最大次数，默认 30
}

// EffectiveRefreshTTL 刷新令牌在 Redis 中的 TTL（与登录写入一致，宜配置化）。
func (l Login) EffectiveRefreshTTL() time.Duration {
	h := l.RefreshTTLHours
	if h <= 0 {
		h = 168
	}
	return time.Duration(h) * time.Hour
}

// EffectiveRefreshTokenRateWindowSec 刷新接口限流窗口（秒）。
func (l Login) EffectiveRefreshTokenRateWindowSec() int {
	if l.RefreshTokenRateLimitWindowSec <= 0 {
		return 60
	}
	if l.RefreshTokenRateLimitWindowSec > 3600 {
		return 3600
	}
	return l.RefreshTokenRateLimitWindowSec
}

// EffectiveRefreshTokenRateMaxPerIP 单 IP 在窗口内允许的刷新次数。
func (l Login) EffectiveRefreshTokenRateMaxPerIP() int {
	if l.RefreshTokenRateLimitMaxPerIP <= 0 {
		return 30
	}
	return l.RefreshTokenRateLimitMaxPerIP
}

// ResetPassword 忘记密码重置：按 IP 的接口级限流（防暴力尝试）。
type ResetPassword struct {
	RateLimitWindowMin int `json:",optional"` // 计数窗口（分钟），默认 60
	RateLimitMaxPerIP  int `json:",optional"` // 窗口内单 IP 最大成功提交次数，默认 10
}

// EffectiveRateLimitWindowMin 限流窗口分钟数。
func (r ResetPassword) EffectiveRateLimitWindowMin() int {
	if r.RateLimitWindowMin <= 0 {
		return 60
	}
	return r.RateLimitWindowMin
}

// EffectiveRateLimitMaxPerIP 单 IP 窗口内允许的重置请求次数。
func (r ResetPassword) EffectiveRateLimitMaxPerIP() int {
	if r.RateLimitMaxPerIP <= 0 {
		return 10
	}
	return r.RateLimitMaxPerIP
}

