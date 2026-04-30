package reqguard

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
)

// Ctx 防止 ctx 为空导致后续 panic。
func Ctx(ctx context.Context) error {
	if ctx == nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	return nil
}

// Service 校验 ServiceContext 非空。
func Service(ctx context.Context, svc *svc.ServiceContext) error {
	if err := Ctx(ctx); err != nil {
		return err
	}
	if svc == nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	return nil
}

// UserRepo 校验用户仓储已注入。
func UserRepo(ctx context.Context, svc *svc.ServiceContext) error {
	if err := Service(ctx, svc); err != nil {
		return err
	}
	if svc.UserRepo == nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	return nil
}
