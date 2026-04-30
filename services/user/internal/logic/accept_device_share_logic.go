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

type AcceptDeviceShareLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAcceptDeviceShareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AcceptDeviceShareLogic {
	return &AcceptDeviceShareLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *AcceptDeviceShareLogic) AcceptDeviceShare(req *types.DeviceShareAcceptReq) (*types.DeviceShareItem, error) {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "")
	}
	view, err := devicesharesvc.New(l.svcCtx).AcceptShareInvite(l.ctx, userID, req.InviteCode)
	if err != nil {
		return nil, err
	}
	return toDeviceShareItem(view), nil
}
