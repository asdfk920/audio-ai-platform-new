package logic

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// DeviceRebootLogic 设备重启指令逻辑
// 处理用户通过 App 向设备下发重启指令的业务逻辑
type DeviceRebootLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceRebootLogic 创建设备重启指令逻辑实例
func NewDeviceRebootLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceRebootLogic {
	return &DeviceRebootLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceReboot 下发设备重启指令（WebSocket方案）
//
// 完整流程：
//  1. 校验请求数据格式（SN、Action）
//  2. 查询设备是否存在并获取设备ID
//  3. 检查设备在线状态
//     - 在线：直接通过 WebSocket 下发重启指令
//     - 离线：缓存指令，待设备上线后补发
//  4. 构造标准化的重启指令消息（JSON格式）
//  5. 通过 WebSocket 连接发送给设备
//  6. 更新设备状态为重启中
//  7. 记录指令日志，返回执行结果
//
// 参数 req *types.DeviceRebootReq: 设备重启指令请求
// 返回 *types.DeviceRebootResp: 设备重启指令响应
// 返回 error: 下发失败时的错误信息
func (l *DeviceRebootLogic) DeviceReboot(req *types.DeviceRebootReq) (*types.DeviceRebootResp, error) {
	// 1. 校验请求数据格式
	if err := validateDeviceRebootReq(req); err != nil {
		return nil, fmt.Errorf("数据格式校验失败: %v", err)
	}

	sn := strings.ToUpper(strings.TrimSpace(req.Sn))

	// 2. 查询设备是否存在并获取设备信息
	deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}
	if deviceInfo == nil {
		return nil, fmt.Errorf("设备不存在: %s", sn)
	}

	deviceId := fmt.Sprintf("%d", deviceInfo.ID)

	logx.Infof("[REBOOT] 开始处理设备重启请求: sn=%s, device_id=%s", sn, deviceId)

	// 3. 检查设备是否正在重启中（幂等性控制，避免重复下发）
	if l.isRebooting(sn) {
		return nil, fmt.Errorf("设备正在重启中，请勿重复操作")
	}

	// 4. 检查设备在线状态
	isOnline := IsDeviceOnline(deviceId)

	if !isOnline {
		// 设备离线：缓存指令或提示用户
		logx.Slowf("[REBOOT] 设备当前离线: sn=%s, device_id=%s", sn, deviceId)

		// 方案A：缓存指令（推荐）
		cacheErr := l.cacheRebootInstruction(sn, deviceId, deviceInfo.ID)
		if cacheErr != nil {
			return nil, fmt.Errorf("设备离线且指令缓存失败: %v", cacheErr)
		}

		return &types.DeviceRebootResp{
			InstructionID: 0,
			Status:        "cached",
			Message:       "设备当前离线，重启指令已缓存，将在设备重新上线后自动下发",
		}, nil

		// 方案B：直接拒绝（可选）
		// return nil, fmt.Errorf("设备当前离线，无法下发重启指令")
	}

	// 5. 设备在线：构造标准化的重启指令消息
	requestId := uuid.New().String()
	rebootCmd := map[string]interface{}{
		"type":       "cmd",
		"cmd":        "reboot",
		"sn":         sn,
		"request_id": requestId,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	logx.Infof("[REBOOT] 准备发送重启指令: sn=%s, request_id=%s", sn, requestId)

	// 6. 通过 WebSocket 发送重启指令给设备（先验证连接→再发送）
	sendErr := l.sendInstructionWithConnectionCheck(deviceId, sn, rebootCmd)
	if sendErr != nil {
		logx.Errorf("[REBOOT] ❌ 指令发送失败，将缓存指令: sn=%s, error=%v", sn, sendErr)

		cacheErr := l.cacheRebootInstruction(sn, deviceId, deviceInfo.ID)
		if cacheErr != nil {
			return nil, fmt.Errorf("指令发送失败且缓存也失败: %v (缓存错误: %v)", sendErr, cacheErr)
		}

		return &types.DeviceRebootResp{
			InstructionID: 0,
			Status:        "cached",
			Message:       fmt.Sprintf("设备当前无法连接，重启指令已缓存，将在设备上线后自动下发。原因: %v", sendErr),
		}, nil
	}

	logx.Infof("[REBOOT] ✅ 重启指令发送成功: sn=%s, request_id=%s", sn, requestId)

	// 7. 更新设备状态为重启中
	updateErr := l.updateDeviceStatusToRebooting(deviceInfo.ID, sn)
	if updateErr != nil {
		logx.Slowf("[REBOOT] 更新设备状态为重启中失败(不影响指令): sn=%s, error=%v", sn, updateErr)
	} else {
		logx.Infof("[REBOOT] 设备状态已更新为重启中: sn=%s", sn)
	}

	// 8. 记录指令到数据库（可选，用于追踪和审计）
	instructionId, logErr := l.logRebootInstruction(deviceInfo.ID, sn, requestId, "delivered")
	if logErr != nil {
		logx.Slowf("[REBOOT] 记录指令日志失败(不影响结果): sn=%s, error=%v", sn, logErr)
	}

	// 9. 返回成功响应
	return &types.DeviceRebootResp{
		InstructionID: instructionId,
		Status:        "delivered",
		Message:       "重启指令已下发，设备即将重启...",
	}, nil
}

// validateDeviceRebootReq 校验设备重启指令请求数据格式
// 校验规则：
//   - SN: 16 位字母数字，正则 ^[A-Z0-9]{16}$，不区分大小写
//   - Action: 必须为 "reboot"
//
// 参数 req *types.DeviceRebootReq: 设备重启指令请求
// 返回 error: 校验失败时的错误信息
func validateDeviceRebootReq(req *types.DeviceRebootReq) error {
	snRegex := regexp.MustCompile(`(?i)^[A-Z0-9]{16}$`)
	if !snRegex.MatchString(req.Sn) {
		return fmt.Errorf("SN 格式错误，必须为 16 位字母数字组合")
	}

	if req.Action != "reboot" {
		return fmt.Errorf("操作类型错误，必须为 reboot")
	}

	return nil
}

// isRebooting 检查设备是否正在重启中
// 幂等性控制：防止重复下发导致多次重启
// 参数 sn string: 设备序列号
// 返回 bool: 设备是否正在重启中
func (l *DeviceRebootLogic) isRebooting(sn string) bool {
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return false
	}

	var count int
	err := l.svcCtx.DB.QueryRowContext(l.ctx, `
		SELECT COUNT(1)
		FROM public.device_instruction
		WHERE sn = $1
		  AND command_code = 'reboot'
		  AND status IN (1, 2)
		  AND created_at >= NOW() - INTERVAL '5 minutes'
	`, sn).Scan(&count)
	if err != nil {
		return false
	}

	return count > 0
}

