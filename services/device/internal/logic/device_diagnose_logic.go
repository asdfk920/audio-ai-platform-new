package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/shadowmqtt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeviceDiagnoseLogic 设备远程诊断逻辑
// 处理用户通过 App 发起设备远程诊断的业务逻辑
type DeviceDiagnoseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceDiagnoseLogic 创建设备远程诊断逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DeviceDiagnoseLogic: 设备远程诊断逻辑实例
func NewDeviceDiagnoseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceDiagnoseLogic {
	return &DeviceDiagnoseLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceDiagnose 设备远程诊断
// 流程：
//  1. 校验请求数据格式（SN、诊断类型等）
//  2. 查询设备是否存在
//  3. 验证用户绑定权限
//  4. 检查设备在线状态
//  5. 生成诊断任务唯一 ID
//  6. 构造 MQTT 诊断指令消息
//  7. 发布诊断指令到 MQTT Topic
//  8. 记录诊断任务到数据库
//  9. 返回诊断任务信息
//
// 参数 req *types.DeviceDiagnoseReq: 设备远程诊断请求
// 参数 userID int64: 用户 ID
// 返回 *types.DeviceDiagnoseResp: 设备远程诊断响应
// 返回 error: 诊断失败时的错误信息
func (l *DeviceDiagnoseLogic) DeviceDiagnose(req *types.DeviceDiagnoseReq, userID int64) (*types.DeviceDiagnoseResp, error) {
	// 1. 校验请求数据格式
	if err := validateDeviceDiagnoseReq(req); err != nil {
		return nil, fmt.Errorf("数据格式校验失败: %v", err)
	}

	sn := strings.ToUpper(strings.TrimSpace(req.Sn))

	// 2. 查询设备是否存在
	deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}
	if deviceInfo == nil {
		return nil, fmt.Errorf("设备未注册")
	}

	// 3. 验证用户绑定权限
	bindInfo, err := l.svcCtx.UserDeviceBindRepo.FindByUserIdAndDeviceId(l.ctx, userID, deviceInfo.ID)
	if err != nil || bindInfo == nil {
		return nil, fmt.Errorf("无权操作该设备")
	}

	// 4. 检查设备在线状态
	online, err := l.isDeviceOnline(sn)
	if err != nil {
		logx.Errorf("检查设备在线状态失败: sn=%s, err=%v", sn, err)
	}
	if !online {
		return nil, fmt.Errorf("设备离线，无法执行远程诊断")
	}

	// 5. 生成诊断任务唯一 ID
	now := time.Now()
	diagID := fmt.Sprintf("diag_%d_%s", now.UnixNano(), generateDiagRandomString(8))

	// 设置超时时间
	timeoutSec := req.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 300
	}

	// 6. 构造 MQTT 诊断指令消息
	diagCmd := map[string]interface{}{
		"cmd":       "diagnose",
		"diag_id":   diagID,
		"diag_type": req.DiagType,
		"timeout":   timeoutSec,
		"timestamp": now.Unix(),
	}

	payload, err := json.Marshal(diagCmd)
	if err != nil {
		return nil, fmt.Errorf("构造诊断指令失败: %v", err)
	}

	// 7. 发布诊断指令到 MQTT Topic
	if l.svcCtx.MQTTClient() != nil {
		if err := shadowmqtt.PublishDesiredCommand(l.svcCtx.Config, l.svcCtx.MQTTClient(), sn, deviceInfo.ID, payload); err != nil {
			logx.Errorf("发布诊断指令到 MQTT 失败: sn=%s, err=%v", sn, err)
		} else {
			logx.Infof("诊断指令已发布到 MQTT: sn=%s, diag_id=%s", sn, diagID)
		}
	}

	// 8. 记录诊断任务到数据库
	if err := l.insertDiagnosisRecord(sn, deviceInfo.ID, diagID, req.DiagType, timeoutSec, userID); err != nil {
		logx.Errorf("记录诊断任务失败: sn=%s, diag_id=%s, err=%v", sn, diagID, err)
	}

	logx.Infof("设备远程诊断任务已创建: sn=%s, diag_id=%s, diag_type=%s, user_id=%d",
		sn, diagID, req.DiagType, userID)

	// 9. 返回诊断任务信息
	return &types.DeviceDiagnoseResp{
		DiagID:      diagID,
		Status:      "running",
		DiagType:    req.DiagType,
		HealthScore: 0,
		Summary:     "诊断任务已下发，等待设备响应",
		CreatedAt:   now.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// isDeviceOnline 检查设备是否在线
// 优先查询 Redis，降级查询 MySQL
func (l *DeviceDiagnoseLogic) isDeviceOnline(sn string) (bool, error) {
	// 优先查询 Redis
	if l.svcCtx.Redis != nil {
		key := fmt.Sprintf("device:online:%s", sn)
		val, err := l.svcCtx.Redis.Get(l.ctx, key).Result()
		if err == nil {
			return val == "online" || val == "1", nil
		}
	}

	// 降级查询 MySQL
	if l.svcCtx.DB != nil {
		query := `SELECT online_status FROM device WHERE sn = $1 LIMIT 1`
		var onlineStatus int16
		err := l.svcCtx.DB.QueryRowContext(l.ctx, query, sn).Scan(&onlineStatus)
		if err == nil {
			return onlineStatus == 1, nil
		}
	}

	return false, nil
}

// insertDiagnosisRecord 将诊断任务记录到数据库
func (l *DeviceDiagnoseLogic) insertDiagnosisRecord(sn string, deviceID int64, diagID string, diagType string, timeoutSec int, userID int64) error {
	if l.svcCtx.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}

	query := `
		INSERT INTO device_diagnosis (
			device_id, sn, diag_type, status, params, 
			timeout_seconds, operator, created_at, updated_at
		) VALUES (
			$1, $2, $3, 1, $4, 
			$5, $6, $7, $7
		)
	`

	paramsJSON := fmt.Sprintf(`{"diag_id":"%s"}`, diagID)
	now := time.Now()

	_, err := l.svcCtx.DB.ExecContext(l.ctx, query,
		deviceID,   // $1: device_id
		sn,         // $2: sn
		diagType,   // $3: diag_type
		paramsJSON, // $4: params
		timeoutSec, // $5: timeout_seconds
		userID,     // $6: operator (使用用户 ID)
		now,        // $7: created_at/updated_at
	)
	if err != nil {
		return fmt.Errorf("写入诊断记录失败: %v", err)
	}

	return nil
}

