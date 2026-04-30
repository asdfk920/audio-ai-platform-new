package userprofile

import (
	"context"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/config"
	"github.com/zeromicro/go-zero/core/logx"
)

// EnsureUpdateRateLimit 按用户 ID 滑动窗口计数，超限返回 CodeUpdateProfileRateLimit。
func EnsureUpdateRateLimit(ctx context.Context, cfg config.UpdateProfile, userID int64) error {
	if err := ctx.Err(); err != nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if userID <= 0 {
		return errorx.NewDefaultError(errorx.CodeInvalidParam)
	}
	key := fmt.Sprintf("user:profile_update:uid:%d", userID)
	window := time.Duration(cfg.EffectiveRateLimitWindowSec()) * time.Second
	max := cfg.EffectiveRateLimitMaxPerUser()

	n, err := redisx.Incr(ctx, key)
	if err != nil {
		logx.WithContext(ctx).Errorf("[userprofile] redis incr rate uid=%d: %v", userID, err)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if n == 1 {
		if err := redisx.Expire(ctx, key, window); err != nil {
			logx.WithContext(ctx).Errorf("[userprofile] redis expire rate uid=%d: %v", userID, err)
			return errorx.NewDefaultError(errorx.CodeRedisError)
		}
	}
	if int(n) > max {
		return errorx.NewDefaultError(errorx.CodeUpdateProfileRateLimit)
	}
	return nil
}
