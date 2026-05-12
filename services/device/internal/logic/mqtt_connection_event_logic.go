package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type MqttConnectionEventLogic struct {
	svcCtx *svc.ServiceContext
}

func NewMqttConnectionEventLogic(svcCtx *svc.ServiceContext) *MqttConnectionEventLogic {
	return &MqttConnectionEventLogic{svcCtx: svcCtx}
}

type MqttConnectionEventReq struct {
	Action         string `json:"action"`          // "client.connected" 或 "client.disconnected"
	ClientID       string `json:"clientid"`        // 设备SN
	Username       string `json:"username"`        // 设备SN
	ConnectedAt    string `json:"connected_at"`    // 连接时间 (ISO8601)
	DisconnectedAt string `json:"disconnected_at"` // 断开时间 (ISO8601)
	Reason         string `json:"reason"`          // 断开原因
}

func (l *MqttConnectionEventLogic) HandleEvent(req *MqttConnectionEventReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sn := strings.ToUpper(strings.TrimSpace(req.ClientID))
	if sn == "" {
		return fmt.Errorf("clientid不能为空")
	}

	switch req.Action {
	case "client.connected":
		return l.handleConnected(ctx, sn, req)
	case "client.disconnected":
		return l.handleDisconnected(ctx, sn, req)
	default:
		return fmt.Errorf("未知的事件类型: %s", req.Action)
	}
}

func (l *MqttConnectionEventLogic) handleConnected(ctx context.Context, sn string, req *MqttConnectionEventReq) error {
	logx.Infof("设备MQTT连接: sn=%s, connected_at=%s", sn, req.ConnectedAt)

	device, err := l.svcCtx.DeviceRepo.FindBySn(ctx, sn)
	if err != nil {
		return fmt.Errorf("查询设备失败: %v", err)
	}

	if device == nil {
		return fmt.Errorf("设备不存在: %s", sn)
	}

	if device.Status != model.DeviceStatusNormal {
		return fmt.Errorf("设备状态异常(非正常状态), status=%d", device.Status)
	}

	err = l.svcCtx.DeviceRegister.UpdateOnlineStatus(ctx, sn)
	if err != nil {
		return fmt.Errorf("更新在线状态失败: %v", err)
	}

	if l.svcCtx.Redis != nil {
		cacheKey := fmt.Sprintf("device:online:%s", sn)
		l.svcCtx.Redis.Set(ctx, cacheKey, "1", 10*time.Minute)
	}

	err = l.syncShadowOnConnect(ctx, sn)
	if err != nil {
		logx.Slowf("同步设备影子失败(不影响主流程): sn=%s, error=%v", sn, err)
	}

	logx.Infof("设备上线成功: sn=%s, online_status=1", sn)

	return nil
}

func (l *MqttConnectionEventLogic) handleDisconnected(ctx context.Context, sn string, req *MqttConnectionEventReq) error {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "unknown"
	}

	logx.Infof("设备MQTT断开: sn=%s, reason=%s, disconnected_at=%s",
		sn, reason, req.DisconnectedAt)

	device, err := l.svcCtx.DeviceRepo.FindBySn(ctx, sn)
	if err != nil {
		return fmt.Errorf("查询设备失败: %v", err)
	}

	if device == nil {
		logx.Slowf("MQTT断开事件: 设备不存在，跳过处理 sn=%s", sn)
		return nil
	}

	if device.OnlineStatus == model.DeviceOnlineStatusOffline {
		logx.Slowf("MQTT断开事件: 设备已离线，跳过重复更新 sn=%s, reason=%s", sn, reason)
		return nil
	}

	now := time.Now()
	_, err = l.svcCtx.DB.ExecContext(ctx, `
		UPDATE device 
		SET online_status = $1, last_active_at = $2, updated_at = $3
		WHERE sn = $4
	`, model.DeviceOnlineStatusOffline, now, now, sn)

	if err != nil {
		return fmt.Errorf("更新离线状态失败: %v", err)
	}

	if l.svcCtx.Redis != nil {
		cacheKey := fmt.Sprintf("device:online:%s", sn)
		l.svcCtx.Redis.Del(ctx, cacheKey)
	}

	switch reason {
	case "keepalive_timeout":
		logx.Slowf("设备心跳超时断开: sn=%s, reason=keepalive_timeout, online_status=0 (需等待设备重连)", sn)
	case "normal":
		logx.Infof("设备正常断开: sn=%s, reason=normal, online_status=0", sn)
	case "kicked":
		logx.Slowf("设备被踢出: sn=%s, reason=kicked (可能账号冲突)", sn)
	default:
		logx.Slowf("设备异常断开: sn=%s, reason=%s, online_status=0", sn, reason)
	}

	err = l.syncShadowOnDisconnect(ctx, sn, reason)
	if err != nil {
		logx.Slowf("同步设备影子失败(不影响主流程): sn=%s, error=%v", sn, err)
	}

	return nil
}

func (l *MqttConnectionEventLogic) syncShadowOnConnect(ctx context.Context, sn string) error {
	if l.svcCtx.DeviceShadowRepo == nil {
		return nil
	}

	disconnectType := ""
	_, err := l.svcCtx.DeviceShadowRepo.UpdateOnlineStatus(ctx, sn, model.ShadowOnlineStatusOnline, disconnectType)
	if err != nil {
		return fmt.Errorf("更新影子在线状态失败: %v", err)
	}

	logx.Infof("设备影子已同步为在线: sn=%s", sn)

	return nil
}

func (l *MqttConnectionEventLogic) syncShadowOnDisconnect(ctx context.Context, sn string, reason string) error {
	if l.svcCtx.DeviceShadowRepo == nil {
		return nil
	}

	var disconnectType string
	switch reason {
	case "normal":
		disconnectType = model.DisconnectTypeNormal
	case "keepalive_timeout":
		disconnectType = model.DisconnectTypeKeepalive
	case "kicked":
		disconnectType = model.DisconnectTypeKicked
	default:
		disconnectType = model.DisconnectTypeUnknown
	}

	_, err := l.svcCtx.DeviceShadowRepo.UpdateOnlineStatus(ctx, sn, model.ShadowOnlineStatusOffline, disconnectType)
	if err != nil {
		return fmt.Errorf("更新影子离线状态失败: %v", err)
	}

	logx.Infof("设备影子已同步为离线: sn=%s, reason=%s", sn, disconnectType)

	return nil
}
