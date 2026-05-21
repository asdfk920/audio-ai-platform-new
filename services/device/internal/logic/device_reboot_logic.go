package logic

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
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

		// 🔴 关键：启动超时保护定时器
		// 如果设备在30秒内没有响应ACK，强制标记为离线并断开连接
		l.startRebootTimeoutProtection(deviceInfo.ID, sn, requestId, deviceId)
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
// updateDeviceStatusToRebooting 更新设备在线状态为"重启中"
// 在设备重启指令成功下发后调用
// 状态值定义（参见 migration 20260520_fix_device_online_status_constraint.sql）：
//
//	0 = 离线（offline）
//	1 = 在线（online）
//	2 = 重启中（rebooting）← 当前使用的值
//
// 参数 deviceID int64: 设备ID（整数）
// 参数 sn string: 设备序列号（用于日志）
// 返回 error: 更新失败时的错误信息
func (l *DeviceRebootLogic) updateDeviceStatusToRebooting(deviceID int64, sn string) error {
	query := `
		UPDATE public.device
		SET online_status = $2,
		    updated_at = NOW()
		WHERE id = $1
		  AND deleted_at IS NULL
	`

	result, err := l.svcCtx.DB.ExecContext(l.ctx, query, deviceID, DeviceOnlineStatusRebooting)
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

// HandleDeviceRebootResponse 处理设备反馈的重启响应（增强版）
// 设备端在重启前通过 WebSocket 反馈执行结果
//
// 完整流程：
//  1. 接收设备的reboot响应消息
//  2. 根据status判断执行结果
//  3. 如果success：更新状态为离线 + 断开WebSocket连接
//  4. 如果failed：恢复状态为在线（设备未重启）
//  5. 记录日志和审计信息
//
// 参数 wsLogic *DeviceWsLogic: WebSocket逻辑实例（用于访问数据库和连接池）
// 参数 deviceId string: 设备ID
// 参数 response map[string]interface{}: 设备反馈的消息体
func HandleDeviceRebootResponse(wsLogic *DeviceWsLogic, deviceId string, response map[string]interface{}) {
	sn, _ := response["sn"].(string)
	requestId, _ := response["request_id"].(string)
	status, _ := response["status"].(string)
	message, _ := response["message"].(string)

	logx.Infof("[REBOOT] 收到设备反馈: device_id=%s, sn=%s, request_id=%s, status=%s, message=%s",
		deviceId, sn, requestId, status, message)

	switch status {
	case "success":
		logx.Infof("[REBOOT] ✅ 设备确认即将重启: sn=%s, request_id=%s", sn, requestId)

		// 🔴 关键：设备确认重启后，立即将状态改为离线并断开连接
		if wsLogic != nil {
			wsLogic.handleDeviceRebootSuccess(deviceId, sn, requestId)
		}

	case "failed":
		logx.Errorf("[REBOOT] ❌ 设备重启失败: sn=%s, request_id=%s, reason=%s", sn, requestId, message)

		// 恢复设备在线状态（因为设备没有真正重启）
		if wsLogic != nil {
			wsLogic.handleDeviceRebootFailed(deviceId, sn, requestId, message)
		}

	default:
		logx.Infof("[REBOOT] 设备反馈未知状态: sn=%s, status=%s", sn, status)
	}
}

// handleDeviceRebootSuccess 处理设备重启成功后的清理工作
// 完成以下操作：
//  1. 将设备状态从"重启中(2)"更新为"离线(0)"
//  2. 从WebSocket连接池中移除设备
//  3. 关闭WebSocket连接（触发onClose事件）
//  4. 同步离线状态到Redis
//
// 参数 deviceId string: 设备ID字符串
// 参数 sn string: 设备序列号
// 参数 requestId string: 请求ID（用于审计）
func (l *DeviceWsLogic) handleDeviceRebootSuccess(deviceId string, sn string, requestId string) {
	logx.Infof("[REBOOT-POST] 开始处理设备重启成功后的清理工作...")
	logx.Infof("[REBOOT-POST]   设备ID: %s", deviceId)
	logx.Infof("[REBOOT-POST]   序列号: %s", sn)
	logx.Infof("[REBOOT-POST]   请求ID: %s", requestId)

	// ① 从数据库查询设备ID（int64类型）
	deviceID, err := l.getDeviceIDBySN(sn)
	if err != nil {
		logx.Errorf("[REBOOT-POST] ❌ 查询设备ID失败: sn=%s, error=%v", sn, err)
		return
	}

	// ② 更新设备状态为"离线"
	l.updateDeviceOfflineStatus(deviceID)
	logx.Infof("[REBOOT-POST] ✅ 设备状态已更新为离线: device_id=%d (重启中→离线)", deviceID)

	// ③ 从WebSocket连接池中移除并关闭连接
	l.forceDisconnectDevice(deviceId, deviceID, 4004, "Device rebooting")
	logx.Infof("[REBOOT-POST] ✅ WebSocket连接已主动关闭: device_id=%s", deviceId)

	// ④ 记录审计日志
	logx.Infof("[REBOOT-POST] 🎉 设备重启流程完成:")
	logx.Infof("            在线(1) → 重启中(2) → 离线(0)")
	logx.Infof("            时间: %s", time.Now().Format("2006-01-02 15:04:05"))
}

// handleDeviceRebootFailed 处理设备重启失败的恢复工作
// 当设备返回重启失败时，需要：
//  1. 将设备状态从"重启中(2)"恢复为"在线(1)"
//  2. 保持WebSocket连接不断开
//  3. 记录失败原因
//
// 参数 deviceId string: 设备ID字符串
// 参数 sn string: 设备序列号
// 参数 requestId string: 请求ID
// 参数 reason string: 失败原因
func (l *DeviceWsLogic) handleDeviceRebootFailed(deviceId string, sn string, requestId string, reason string) {
	logx.Infof("[REBOOT-FAIL] 开始处理设备重启失败后的恢复工作...")

	// ① 从数据库查询设备ID
	deviceID, err := l.getDeviceIDBySN(sn)
	if err != nil {
		logx.Errorf("[REBOOT-FAIL] ❌ 查询设备ID失败: sn=%s, error=%v", sn, err)
		return
	}

	// ② 恢复设备状态为"在线"（因为设备没有真正重启）
	l.updateDeviceOnlineStatus(deviceID)
	logx.Infof("[REBOOT-FAIL] ✅ 设备状态已恢复为在线: device_id=%d (重启中→在线)", deviceID)

	// ③ 记录失败原因
	logx.Errorf("[REBOOT-FAIL] ⚠️  设备重启失败详情:")
	logx.Errorf("           设备ID: %s", deviceId)
	logx.Errorf("           序列号: %s", sn)
	logx.Errorf("           请求ID: %s", requestId)
	logx.Errorf("           失败原因: %s", reason)
	logx.Errorf("           处理方案: 已恢复在线状态，保持WebSocket连接")
}

// getDeviceIDBySN 根据序列号查询设备ID
// 用于在处理设备响应时获取数据库主键
//
// 参数 sn string: 设备序列号
// 返回 int64: 设备ID
// 返回 error: 查询错误
func (l *DeviceWsLogic) getDeviceIDBySN(sn string) (int64, error) {
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return 0, fmt.Errorf("ServiceContext或数据库未初始化")
	}

	var deviceID int64
	err := l.svcCtx.DB.QueryRowContext(l.ctx, `
		SELECT id 
		FROM public.device 
		WHERE sn = $1 
		  AND deleted_at IS NULL
	`, sn).Scan(&deviceID)

	if err != nil {
		return 0, fmt.Errorf("查询设备ID失败: %w", err)
	}

	return deviceID, nil
}

