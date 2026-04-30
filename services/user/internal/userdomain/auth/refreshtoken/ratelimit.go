package refreshtoken

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

func sanitizeIPKey(ip string) string {
	s := strings.TrimSpace(ip)
	if s == "" {
		return "unknown"
	}
	return strings.ReplaceAll(s, ":", "_")
}

// EnsureRefreshIPRateLimit 刷新 Token 接口按 IP 滑动窗口计数。
func EnsureRefreshIPRateLimit(ctx context.Context, cfg config.Login, clientIP string) error {
	if err := ctx.Err(); err != nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	key := fmt.Sprintf("user:refresh_api:ip:%s", sanitizeIPKey(clientIP))
	window := time.Duration(cfg.EffectiveRefreshTokenRateWindowSec()) * time.Second
	max := cfg.EffectiveRefreshTokenRateMaxPerIP()

	n, err := redisx.Incr(ctx, key)
	if err != nil {
		logx.WithContext(ctx).Errorf("[refreshtoken] redis incr rate key=%s: %v", key, err)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if n == 1 {
		if err := redisx.Expire(ctx, key, window); err != nil {
			logx.WithContext(ctx).Errorf("[refreshtoken] redis expire rate key=%s: %v", key, err)
			return errorx.NewDefaultError(errorx.CodeRedisError)
		}
	}
	if int(n) > max {
		return errorx.NewDefaultError(errorx.CodeRefreshRateLimit)
	}
	return nil
}
