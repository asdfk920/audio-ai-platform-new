package refreshtoken

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// RevokeAllForUser 删除该用户在 Redis 中的 refresh 索引及当前 refresh 键（申请注销等场景）。
func RevokeAllForUser(ctx context.Context, userID int64) error {
	if ctx == nil || userID <= 0 {
		return nil
	}
	idx := KeyUserIndex(userID)
	old, gerr := redisx.Get(ctx, idx)
	if gerr != nil && gerr != redis.Nil {
		logx.WithContext(ctx).Errorf("RevokeAllForUser get idx uid=%d: %v", userID, gerr)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	if gerr == redis.Nil {
		old = ""
	}
	if old != "" {
		if derr := redisx.Del(ctx, KeyRefresh(old)); derr != nil {
			logx.WithContext(ctx).Errorf("RevokeAllForUser del refresh: %v", derr)
			return errorx.NewDefaultError(errorx.CodeRedisError)
		}
	}
	if derr := redisx.Del(ctx, idx); derr != nil {
		logx.WithContext(ctx).Errorf("RevokeAllForUser del idx: %v", derr)
		return errorx.NewDefaultError(errorx.CodeRedisError)
	}
	return nil
}