// cacheRebootInstruction 缓存重启指令（设备离线时使用）
// 将指令保存到数据库，待设备上线后自动补发
// 参数 sn string: 设备序列号
// 参数 deviceId string: 设备ID字符串
// 参数 deviceIDInt int64: 设备ID整数
// 返回 error: 缓存失败时的错误信息
func (l *DeviceRebootLogic) cacheRebootInstruction(sn string, deviceId string, deviceIDInt int64) error {
	query := `
		INSERT INTO public.device_instruction (
			device_id, sn, command_code, cmd, status,
			params, created_at, expires_at
		) VALUES ($1, $2, 'reboot', 'reboot', 0,
			$3::jsonb, NOW(), NOW() + INTERVAL '24 hours')
		RETURNING id
	`

	params := map[string]interface{}{
		"request_id": uuid.New().String(),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"cached":     true,
		"reason":     "设备离线时缓存",
	}

	var instructionId int64
	err := l.svcCtx.DB.QueryRowContext(l.ctx, query, deviceIDInt, sn, params).Scan(&instructionId)
	if err != nil {
		return fmt.Errorf("插入缓存指令失败: %v", err)
	}

	logx.Infof("[REBOOT] 重启指令已缓存: sn=%s, instruction_id=%d", sn, instructionId)
	return nil
}

