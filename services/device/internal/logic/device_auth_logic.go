package logic

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeviceAuthLogic 设备认证逻辑
// 处理设备使用 token 向云端认证身份的业务逻辑
type DeviceAuthLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceAuthLogic 创建设备认证逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DeviceAuthLogic: 设备认证逻辑实例
func NewDeviceAuthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceAuthLogic {
	return &DeviceAuthLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceAuth 设备认证
// 流程：
//   1. 校验请求数据格式（SN、Token）
//   2. 查询数据库验证 SN 和 Token 是否匹配
//   3. 验证通过：更新设备在线状态，返回认证成功信息
//   4. 验证失败：返回错误信息
//
// 参数 req *types.DeviceAuthReq: 设备认证请求
// 返回 *types.DeviceAuthResp: 设备认证响应
// 返回 error: 认证失败时的错误信息
func (l *DeviceAuthLogic) DeviceAuth(req *types.DeviceAuthReq) (*types.DeviceAuthResp, error) {
	// 1. 校验请求数据格式
	if err := validateDeviceAuthReq(req); err != nil {
		return nil, fmt.Errorf("数据格式校验失败: %v", err)
	}

	// 2. 查询数据库验证 SN 和 Token 是否匹配
	deviceInfo, err := l.svcCtx.DeviceRegister.VerifyToken(l.ctx, req.Sn, req.Token)
	if err != nil {
		return nil, fmt.Errorf("验证设备 token 失败: %v", err)
	}

	// 3. 验证失败分支
	if deviceInfo == nil {
		logx.Infof("设备认证失败: sn=%s, 原因: token 无效或设备不存在", req.Sn)
		return nil, fmt.Errorf("认证失败: token 无效或设备未注册")
	}

	// 4. 验证通过分支：更新设备在线状态
	if err := l.svcCtx.DeviceRegister.UpdateOnlineStatus(l.ctx, req.Sn); err != nil {
		logx.Errorf("更新设备在线状态失败: sn=%s, err=%v", req.Sn, err)
		// 不影响认证结果，仅记录日志
	}

	logx.Infof("设备认证成功: sn=%s, device_id=%d", req.Sn, deviceInfo.ID)

	// 5. 返回认证成功信息
	return &types.DeviceAuthResp{
		Success:         true,
		DeviceID:        deviceInfo.ID,
		Sn:              deviceInfo.Sn,
		Model:           deviceInfo.Model,
		FirmwareVersion: deviceInfo.FirmwareVersion,
		Message:         "连接成功",
	}, nil
}

// validateDeviceAuthReq 校验设备认证请求数据格式
// 校验规则：
//   - SN: 16 位字母数字，正则 ^[A-Z0-9]{16}$，不区分大小写
//   - Token: 非空字符串
//
// 参数 req *types.DeviceAuthReq: 设备认证请求
// 返回 error: 校验失败时的错误信息
func validateDeviceAuthReq(req *types.DeviceAuthReq) error {
	// SN 校验：16 位字母数字
	snRegex := regexp.MustCompile(`(?i)^[A-Z0-9]{16}$`)
	if !snRegex.MatchString(req.Sn) {
		return fmt.Errorf("SN 格式错误，必须为 16 位字母数字组合")
	}

	// Token 校验：非空字符串
	if req.Token == "" {
		return fmt.Errorf("认证 token 不能为空")
	}

	return nil
}
