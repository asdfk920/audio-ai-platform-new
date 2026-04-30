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

type QuitDeviceShareLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQuitDeviceShareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QuitDeviceShareLogic {
	return &QuitDeviceShareLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *QuitDeviceShareLogic) QuitDeviceShare(req *types.DeviceShareQuitReq) error {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID <= 0 {
		return errorx.NewCodeError(errorx.CodeTokenInvalid, "")
	}
	return devicesharesvc.New(l.svcCtx).QuitShare(l.ctx, userID, req.ShareId)
}