// updateDeviceStatusToRebooting 更新设备状态为重启中
// 参数 deviceID int64: 设备ID
// 参数 sn string: 设备序列号
// 返回 error: 更新失败时的错误信息
func (l *DeviceRebootLogic) updateDeviceStatusToRebooting(deviceID int64, sn string) error {
	query := `
		UPDATE public.device
		SET online_status = 2,
		    updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
	`

	result, err := l.svcCtx.DB.ExecContext(l.ctx, query, deviceID)
	if err != nil {
		return fmt.Errorf("更新设备状态失败: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("设备不存在或已被删除")
	}

	return nil
}

// logRebootInstruction 记录重启指令日志
// 用于审计和追踪指令执行情况
// 参数 deviceID int64: 设备ID
// 参数 sn string: 设备序列号
// 参数 requestId string: 请求唯一标识
// 参数 status string: 指令状态
// 返回 int64: 指令记录ID
// 返回 error: 记录失败时的错误信息
func (l *DeviceRebootLogic) logRebootInstruction(deviceID int64, sn string, requestId string, status string) (int64, error) {
	query := `
		INSERT INTO public.device_instruction (
			device_id, sn, command_code, cmd, status,
			params, created_at
		) VALUES ($1, $2, 'reboot', 'reboot', CASE WHEN $4='delivered' THEN 1 ELSE 0 END,
			$3::jsonb, NOW())
		RETURNING id
	`

	params := map[string]interface{}{
		"request_id": requestId,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"status":     status,
	}

	var instructionId int64
	err := l.svcCtx.DB.QueryRowContext(l.ctx, query, deviceID, sn, params, status).Scan(&instructionId)
	if err != nil {
		return 0, fmt.Errorf("记录指令日志失败: %v", err)
	}

	return instructionId, nil
}

// HandleDeviceRebootResponse 处理设备反馈的重启响应
// 设备端在重启前或重启后通过 WebSocket 反馈执行结果
// 参数 deviceId string: 设备ID
// 参数 response map[string]interface{}: 设备反馈的消息体
func HandleDeviceRebootResponse(deviceId string, response map[string]interface{}) {
	sn, _ := response["sn"].(string)
	requestId, _ := response["request_id"].(string)
	status, _ := response["status"].(string)
	message, _ := response["message"].(string)

	logx.Infof("[REBOOT] 收到设备反馈: device_id=%s, sn=%s, request_id=%s, status=%s, message=%s",
		deviceId, sn, requestId, status, message)

	switch status {
	case "success":
		logx.Infof("[REBOOT] ✅ 设备确认即将重启: sn=%s, request_id=%s", sn, requestId)
	case "failed":
		logx.Errorf("[REBOOT] ❌ 设备重启失败: sn=%s, request_id=%s, reason=%s", sn, requestId, message)
	default:
		logx.Infof("[REBOOT] 设备反馈未知状态: sn=%s, status=%s", sn, status)
	}
}

// sendInstructionWithConnectionCheck 通过WebSocket发送指令（带连接验证）
// 完整流程：
//  1. 检查设备是否在连接池中（在线状态）
//  2. 尝试发送指令
//  3. 发送失败时自动重试1次
//  4. 返回发送结果或错误
//
// 参数 deviceId string: 设备ID字符串
// 参数 sn string: 设备序列号
// 参数 cmd interface{}: 要发送的指令消息
// 返回 error: 发送失败时的错误信息
func (l *DeviceRebootLogic) sendInstructionWithConnectionCheck(deviceId string, sn string, cmd interface{}) error {
	// ① 检查设备是否在线（从WebSocket连接池中查找）
	if !IsDeviceOnline(deviceId) {
		return fmt.Errorf("设备未建立WebSocket连接（离线）")
	}

	logx.Infof("[WS-SEND] 设备在线，准备通过WebSocket发送指令: device_id=%s, sn=%s", deviceId, sn)

	// ② 第一次尝试发送
	sendErr := SendCmdToDevice(deviceId, cmd)
	if sendErr == nil {
		logx.Infof("[WS-SEND] ✅ 指令发送成功（首次）: device_id=%s", deviceId)
		return nil
	}

	// ③ 首次失败，等待后重试
	logx.Slowf("[WS-SEND] ⚠️ 首次发送失败，500ms后重试: device_id=%s, error=%v", deviceId, sendErr)

	time.Sleep(500 * time.Millisecond)

	retryErr := SendCmdToDevice(deviceId, cmd)
	if retryErr != nil {
		logx.Errorf("[WS-SEND] ❌ 重试发送仍然失败: device_id=%s, error=%v", deviceId, retryErr)
		return fmt.Errorf("WebSocket发送失败（已重试1次）: %v", retryErr)
	}

	logx.Infof("[WS-SEND] ✅ 指令发送成功（重试后）: device_id=%s", deviceId)
	return nil
}
