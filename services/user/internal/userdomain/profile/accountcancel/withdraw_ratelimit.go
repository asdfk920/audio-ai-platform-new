package accountcancel

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

func sanitizeIPForKey(ip string) string {
	s := strings.TrimSpace(ip)
	if s == "" {
		return "unknown"
	}
	return strings.ReplaceAll(s, ":", "_")
}

// EnsureWithdrawUserRateLimit 撤销注销申请按用户 ID 滑动窗口计数。
func EnsureWithdrawUserRateLimit(ctx context.Context, cfg config.Cancellation, userID int64) error {
	if err := ctx.Err(); err != nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if userID <= 0 {
		return errorx.NewDefaultError(errorx.CodeInvalidParam)
	}
	key := fmt.Sprintf("user:cancellation_withdraw:uid:%d", userID)
	window := time.Duration(cfg.EffectiveWithdrawRateLimitWindowSec()) * time.Second
	max := cfg.EffectiveWithdrawRateLimitMaxPerUser()

	n, err := redisx.Incr(ctx, key)
	if err != nil {
		logx.WithContext(ctx).Errorf("[accountcancel.withdraw] redis incr uid rate uid=%d: %v", userID, err)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if n == 1 {
		if err := redisx.Expire(ctx, key, window); err != nil {
			logx.WithContext(ctx).Errorf("[accountcancel.withdraw] redis expire uid rate uid=%d: %v", userID, err)
			return errorx.NewDefaultError(errorx.CodeRedisError)
		}
	}
	if int(n) > max {
		return errorx.NewDefaultError(errorx.CodeCancellationWithdrawRateLimit)
	}
	return nil
}

// EnsureWithdrawIPRateLimit 撤销注销申请按客户端 IP 滑动窗口计数。
func EnsureWithdrawIPRateLimit(ctx context.Context, cfg config.Cancellation, clientIP string) error {
	if err := ctx.Err(); err != nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	key := fmt.Sprintf("user:cancellation_withdraw:ip:%s", sanitizeIPForKey(clientIP))
	window := time.Duration(cfg.EffectiveWithdrawIPRateLimitWindowMin()) * time.Minute
	max := cfg.EffectiveWithdrawIPRateLimitMaxPerIP()

	n, err := redisx.Incr(ctx, key)
	if err != nil {
		logx.WithContext(ctx).Errorf("[accountcancel.withdraw] redis incr ip rate key=%s: %v", key, err)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if n == 1 {
		if err := redisx.Expire(ctx, key, window); err != nil {
			logx.WithContext(ctx).Errorf("[accountcancel.withdraw] redis expire ip rate key=%s: %v", key, err)
			return errorx.NewDefaultError(errorx.CodeRedisError)
		}
	}
	if int(n) > max {
		return errorx.NewDefaultError(errorx.CodeCancellationWithdrawRateLimit)
	}
	return nil
}
