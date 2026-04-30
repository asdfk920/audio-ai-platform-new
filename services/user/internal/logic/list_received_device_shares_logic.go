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

type ListReceivedDeviceSharesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListReceivedDeviceSharesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListReceivedDeviceSharesLogic {
	return &ListReceivedDeviceSharesLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ListReceivedDeviceSharesLogic) ListReceivedDeviceShares() (*types.DeviceShareListResp, error) {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "")
	}
	list, err := devicesharesvc.New(l.svcCtx).ListSharesForReceiver(l.ctx, userID)
	if err != nil {
		return nil, err
	}
	return toDeviceShareListResp(list), nil
}
