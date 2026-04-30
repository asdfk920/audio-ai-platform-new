package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/user/internal/devicesharesvc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeviceShareExpireTask struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeviceShareExpireTask(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceShareExpireTask {
	return &DeviceShareExpireTask{ctx: ctx, svcCtx: svcCtx}
}

func (t *DeviceShareExpireTask) Execute(limit int) (int, error) {
	count, err := devicesharesvc.New(t.svcCtx).ExpireShares(t.ctx, limit)
	if err != nil {
		logx.WithContext(t.ctx).Errorf("expire device shares failed: %v", err)
		return 0, err
	}
	return count, nil
}
