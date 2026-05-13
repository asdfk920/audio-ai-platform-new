// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListSentDeviceSharesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询我发出的共享
func NewListSentDeviceSharesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSentDeviceSharesLogic {
	return &ListSentDeviceSharesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListSentDeviceSharesLogic) ListSentDeviceShares() (resp *types.DeviceShareListResp, err error) {
	// todo: add your logic here and delete this line

	return
}
