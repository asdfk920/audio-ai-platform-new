// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RevokeDeviceShareLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 撤销设备共享
func NewRevokeDeviceShareLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeDeviceShareLogic {
	return &RevokeDeviceShareLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RevokeDeviceShareLogic) RevokeDeviceShare(req *types.DeviceShareRevokeReq) error {
	// todo: add your logic here and delete this line

	return nil
}