// forceDisconnectDevice 强制断开设备WebSocket连接
// 用于服务端主动踢掉设备（如重启、禁用等场景）
//
// 完成以下操作：
//  1. 从连接池中移除设备
//  2. 发送关闭帧给设备（附带原因码）
//  3. 清理相关资源
//
// 参数 deviceId string: 设备ID字符串
// 参数 deviceID int64: 设备数据库ID
// 参数 closeCode int: WebSocket关闭码（参考RFC6455）
// 参数 closeReason string: 关闭原因描述
func (l *DeviceWsLogic) forceDisconnectDevice(deviceId string, deviceID int64, closeCode int, closeReason string) {
	lock.Lock()
	dc, ok := deviceConnMap[deviceId]
	if ok {
		delete(deviceConnMap, deviceId)
	}
	lock.Unlock()

	if !ok {
		logx.Slowf("[FORCE-DISCONNECT] 设备不在连接池中: device_id=%s", deviceId)
		return
	}

	// 发送关闭帧（带自定义关闭码）
	closeMessage := websocket.FormatCloseMessage(closeCode, closeReason)
	if err := dc.conn.WriteMessage(websocket.CloseMessage, closeMessage); err != nil {
		logx.Slowf("[FORCE-DISCONNECT] 发送关闭帧失败: device_id=%s, error=%v", deviceId, err)
	}

	// 底层关闭连接
	if err := dc.conn.Close(); err != nil {
		logx.Slowf("[FORCE-DISCONNECT] 关闭连接失败: device_id=%s, error=%v", deviceId, err)
	}

	logx.Infof("[FORCE-DISCONNECT] ✅ 设备已强制断开:")
	logx.Infof("              设备ID: %s", deviceId)
	logx.Infof("              关闭码: %d (%s)", closeCode, closeReason)
	logx.Infof("              剩余在线设备: %d", len(deviceConnMap))

	// 触发断开回调（更新离线状态等）
	l.handleDeviceDisconnect(deviceId, deviceID)
}

