package verifycode

import (
	"time"

	"github.com/jacklau/audio-ai-platform/services/user/internal/config"
)

const (
	defaultExpireSec = 180 // 3 分钟
	defaultMaxPerMin = 3
	defaultBlockMin  = 3
	sendWindow       = time.Minute
)

// CodeTTL 验证码在 Redis 中的存活时间（默认 3 分钟）。
func CodeTTL(cfg config.VerifyCodeConfig) time.Duration {
	if cfg.ExpireSeconds > 0 {
		return time.Duration(cfg.ExpireSeconds) * time.Second
	}
	return defaultExpireSec * time.Second
}

// ExpireSecondsForResponse 返回给前端的过期秒数。
func ExpireSecondsForResponse(cfg config.VerifyCodeConfig) int {
	if cfg.ExpireSeconds > 0 {
		return cfg.ExpireSeconds
	}
	return defaultExpireSec
}

func maxPerMinute(cfg config.VerifyCodeConfig) int64 {
	if cfg.MaxPerMinute > 0 {
		return int64(cfg.MaxPerMinute)
	}
	return int64(defaultMaxPerMin)
}

func blockDuration(cfg config.VerifyCodeConfig) time.Duration {
	if cfg.BlockMinutes > 0 {
		return time.Duration(cfg.BlockMinutes) * time.Minute
	}
	return defaultBlockMin * time.Minute
}
