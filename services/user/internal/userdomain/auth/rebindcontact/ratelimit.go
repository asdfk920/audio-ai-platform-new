package rebindcontact

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

// EnsureRebindIPRateLimit 换绑接口按 IP 滑动窗口计数。
func EnsureRebindIPRateLimit(ctx context.Context, cfg config.RebindContact, clientIP string) error {
	if err := ctx.Err(); err != nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	key := fmt.Sprintf("user:rebind:ip:%s", sanitizeIPKey(clientIP))
	window := time.Duration(cfg.EffectiveIPRateLimitWindowMin()) * time.Minute
	max := cfg.EffectiveIPRateLimitMaxPerIP()

	n, err := redisx.Incr(ctx, key)
	if err != nil {
		logx.WithContext(ctx).Errorf("[rebindcontact] redis incr ip rate key=%s: %v", key, err)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if n == 1 {
		if err := redisx.Expire(ctx, key, window); err != nil {
			logx.WithContext(ctx).Errorf("[rebindcontact] redis expire ip rate key=%s: %v", key, err)
			return errorx.NewDefaultError(errorx.CodeRedisError)
		}
	}
	if int(n) > max {
		return errorx.NewDefaultError(errorx.CodeRebindContactRateLimit)
	}
	return nil
}

// EnsureRebindUserRateLimit 换绑接口按用户 ID 滑动窗口计数。
func EnsureRebindUserRateLimit(ctx context.Context, cfg config.RebindContact, userID int64) error {
	if err := ctx.Err(); err != nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if userID <= 0 {
		return errorx.NewDefaultError(errorx.CodeInvalidParam)
	}
	key := fmt.Sprintf("user:rebind:uid:%d", userID)
	window := time.Duration(cfg.EffectiveRateLimitWindowSec()) * time.Second
	max := cfg.EffectiveRateLimitMaxPerUser()

	n, err := redisx.Incr(ctx, key)
	if err != nil {
		logx.WithContext(ctx).Errorf("[rebindcontact] redis incr uid rate uid=%d: %v", userID, err)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if n == 1 {
		if err := redisx.Expire(ctx, key, window); err != nil {
			logx.WithContext(ctx).Errorf("[rebindcontact] redis expire uid rate uid=%d: %v", userID, err)
			return errorx.NewDefaultError(errorx.CodeRedisError)
		}
	}
	if int(n) > max {
		return errorx.NewDefaultError(errorx.CodeRebindContactRateLimit)
	}
	return nil
}

// EnsureRebindCooldown 若用户在冷却期内成功换绑过，则拒绝本次请求。
func EnsureRebindCooldown(ctx context.Context, cfg config.RebindContact, userID int64) error {
	d := cfg.EffectiveCooldown()
	if d <= 0 || userID <= 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	key := fmt.Sprintf("user:rebind:cooldown:%d", userID)
	n, err := redisx.Exists(ctx, key)
	if err != nil {
		logx.WithContext(ctx).Errorf("[rebindcontact] redis exists cooldown uid=%d: %v", userID, err)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if n > 0 {
		return errorx.NewDefaultError(errorx.CodeRebindContactCooldown)
	}
	return nil
}

// SetRebindCooldownAfterSuccess 换绑成功后设置冷却键 TTL。
func SetRebindCooldownAfterSuccess(ctx context.Context, cfg config.RebindContact, userID int64) {
	d := cfg.EffectiveCooldown()
	if d <= 0 || userID <= 0 {
		return
	}
	key := fmt.Sprintf("user:rebind:cooldown:%d", userID)
	if err := redisx.Set(ctx, key, "1", d); err != nil {
		logx.WithContext(ctx).Errorf("[rebindcontact] redis set cooldown uid=%d: %v", userID, err)
	}
}
