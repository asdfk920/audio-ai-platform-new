package resetpassword

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/config"
	"github.com/zeromicro/go-zero/core/logx"
)

func ipRateLimitKey(clientIP string) string {
	s := strings.TrimSpace(clientIP)
	if s == "" {
		s = "unknown"
	}
	s = strings.ReplaceAll(s, ":", "_")
	return fmt.Sprintf("user:reset_pwd:ip:%s", s)
}

// EnsureIPRateLimit 重置密码接口按 IP 限流；每次进入业务即计数，防止暴力尝试。
func EnsureIPRateLimit(ctx context.Context, cfg config.ResetPassword, clientIP string) error {
	if err := ctx.Err(); err != nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	key := ipRateLimitKey(clientIP)
	window := time.Duration(cfg.EffectiveRateLimitWindowMin()) * time.Minute
	max := cfg.EffectiveRateLimitMaxPerIP()

	n, err := redisx.Incr(ctx, key)
	if err != nil {
		logx.WithContext(ctx).Errorf("[resetpassword] redis incr rate key=%s: %v", key, err)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if n == 1 {
		if err := redisx.Expire(ctx, key, window); err != nil {
			logx.WithContext(ctx).Errorf("[resetpassword] redis expire rate key=%s: %v", key, err)
			return errorx.NewDefaultError(errorx.CodeRedisError)
		}
	}
	if int(n) > max {
		return errorx.NewDefaultError(errorx.CodeResetPasswordRateLimit)
	}
	return nil
}