// startRebootTimeoutProtection 启动重启超时保护定时器
// 作用：防止设备收到reboot指令后不响应，导致状态卡在"重启中"
//
// 工作原理：
//  1. 在发送reboot指令成功后启动定时器（默认30秒）
//  2. 定时器到期后检查设备是否还在"重启中"状态
//  3. 如果是，强制标记为离线并断开连接
//  4. 记录超时警告日志
//
// 参数 deviceID int64: 设备数据库ID
// 参数 sn string: 设备序列号
// 参数 requestId string: 重启请求ID（用于日志追踪）
// 参数 deviceIdStr string: 设备ID字符串（用于连接池操作）
func (l *DeviceRebootLogic) startRebootTimeoutProtection(deviceID int64, sn string, requestId string, deviceIdStr string) {
	// 🔴 关键优化：使用5秒超时（原30秒太长）
	timeoutDuration := RebootTimeoutDuration

	logx.Infof("[REBOOT-TIMEOUT] ⏱️  启动超时保护定时器:")
	logx.Infof("              设备SN: %s", sn)
	logx.Infof("              超时时间: %v", timeoutDuration)
	logx.Infof("              启动时间: %s", time.Now().Format("2006-01-02 15:04:05"))

	// 使用time.AfterFunc启动异步定时器
	time.AfterFunc(timeoutDuration, func() {
		l.handleRebootTimeout(deviceID, sn, requestId, deviceIdStr)
	})
}

