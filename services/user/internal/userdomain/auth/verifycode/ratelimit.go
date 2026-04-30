package verifycode

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/config"
	"github.com/zeromicro/go-zero/core/logx"
)

// EnsureSendRateLimit 1 分钟滑动窗口计数，超限后写入封禁键。
// 任一 Redis 步骤失败则整请求失败（fail-closed），避免限流失效导致刷接口。
func EnsureSendRateLimit(ctx context.Context, cfg config.VerifyCodeConfig, target string) error {
	if err := ctx.Err(); err != nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	bk := blockKey(target)
	cntKey := sendCountKey(target)

	blocked, n, err := redisx.ExistsIncrPipeline(ctx, bk, cntKey)
	if err != nil {
		logx.WithContext(ctx).Errorf("[verifycode] redis rate-limit pipeline: %v", err)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if blocked > 0 {
		return errorx.NewDefaultError(errorx.CodeVerifyCodeLimit)
	}

	if n == 1 {
		if err := redisx.Expire(ctx, cntKey, sendWindow); err != nil {
			logx.WithContext(ctx).Errorf("[verifycode] redis expire send cnt: %v", err)
			return errorx.NewDefaultError(errorx.CodeRedisError)
		}
	}

	limit := maxPerMinute(cfg)
	if n > limit {
		if err := redisx.Set(ctx, bk, "1", blockDuration(cfg)); err != nil {
			logx.WithContext(ctx).Errorf("[verifycode] redis set verify block: %v", err)
			return errorx.NewDefaultError(errorx.CodeRedisError)
		}
		return errorx.NewDefaultError(errorx.CodeVerifyCodeLimit)
	}
	return nil
}