// validateDeviceDiagnoseReq 校验设备远程诊断请求数据格式
// 校验规则：
//   - SN: 16 位字母数字，正则 ^[A-Z0-9]{16}$，不区分大小写
//   - DiagType: 仅支持 full、quick、network、audio
//   - TimeoutSec: 如果提供，必须大于 0
//
// 参数 req *types.DeviceDiagnoseReq: 设备远程诊断请求
// 返回 error: 校验失败时的错误信息
func validateDeviceDiagnoseReq(req *types.DeviceDiagnoseReq) error {
	snRegex := regexp.MustCompile(`(?i)^[A-Z0-9]{16}$`)
	if !snRegex.MatchString(req.Sn) {
		return fmt.Errorf("SN 格式错误，必须为 16 位字母数字组合")
	}

	// 校验 diag_type
	validDiagTypes := map[string]bool{
		"full":    true,
		"quick":   true,
		"network": true,
		"audio":   true,
	}
	diagType := strings.ToLower(strings.TrimSpace(req.DiagType))
	if !validDiagTypes[diagType] {
		return fmt.Errorf("diag_type 无效，仅支持 full、quick、network、audio")
	}

	// 校验 timeout_sec
	if req.TimeoutSec < 0 {
		return fmt.Errorf("timeout_sec 不能为负数")
	}

	return nil
}

// generateDiagRandomString 生成指定长度的随机字符串
// 用于生成诊断 ID 中的随机部分
func generateDiagRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
