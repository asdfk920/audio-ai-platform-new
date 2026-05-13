// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDeviceShareDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询共享详情
func NewGetDeviceShareDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDeviceShareDetailLogic {
	return &GetDeviceShareDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDeviceShareDetailLogic) GetDeviceShareDetail(req *types.DeviceShareDetailReq) (resp *types.DeviceShareItem, err error) {
	// todo: add your logic here and delete this line

	return
}
