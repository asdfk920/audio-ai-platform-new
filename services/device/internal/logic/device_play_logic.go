package logic

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/device/internal/commandsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

var validPlayActions = map[string]bool{
	"play":  true,
	"pause": true,
	"stop":  true,
	"next":  true,
	"prev":  true,
}

// DevicePlayLogic 设备播放指令逻辑
// 处理用户通过 App 向设备下发播放指令的业务逻辑
type DevicePlayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDevicePlayLogic 创建设备播放指令逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DevicePlayLogic: 设备播放指令逻辑实例
func NewDevicePlayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DevicePlayLogic {
	return &DevicePlayLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DevicePlay 下发设备播放指令
// 流程：
//  1. 解析请求头，提取 Authorization 字段获取用户 Token
//  2. 解析请求体 JSON 数据，获取 sn、action、params
//  3. 校验 Token 签名和有效期，提取 user_id
//  4. 校验 sn 格式、action 有效性、media_url 格式、volume 范围
//  5. 查询 user_device_bind 验证用户权限
//  6. 查询 Redis/MySQL 检查设备在线状态
//  7. 在线：生成指令 ID，构造 MQTT 消息，发布到 device/cmd/{sn}
//  8. 离线：将命令缓存到 Redis 队列 device:cmd:queue:{sn}
//  9. 写入 device_cmd_log 命令日志表
//  10. 返回响应：delivered（在线）或 cached（离线）
//
// 参数 req *types.DevicePlayReq: 设备播放指令请求
// 返回 *types.DevicePlayResp: 设备播放指令响应
// 返回 error: 下发失败时的错误信息
func (l *DevicePlayLogic) DevicePlay(req *types.DevicePlayReq) (*types.DevicePlayResp, error) {
	// 1. 校验 Token 是否存在
	userID, ok := jwt.GetUserIdFromContext(l.ctx)
	if !ok || userID <= 0 {
		return nil, fmt.Errorf("请先登录")
	}

	// 2. 校验请求参数
	if err := validateDevicePlayReq(req); err != nil {
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

	// 5. 构造播放指令参数
	params := map[string]interface{}{
		"action": action,
	}
	if req.MediaURL != "" {
		params["media_url"] = req.MediaURL
	}
	if req.Volume > 0 {
		params["volume"] = req.Volume
	}

	// 6. 通过 commandsvc 创建并下发播放指令
	// commandsvc 内部会：
	//   - 检查设备在线状态（Redis + MySQL）
	//   - 在线时通过 MQTT 立即下发
	//   - 离线时缓存为 pending 状态，设备上线后自动推送
	cmdSvc := commandsvc.New(l.svcCtx)
	result, err := cmdSvc.CreateImmediateInstructionFromDesired(l.ctx, commandsvc.CreateImmediateInstructionInput{
		DeviceID:        deviceInfo.ID,
		DeviceSN:        sn,
		UserID:          userID,
		CommandCode:     "play",
		InstructionType: commandsvc.InstructionTypeManual,
		Params:          params,
		Operator:        fmt.Sprintf("user:%d", userID),
		Reason:          fmt.Sprintf("用户下发播放指令: action=%s", action),
	})
	if err != nil {
		return nil, fmt.Errorf("创建播放指令失败: %v", err)
	}

	// 7. 组装响应
	status := "cached"
	message := "设备离线，指令已缓存，设备上线后将自动执行"
	if result.Status == "dispatched" || result.Status == "delivered" {
		status = "delivered"
		message = "播放指令已下发"
	}

	logx.Infof("设备播放指令已下发: user_id=%d, sn=%s, action=%s, instruction_id=%d, status=%s",
		userID, sn, action, result.InstructionID, status)

	return &types.DevicePlayResp{
		InstructionID: result.InstructionID,
		Status:        status,
		Message:       message,
	}, nil
}

// validateDevicePlayReq 校验设备播放指令请求参数
// 校验规则：
//   - sn: 16 位字母数字组合
//   - action: 必须是 play/pause/stop/next/prev 之一
//   - media_url: 如果提供，必须是有效 URL
//   - volume: 如果提供，必须在 0-100 范围内
//
// 参数 req *types.DevicePlayReq: 设备播放指令请求
// 返回 error: 校验失败时的错误信息
func validateDevicePlayReq(req *types.DevicePlayReq) error {
	// sn 校验：16 位字母数字
	snRegex := regexp.MustCompile(`(?i)^[A-Z0-9]{16}$`)
	if !snRegex.MatchString(req.Sn) {
		return fmt.Errorf("SN 格式错误，必须为 16 位字母数字组合")
	}

	// action 校验
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if !validPlayActions[action] {
		return fmt.Errorf("无效的 action 参数，支持: play/pause/stop/next/prev")
	}

	// media_url 校验
	if req.MediaURL != "" {
		if _, err := url.ParseRequestURI(req.MediaURL); err != nil {
			return fmt.Errorf("media_url 格式无效")
		}
	}

	// volume 校验
	if req.Volume < 0 || req.Volume > 100 {
		return fmt.Errorf("volume 必须在 0-100 范围内")
	}

	return nil
}