// handleRebootTimeout 处理重启超时事件
// 当设备在规定时间内未响应reboot指令时调用
//
// 完成以下操作：
//  1. 检查设备当前状态是否仍为"重启中"
//  2. 如果是，强制更新为"离线"
//  3. 断开WebSocket连接
//  4. 发送告警通知（可选）
//
// 参数 deviceID int64: 设备数据库ID
// 参数 sn string: 设备序列号
// 参数 requestId string: 请求ID
// 参数 deviceIdStr string: 设备ID字符串
func (l *DeviceRebootLogic) handleRebootTimeout(deviceID int64, sn string, requestId string, deviceIdStr string) {
	logx.Slowf("\n⏰⏰⏰ [REBOOT-TIMEOUT] 设备重启响应超时 ⏰⏰⏰")
	logx.Slowf("   设备SN: %s", sn)
	logx.Slowf("   请求ID: %s", requestId)
	logx.Slowf("   超时时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	logx.Slowf("   原因: 设备在30秒内未响应重启指令")
	logx.Slowf("   处理方案: 强制标记为离线并断开连接\n")

	// ① 检查设备当前状态（防止重复处理）
	currentStatus, err := l.getDeviceOnlineStatus(deviceID)
	if err != nil {
		logx.Errorf("[REBOOT-TIMEOUT] ❌ 查询设备状态失败: error=%v", err)
		return
	}

	// 如果已经不是"重启中"状态，说明已经正常处理过了
	if currentStatus != DeviceOnlineStatusRebooting {
		logx.Infof("[REBOOT-TIMEOUT] ℹ️  设备已不在重启中状态(当前=%d)，跳过超时处理", currentStatus)
		return
	}

	// ② 强制更新为离线状态
	if l.svcCtx != nil && l.svcCtx.DB != nil {
		_, updateErr := l.svcCtx.DB.ExecContext(context.Background(), `
			UPDATE public.device
			SET online_status = $2,
			    updated_at = NOW()
			WHERE id = $1
			  AND deleted_at IS NULL
		`, deviceID, DeviceOnlineStatusOffline)

		if updateErr != nil {
			logx.Errorf("[REBOOT-TIMEOUT] ❌ 强制更新离线状态失败: error=%v", updateErr)
		} else {
			logx.Infof("[REBOOT-TIMEOUT] ✅ 设备状态已强制更新为离线: device_id=%d (重启中→离线)", deviceID)
		}
	}

	// ③ 强制断开WebSocket连接（必须执行！）
	if IsDeviceOnline(deviceIdStr) {
		logx.Slowf("[REBOOT-TIMEOUT] ⚠️  设备仍在线，立即强制断开: device_id=%s", deviceIdStr)

		// 🔴 关键修复：直接操作连接池并关闭连接
		lock.Lock()
		dc, connExists := deviceConnMap[deviceIdStr]
		if connExists {
			delete(deviceConnMap, deviceIdStr)
		}
		lock.Unlock()

		if connExists && dc != nil && dc.conn != nil {
			// 发送关闭帧
			closeMsg := websocket.FormatCloseMessage(4005, "Reboot timeout")
			if writeErr := dc.conn.WriteMessage(websocket.CloseMessage, closeMsg); writeErr != nil {
				logx.Slowf("[REBOOT-TIMEOUT] 发送关闭帧失败: %v", writeErr)
			}

			// 强制关闭底层连接（必须！）
			if closeErr := dc.conn.Close(); closeErr != nil {
				logx.Slowf("[REBOOT-TIMEOUT] 关闭连接失败: %v", closeErr)
			} else {
				logx.Infof("[REBOOT-TIMEOUT] ✅ 连接已强制关闭: device_id=%s", deviceIdStr)
			}

			// 更新离线状态到数据库和Redis
			l.handleDeviceDisconnectFromTimeout(deviceIdStr, deviceID)
		} else {
			logx.Slowf("[REBOOT-TIMEOUT] 设备已不在连接池中或连接为空")
		}
	} else {
		logx.Infof("[REBOOT-TIMEOUT] 设备已离线，无需断开")
	}

	// ④ 记录超时审计日志
	logx.Errorf("[REBOOT-TIMEOUT] 🚨 超时审计记录:")
	logx.Errorf("           设备SN: %s", sn)
	logx.Errorf("           请求ID: %s", requestId)
	logx.Errorf("           超时时长: %ds", int(RebootTimeoutDuration.Seconds()))
	logx.Errorf("           最终状态: 离线(0)")
	logx.Errorf("           建议: 检查设备网络或固件是否正常")
}

// handleDeviceDisconnectFromTimeout 超时时调用的断开处理
// 用于更新数据库和Redis的离线状态
func (l *DeviceRebootLogic) handleDeviceDisconnectFromTimeout(deviceId string, deviceID int64) {
	if l.svcCtx != nil && l.svcCtx.DB != nil {
		_, err := l.svcCtx.DB.ExecContext(context.Background(), `
			UPDATE public.device
			SET online_status = $2,
			    updated_at = NOW()
			WHERE id = $1
			  AND deleted_at IS NULL
		`, deviceID, DeviceOnlineStatusOffline)

		if err != nil {
			logx.Errorf("[REBOOT-TIMEOUT] 更新离线状态失败: %v", err)
		} else {
			logx.Infof("[REBOOT-TIMEOUT] 数据库状态已更新为离线: device_id=%d", deviceID)
		}
	}

	logx.Infof("[REBOOT-TIMEOUT] ✅ 超时断开完成: device_id=%s, 剩余设备=%d", deviceId, len(deviceConnMap))
}

// getDeviceOnlineStatus 查询设备当前在线状态
// 用于超时保护等场景的状态检查
//
// 参数 deviceID int64: 设备数据库ID
// 返回 int: 当前在线状态值 (0=离线, 1=在线, 2=重启中)
// 返回 error: 查询错误
func (l *DeviceRebootLogic) getDeviceOnlineStatus(deviceID int64) (int, error) {
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return 0, fmt.Errorf("ServiceContext或数据库未初始化")
	}

	var status int
	err := l.svcCtx.DB.QueryRowContext(context.Background(), `
		SELECT online_status 
		FROM public.device 
		WHERE id = $1 
		  AND deleted_at IS NULL
	`, deviceID).Scan(&status)

	if err != nil {
		return 0, fmt.Errorf("查询设备状态失败: %w", err)
	}

	return status, nil
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
