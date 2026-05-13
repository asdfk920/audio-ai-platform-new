package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

type WillMessageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewWillMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WillMessageLogic {
	return &WillMessageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

type WillMessageReq struct {
	SN     string `json:"sn"`
	Status string `json:"status"`
	Time   string `json:"time"`
	Reason string `json:"reason,omitempty"`
}

func (l *WillMessageLogic) ProcessWillMessage(req *WillMessageReq) error {
	sn := strings.TrimSpace(req.SN)
	if sn == "" {
		l.Errorf("WILL消息缺少设备SN")
		return fmt.Errorf("WILL消息缺少设备SN")
	}

	status := strings.TrimSpace(req.Status)
	if status != "offline" {
		l.Slowf("收到非离线WILL消息，跳过处理: sn=%s, status=%s", sn, status)
		return nil
	}

	disconnectTime := req.Time
	if disconnectTime == "" {
		disconnectTime = time.Now().Format(time.RFC3339)
	}

	reason := model.DisconnectTypeWill
	if req.Reason != "" {
		reason = req.Reason
	}

	l.Infof("收到设备WILL离线消息: sn=%s, time=%s, reason=%s", sn, disconnectTime, reason)

	device, err := l.svcCtx.DeviceRepo.FindBySn(l.ctx, sn)
	if err != nil {
		l.Errorf("查询设备失败: sn=%s, error=%v", sn, err)
		return fmt.Errorf("查询设备失败: %v", err)
	}

	if device == nil {
		l.Slowf("WILL消息: 设备不存在，跳过处理 sn=%s", sn)
		return nil
	}

	if device.OnlineStatus == model.DeviceOnlineStatusOffline {
		l.Slowf("WILL消息: 设备已离线，跳过重复更新 sn=%s", sn)
		return nil
	}

	err = l.updateDeviceMainTable(sn)
	if err != nil {
		l.Errorf("更新设备主表失败: sn=%s, error=%v", sn, err)
		return fmt.Errorf("更新设备主表离线状态失败: %v", err)
	}

	l.clearRedisCache(sn)

	err = l.syncDeviceShadowSafe(sn, model.ShadowOnlineStatusOffline, reason)
	if err != nil {
		l.Slowf("同步设备影子失败(不影响主流程): sn=%s, error=%v", sn, err)
	} else {
		l.Infof("设备影子已同步为离线: sn=%s, reason=%s", sn, reason)
	}

	l.Slowf("设备因WILL消息触发离线: sn=%s, reason=%s, time=%s (异常断开，需排查)", sn, reason, disconnectTime)

	return nil
}

func (l *WillMessageLogic) updateDeviceMainTable(sn string) error {
	now := time.Now()

	query := `
		UPDATE device 
		SET online_status = $1, last_active_at = $2, updated_at = $3
		WHERE sn = $4
	`

	result, err := l.svcCtx.DB.ExecContext(l.ctx, query,
		model.DeviceOnlineStatusOffline, now, now, sn)

	if err != nil {
		return fmt.Errorf("执行UPDATE语句失败: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %v", err)
	}

	if rowsAffected == 0 {
		l.Slowf("更新设备主表: 未找到匹配的记录 sn=%s", sn)
	} else {
		l.Infof("更新设备主表成功: sn=%s, online_status=0, affected=%d", sn, rowsAffected)
	}

	return nil
}

func (l *WillMessageLogic) clearRedisCache(sn string) {
	if l.svcCtx.Redis == nil {
		return
	}

	cacheKey := fmt.Sprintf("device:online:%s", sn)
	err := l.svcCtx.Redis.Del(l.ctx, cacheKey).Err()
	if err != nil {
		l.Slowf("清除Redis缓存失败(不影响主流程): sn=%s, error=%v", sn, err)
	} else {
		l.Infof("Redis缓存已清除: key=%s", cacheKey)
	}
}

func (l *WillMessageLogic) syncDeviceShadowSafe(sn string, onlineStatus int16, disconnectType string) error {
	defer func() {
		if r := recover(); r != nil {
			l.Errorf("syncDeviceShadowSafe 发生panic: sn=%s, recovered=%v", sn, r)
		}
	}()

	if l.svcCtx.DeviceShadowRepo == nil {
		l.Slowf("DeviceShadowRepo未初始化，跳过影子同步")
		return nil
	}

	shadow, err := l.svcCtx.DeviceShadowRepo.FindBySn(l.ctx, sn)
	if err != nil {
		return fmt.Errorf("查询设备影子失败: %v", err)
	}

	if shadow == nil {
		l.Infof("设备影子不存在，创建新记录: sn=%s", sn)
		return l.createNewShadowRecord(sn, onlineStatus, disconnectType)
	}

	if shadow.OnlineStatus == onlineStatus && shadow.DisconnectType == disconnectType {
		l.Slowf("设备影子状态已是目标状态，跳过更新: sn=%s, online_status=%d", sn, onlineStatus)
		return nil
	}

	l.Infof("更新现有设备影子: sn=%s, old_status=%d -> new_status=%d",
		sn, shadow.OnlineStatus, onlineStatus)

	newVersion, err := l.svcCtx.DeviceShadowRepo.UpdateOnlineStatus(l.ctx, sn, onlineStatus, disconnectType)
	if err != nil {
		return fmt.Errorf("更新设备影子在线状态失败: %v", err)
	}

	l.Infof("设备影子更新成功: sn=%s, version=%d", sn, newVersion)
	return nil
}

func (l *WillMessageLogic) createNewShadowRecord(sn string, onlineStatus int16, disconnectType string) error {
	shadow := &model.DeviceShadow{
		Sn:             sn,
		OnlineStatus:   onlineStatus,
		DisconnectType: disconnectType,
	}
	now := time.Now()
	shadow.DisconnectAt = &now

	err := l.svcCtx.DeviceShadowRepo.CreateOrUpdate(l.ctx, shadow)
	if err != nil {
		return fmt.Errorf("创建设备影子失败: %v", err)
	}

	l.Infof("新设备影子记录创建成功: sn=%s, id=%d", sn, shadow.ID)
	return nil
}
