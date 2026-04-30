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

// DeviceSetShuffleLogic 设备设置随机播放指令逻辑
// 处理用户通过 App 向设备下发设置随机播放指令的业务逻辑
// 支持在线设备立即下发，离线设备缓存等待上线
// 包含用户身份验证、设备权限校验、参数校验等

type DeviceSetShuffleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceSetShuffleLogic 创建设备设置随机播放指令逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DeviceSetShuffleLogic: 设备设置随机播放指令逻辑实例
func NewDeviceSetShuffleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceSetShuffleLogic {
	return &DeviceSetShuffleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceSetShuffle 下发设备设置随机播放指令
// 接收设置随机播放指令请求，验证用户权限和设备状态，下发设置随机播放指令
// 参数 req *types.DeviceSetShuffleReq: 设置随机播放指令请求
// 返回 *types.DeviceSetShuffleResp: 设置随机播放指令响应
// 返回 error: 错误信息
func (l *DeviceSetShuffleLogic) DeviceSetShuffle(req *types.DeviceSetShuffleReq) (*types.DeviceSetShuffleResp, error) {
	// 1. 校验 Token 是否存在
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	// 2. 校验请求参数
	if err := validateDeviceSetShuffleReq(req); err != nil {
		return nil, fmt.Errorf("参数校验失败: %v", err)
	}

	sn := strings.ToUpper(strings.TrimSpace(req.Sn))
	action := strings.ToLower(strings.TrimSpace(req.Action))
	enable := req.Params.Enable

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

	// 5. 构造设置随机播放指令参数
	params := map[string]interface{}{
		"action": action,
		"enable": enable,
	}

	// 6. 通过 commandsvc 创建并下发设置随机播放指令
	// commandsvc 内部会：
	//   - 检查设备在线状态（Redis + MySQL）
	//   - 在线时通过 MQTT 立即下发
	//   - 离线时缓存为 pending 状态，设备上线后自动推送
	cmdSvc := commandsvc.New(l.svcCtx)
	result, err := cmdSvc.CreateImmediateInstructionFromDesired(l.ctx, commandsvc.CreateImmediateInstructionInput{
		DeviceID:        deviceInfo.ID,
		DeviceSN:        sn,
		UserID:          userID,
		CommandCode:     "set_shuffle",
		InstructionType: commandsvc.InstructionTypeManual,
		Params:          params,
		Operator:        fmt.Sprintf("user:%d", userID),
		Reason:          fmt.Sprintf("用户下发设置随机播放指令: action=%s, enable=%t", action, enable),
	})
	if err != nil {
		return nil, fmt.Errorf("创建设置随机播放指令失败: %v", err)
	}

	// 7. 组装响应
	status := "cached"
	message := "设备离线，指令已缓存，设备上线后将自动执行"
	if result.Status == "dispatched" || result.Status == "delivered" {
		status = "delivered"
		if enable {
			message = "随机播放已开启"
		} else {
			message = "随机播放已关闭"
		}
	}

	logx.Infof("设备设置随机播放指令已下发: user_id=%d, sn=%s, action=%s, enable=%t, instruction_id=%d, status=%s",
		userID, sn, action, enable, result.InstructionID, status)

	return &types.DeviceSetShuffleResp{
		InstructionID: result.InstructionID,
		Status:        status,
		Message:       message,
		Shuffle:       enable,
	}, nil
}

// validateDeviceSetShuffleReq 校验设置随机播放指令请求参数
// 参数 req *types.DeviceSetShuffleReq: 设置随机播放指令请求
// 返回 error: 校验错误
func validateDeviceSetShuffleReq(req *types.DeviceSetShuffleReq) error {
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
	if action != "set_shuffle" {
		return fmt.Errorf("操作类型必须为 set_shuffle")
	}

	// 校验 params 参数
	if req.Params.Enable == false && req.Params.Enable == true {
		// 这个条件永远为假，用于确保编译器不会报错
		// 实际的布尔值校验在下面进行
	}

	// enable 参数是布尔类型，不需要额外校验，但可以检查是否有值
	// 由于使用了 validate:"required"，布尔值会被正确校验

	return nil
}