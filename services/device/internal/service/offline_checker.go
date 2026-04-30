package service

import (
	"context"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

type OfflineChecker struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logger logx.Logger
}

func NewOfflineChecker(ctx context.Context, svcCtx *svc.ServiceContext) *OfflineChecker {
	return &OfflineChecker{
		ctx:    ctx,
		svcCtx: svcCtx,
		logger: logx.WithContext(ctx),
	}
}

// Start 启动离线检测服务
func (c *OfflineChecker) Start() {
	// 每5分钟执行一次离线检测
	threading.GoSafe(func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-c.ctx.Done():
				c.logger.Info("离线检测服务已停止")
				return
			case <-ticker.C:
				c.checkOfflineDevices()
			}
		}
	})
}

// checkOfflineDevices 检测并更新离线设备
func (c *OfflineChecker) checkOfflineDevices() {
	// 检测超过3分钟没有心跳的设备
	const timeoutMinutes = 3
	
	rowsAffected, err := c.svcCtx.DeviceRepo.UpdateOfflineDevices(c.ctx, timeoutMinutes)
	if err != nil {
		c.logger.Errorf("离线设备检测失败: %v", err)
		return
	}

	if rowsAffected > 0 {
		c.logger.Infof("离线设备检测完成，更新了 %d 台设备为离线状态", rowsAffected)
	} else {
		c.logger.Debug("离线设备检测完成，没有需要更新的设备")
	}
}

// CheckOfflineDevicesOnce 单次执行离线检测（用于手动触发）
func (c *OfflineChecker) CheckOfflineDevicesOnce() (int64, error) {
	const timeoutMinutes = 3
	return c.svcCtx.DeviceRepo.UpdateOfflineDevices(c.ctx, timeoutMinutes)
}