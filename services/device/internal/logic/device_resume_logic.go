package logic

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/device/internal/commandsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeviceResumeLogic 设备继续播放指令逻辑
// 处理用户通过 App 向设备下发继续播放指令的业务逻辑
// 支持在线设备立即下发，离线设备缓存等待上线
// 包含用户身份验证、设备权限校验、参数校验等

type DeviceResumeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceResumeLogic 创建设备继续播放指令逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DeviceResumeLogic: 设备继续播放指令逻辑实例
func NewDeviceResumeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceResumeLogic {
	return &DeviceResumeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceResume 下发设备继续播放指令
// 接收继续播放指令请求，验证用户权限和设备状态，下发继续播放指令
// 参数 req *types.DeviceResumeReq: 继续播放指令请求
// 返回 *types.DeviceResumeResp: 继续播放指令响应
// 返回 error: 错误信息
func (l *DeviceResumeLogic) DeviceResume(req *types.DeviceResumeReq) (*types.DeviceResumeResp, error) {
	// 1. 校验 Token 是否存在
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	// 2. 校验请求参数
	if err := validateDeviceResumeReq(req); err != nil {
		return nil, fmt.Errorf("参数校验失败: %v", err)
	}

	sn := strings.ToUpper(strings.TrimSpace(req.Sn))
	action := strings.ToLower(strings.TrimSpace(req.Action))

	// 3. 查询设备是否存在
	deviceInfo, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}
	if deviceInfo == nil {
		return nil, fmt.Errorf("设备不存在: %s", sn)
	}

	// 4. 验证用户权限：查询 user_device_bind 绑定表
	bindInfo, err := l.svcCtx.UserDeviceBindRepo.FindByUserIdAndDeviceId(l.ctx, userID, deviceInfo.ID)
	if err != nil {
		return nil, fmt.Errorf("查询绑定关系失败: %v", err)
	}
	if bindInfo == nil {
		return nil, fmt.Errorf("无权限控制该设备")
	}

	// 5. 构造继续播放指令参数
	params := map[string]interface{}{
		"action": action,
	}
	// 如果指定了音量参数，添加到参数中
	if req.Volume > 0 {
		params["volume"] = req.Volume
	}

	// 6. 通过 commandsvc 创建并下发继续播放指令
	// commandsvc 内部会：
	//   - 检查设备在线状态（Redis + MySQL）
	//   - 在线时通过 MQTT 立即下发
	//   - 离线时缓存为 pending 状态，设备上线后自动推送
	cmdSvc := commandsvc.New(l.svcCtx)
	result, err := cmdSvc.CreateImmediateInstructionFromDesired(l.ctx, commandsvc.CreateImmediateInstructionInput{
		DeviceID:        deviceInfo.ID,
		DeviceSN:        sn,
		UserID:          userID,
		CommandCode:     "resume",
		InstructionType: commandsvc.InstructionTypeManual,
		Params:          params,
		Operator:        fmt.Sprintf("user:%d", userID),
		Reason:          fmt.Sprintf("用户下发继续播放指令: action=%s", action),
	})
	if err != nil {
		return nil, fmt.Errorf("创建继续播放指令失败: %v", err)
	}

	// 7. 组装响应
	status := "cached"
	message := "设备离线，指令已缓存，设备上线后将自动执行"
	if result.Status == "dispatched" || result.Status == "delivered" {
		status = "delivered"
		message = "继续播放指令已下发"
	}

	logx.Infof("设备继续播放指令已下发: user_id=%d, sn=%s, action=%s, instruction_id=%d, status=%s",
		userID, sn, action, result.InstructionID, status)

	return &types.DeviceResumeResp{
		InstructionID: result.InstructionID,
		Status:        status,
		Message:       message,
	}, nil
}

// validateDeviceResumeReq 校验继续播放指令请求参数
// 参数 req *types.DeviceResumeReq: 继续播放指令请求
// 返回 error: 校验错误
func validateDeviceResumeReq(req *types.DeviceResumeReq) error {
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}

	// 校验 SN 格式：16位字母数字组合
	sn := strings.TrimSpace(req.Sn)
	if sn == "" {
		return fmt.Errorf("设备序列号不能为空")
	}
	
	matched, _ := regexp.MatchString(`^[A-Za-z0-9]{16}$`, sn)
	if !matched {
		return fmt.Errorf("设备序列号格式错误，必须为16位字母数字组合")
	}

	// 校验 action 参数
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != "resume" {
		return fmt.Errorf("操作类型必须为 resume")
	}

	// 校验 volume 参数（可选）
	if req.Volume < 0 || req.Volume > 100 {
		return fmt.Errorf("音量值必须在 0-100 范围内")
	}

	return nil
}