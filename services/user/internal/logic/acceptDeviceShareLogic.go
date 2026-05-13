// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AcceptDeviceShareLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 接受设备共享邀请
func NewAcceptDeviceShareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AcceptDeviceShareLogic {
	return &AcceptDeviceShareLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AcceptDeviceShareLogic) AcceptDeviceShare(req *types.DeviceShareAcceptReq) (resp *types.DeviceShareItem, err error) {
	// todo: add your logic here and delete this line

	return
}
