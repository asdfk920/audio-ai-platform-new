package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
	"github.com/jacklau/audio-ai-platform/services/device/internal/repo"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

// MqttAuthLogic MQTT 设备认证逻辑
// 处理 EMQX Broker 转发的设备连接认证请求
type MqttAuthLogic struct {
	svcCtx *svc.ServiceContext
}

// NewMqttAuthLogic 创建 MQTT 设备认证逻辑实例
func NewMqttAuthLogic(svcCtx *svc.ServiceContext) *MqttAuthLogic {
	return &MqttAuthLogic{
		svcCtx: svcCtx,
	}
}

// Authenticate 执行设备认证
//
// 参数：
//   - clientID: 设备客户端ID（应等于设备SN）
//   - username: 用户名（应等于设备SN）
//   - password: 密码（应等于设备专属密钥）
//
// 返回值：
//   - bool: true=允许接入，false=拒绝接入
//   - string: 拒绝原因说明（成功时为空字符串）
func (l *MqttAuthLogic) Authenticate(clientID, username, password string) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sn := strings.ToUpper(strings.TrimSpace(clientID))
	usernameNorm := strings.ToUpper(strings.TrimSpace(username))

	logx.Infof("开始MQTT认证：sn=%s, username=%s", sn, usernameNorm)

	if sn == "" {
		return false, "客户端标识不能为空"
	}

	if sn != usernameNorm {
		logx.Slowf("MQTT认证失败：clientid与username不匹配，sn=%s, username=%s", sn, usernameNorm)
		return false, "客户端标识与用户名不匹配"
	}

	deviceInfo, err := repo.GetDeviceBySN(ctx, l.svcCtx.DB, sn)
	if err != nil {
		logx.Errorf("MQTT认证：查询设备失败，sn=%s, err=%v", sn, err)
		return false, "设备不存在或数据库查询失败"
	}

	if deviceInfo == nil {
		logx.Slowf("MQTT认证失败：设备不存在，sn=%s", sn)
		return false, "设备未注册"
	}

	if deviceInfo.Secret == "" {
		logx.Errorf("MQTT认证失败：设备密钥为空，sn=%s", sn)
		return false, "设备密钥未配置，请联系管理员"
	}

	passwordErr := bcrypt.CompareHashAndPassword([]byte(deviceInfo.Secret), []byte(password))
	if passwordErr != nil {
		logx.Slowf("MQTT认证失败：密钥不匹配，sn=%s", sn)
		l.recordAuthFailure(ctx, sn, "密钥错误")
		return false, "设备密钥错误"
	}

	switch deviceInfo.Status {
	case model.DeviceStatusDisabled:
		logx.Slowf("MQTT认证失败：设备已被禁用，sn=%s", sn)
		return false, "设备已被管理员禁用"
	case model.DeviceStatusScrapped:
		logx.Slowf("MQTT认证失败：设备已报废，sn=%s", sn)
		return false, "设备已报废注销"
	case model.DeviceStatusNormal:
	default:
		logx.Errorf("MQTT认证失败：未知设备状态，sn=%s, status=%d", sn, deviceInfo.Status)
		return false, "设备状态异常"
	}

	if err := l.updateDeviceOnlineStatus(ctx, sn); err != nil {
		logx.Errorf("更新设备在线状态失败（不影响认证结果）：sn=%s, err=%v", sn, err)
	}

	l.recordAuthSuccess(ctx, sn)

	logx.Infof("MQTT认证成功：sn=%s, device_id=%d", sn, deviceInfo.ID)

	return true, ""
}

func (l *MqttAuthLogic) updateDeviceOnlineStatus(ctx context.Context, sn string) error {
	now := time.Now()
	_, err := l.svcCtx.DB.ExecContext(ctx, `
		UPDATE device 
		SET online_status = $1, last_active_at = $2, updated_at = $3
		WHERE sn = $4`,
		model.DeviceOnlineStatusOnline,
		now,
		now,
		sn,
	)
	return err
}

func (l *MqttAuthLogic) recordAuthSuccess(ctx context.Context, sn string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logx.Errorf("记录认证成功日志异常：%v", r)
			}
		}()

		recordCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := l.svcCtx.DB.ExecContext(recordCtx, `
			INSERT INTO device_auth_log (device_sn, auth_result, client_ip, created_at)
			VALUES ($1, 'success', '', NOW())
		`, sn)
		if err != nil {
			logx.Debugf("记录认证日志失败（可忽略）：%v", err)
		}
	}()
}

func (l *MqttAuthLogic) recordAuthFailure(ctx context.Context, sn, reason string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logx.Errorf("记录认证失败日志异常：%v", r)
			}
		}()

		recordCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := l.svcCtx.DB.ExecContext(recordCtx, `
			INSERT INTO device_auth_log (device_sn, auth_result, error_msg, created_at)
			VALUES ($1, 'failed', $2, NOW())
		`, sn, fmt.Sprintf("%.50s", reason))
		if err != nil {
			logx.Debugf("记录认证日志失败（可忽略）：%v", err)
		}
	}()
}

// HandleDisconnect 处理设备断开连接事件
// 更新设备状态为离线，记录断开时间
//
// 参数：
//   - clientID: 设备客户端ID（= 设备SN）
//
// 返回值：
//   - error: 更新失败时的错误信息
func (l *MqttAuthLogic) HandleDisconnect(clientID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sn := strings.ToUpper(strings.TrimSpace(clientID))
	if sn == "" {
		return fmt.Errorf("客户端标识不能为空")
	}

	now := time.Now()
	result, err := l.svcCtx.DB.ExecContext(ctx, `
		UPDATE device
		SET online_status = $1, last_active_at = $2, updated_at = $3
		WHERE sn = $4 AND online_status = $5
	`,
		model.DeviceOnlineStatusOffline,
		now,
		now,
		sn,
		model.DeviceOnlineStatusOnline,
	)

	if err != nil {
		return fmt.Errorf("更新设备离线状态失败：%w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		logx.Infof("设备离线更新：设备已离线或不存在，sn=%s", sn)
		return nil
	}

	logx.Infof("设备已标记为离线：sn=%s", sn)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logx.Errorf("记录断开事件异常：%v", r)
			}
		}()

		recordCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, _ = l.svcCtx.DB.ExecContext(recordCtx, `
			INSERT INTO device_auth_log (device_sn, auth_result, error_msg, created_at)
			VALUES ($1, 'disconnected', 'MQTT连接断开', NOW())
		`, sn)
	}()

	return nil
}
