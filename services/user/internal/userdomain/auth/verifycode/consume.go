package verifycode

import (
	"context"

	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeleteStored 校验通过并完成后删除验证码；Redis 失败时打日志并返回错误（业务已成功时通常仍返回 nil 给前端，但服务端可追溯）。
func DeleteStored(ctx context.Context, target string) error {
	err := redisx.Del(ctx, CodeKey(target))
	if err != nil {
		logx.WithContext(ctx).Errorf("[verifycode] redis del verify code target=%s: %v", MaskTargetLoose(target), err)
	}
	return err
}
