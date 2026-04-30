package register

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// EnsureRegisterIPRateLimit 注册接口按 IP 计数；窗口与阈值来自配置 Register。
func EnsureRegisterIPRateLimit(ctx context.Context, cfg config.Register, clientIP string) error {
	if err := ctx.Err(); err != nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	key := fmt.Sprintf("user:register:ip:%s", sanitizeIPKey(clientIP))
	window := time.Duration(cfg.EffectiveRateLimitWindowMin()) * time.Minute
	max := cfg.EffectiveRateLimitMaxPerIP()

	n, err := redisx.Incr(ctx, key)
	if err != nil {
		logx.WithContext(ctx).Errorf("[register] redis incr ip rate key=%s: %v", key, err)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if n == 1 {
		if err := redisx.Expire(ctx, key, window); err != nil {
			logx.WithContext(ctx).Errorf("[register] redis expire ip rate key=%s: %v", key, err)
			return errorx.NewDefaultError(errorx.CodeRedisError)
		}
	}
	if int(n) > max {
		return errorx.NewDefaultError(errorx.CodeRegisterRateLimit)
	}
	return nil
}

func registerTargetRateKey(target string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(target)))
	return fmt.Sprintf("user:register:target:%s", hex.EncodeToString(sum[:12]))
}

// EnsureRegisterTargetRateLimit 按注册目标（邮箱/手机）计数，窗口与阈值与 EnsureRegisterIPRateLimit 相同，防单号刷注册。
func EnsureRegisterTargetRateLimit(ctx context.Context, cfg config.Register, target string) error {
	if err := ctx.Err(); err != nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	key := registerTargetRateKey(target)
	window := time.Duration(cfg.EffectiveRateLimitWindowMin()) * time.Minute
	max := cfg.EffectiveRateLimitMaxPerIP()

	n, err := redisx.Incr(ctx, key)
	if err != nil {
		logx.WithContext(ctx).Errorf("[register] redis incr target rate key=%s: %v", key, err)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if n == 1 {
		if err := redisx.Expire(ctx, key, window); err != nil {
			logx.WithContext(ctx).Errorf("[register] redis expire target rate key=%s: %v", key, err)
			return errorx.NewDefaultError(errorx.CodeRedisError)
		}
	}
	if int(n) > max {
		return errorx.NewDefaultError(errorx.CodeRegisterRateLimit)
	}
	return nil
}
