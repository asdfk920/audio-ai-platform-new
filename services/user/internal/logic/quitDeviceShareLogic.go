// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type QuitDeviceShareLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 退出设备共享
func NewQuitDeviceShareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QuitDeviceShareLogic {
	return &QuitDeviceShareLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QuitDeviceShareLogic) QuitDeviceShare(req *types.DeviceShareQuitReq) error {
	// todo: add your logic here and delete this line

	return nil
}
