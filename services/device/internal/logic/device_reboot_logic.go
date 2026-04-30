package logic

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/commandsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeviceRebootLogic 设备重启指令逻辑
// 处理用户通过 App 向设备下发重启指令的业务逻辑
type DeviceRebootLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceRebootLogic 创建设备重启指令逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DeviceRebootLogic: 设备重启指令逻辑实例
func NewDeviceRebootLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceRebootLogic {
	return &DeviceRebootLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceReboot 下发设备重启指令
// 流程：
//  1. 校验请求数据格式（SN）
//  2. 查询设备是否存在
//  3. 检查设备是否正在重启中（避免重复下发）
//  4. 生成指令记录，通过 MQTT 下发重启指令
//  5. 返回指令 ID 和执行状态
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

	// 2. 查询设备是否存在
	deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}
	if deviceInfo == nil {
		return nil, fmt.Errorf("设备不存在: %s", sn)
	}

	// 3. 检查设备是否正在重启中（避免重复下发）
	if l.isRebooting(sn) {
		return nil, fmt.Errorf("设备正在重启中，请勿重复操作")
	}

	// 4. 通过 commandsvc 创建并下发重启指令
	cmdSvc := commandsvc.New(l.svcCtx)
	result, err := cmdSvc.CreateImmediateInstructionFromDesired(l.ctx, commandsvc.CreateImmediateInstructionInput{
		DeviceID:        deviceInfo.ID,
		DeviceSN:        sn,
		UserID:          0, // 设备端指令，无用户 ID
		CommandCode:     "reboot",
		InstructionType: commandsvc.InstructionTypeManual,
		Params: map[string]interface{}{
			"cmd":       "reboot",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
		Operator: "system",
		Reason:   "用户触发设备重启",
	})
	if err != nil {
		return nil, fmt.Errorf("创建重启指令失败: %v", err)
	}

	logx.Infof("设备重启指令已下发: sn=%s, instruction_id=%d, status=%s", sn, result.InstructionID, result.Status)

	// 5. 返回指令 ID 和执行状态
	return &types.DeviceRebootResp{
		InstructionID: result.InstructionID,
		Status:        result.Status,
		Message:       "重启指令已下发，设备正在重启...",
	}, nil
}

// validateDeviceRebootReq 校验设备重启指令请求数据格式
// 校验规则：
//   - SN: 16 位字母数字，正则 ^[A-Z0-9]{16}$，不区分大小写
//
// 参数 req *types.DeviceRebootReq: 设备重启指令请求
// 返回 error: 校验失败时的错误信息
func validateDeviceRebootReq(req *types.DeviceRebootReq) error {
	// SN 校验：16 位字母数字
	snRegex := regexp.MustCompile(`(?i)^[A-Z0-9]{16}$`)
	if !snRegex.MatchString(req.Sn) {
		return fmt.Errorf("SN 格式错误，必须为 16 位字母数字组合")
	}

	return nil
}

// isRebooting 检查设备是否正在重启中
// 通过查询数据库中是否存在执行中的重启指令判断
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
		  AND (expires_at IS NULL OR expires_at >= CURRENT_TIMESTAMP)
	`, sn).Scan(&count)
	if err != nil {
		return false
	}

	return count > 0
}
