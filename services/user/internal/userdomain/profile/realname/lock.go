package realname

import (
	"context"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/zeromicro/go-zero/core/logx"
)

// TryAcquireSubmitLock 单用户提交互斥，防止短时间内重复点击。
func TryAcquireSubmitLock(ctx context.Context, userID int64, ttl time.Duration) (ok bool, err error) {
	if userID <= 0 {
		return false, errorx.NewDefaultError(errorx.CodeInvalidParam)
	}
	if ttl <= 0 {
		ttl = 5 * time.Second
	}
	key := fmt.Sprintf("user:realname:submit:lock:%d", userID)
	created, err := redisx.SetNX(ctx, key, "1", ttl)
	if err != nil {
		logx.WithContext(ctx).Errorf("[realname] submit lock redis: %v", err)
		return false, errorx.NewDefaultError(errorx.CodeRedisError)
	}
	return created, nil
}
