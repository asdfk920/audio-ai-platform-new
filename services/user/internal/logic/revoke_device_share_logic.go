package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/devicesharesvc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type RevokeDeviceShareLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRevokeDeviceShareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeDeviceShareLogic {
	return &RevokeDeviceShareLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *RevokeDeviceShareLogic) RevokeDeviceShare(req *types.DeviceShareRevokeReq) error {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID <= 0 {
		return errorx.NewCodeError(errorx.CodeTokenInvalid, "")
	}
	return devicesharesvc.New(l.svcCtx).RevokeShare(l.ctx, userID, req.ShareId)
}
