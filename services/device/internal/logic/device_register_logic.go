package logic

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeviceRegisterLogic 设备注册逻辑
// 处理设备首次注册的业务逻辑
type DeviceRegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceRegisterLogic 创建设备注册逻辑实例
// 参数 ctx context.Context: 请求上下文
// 参数 svcCtx *svc.ServiceContext: 服务上下文
// 返回 *DeviceRegisterLogic: 设备注册逻辑实例
func NewDeviceRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceRegisterLogic {
	return &DeviceRegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceRegister 设备注册
// 流程：
//  1. 校验请求数据格式（SN、Model、Version）
//  2. 查询数据库验证 SN 是否已存在
//  3. SN 已存在：返回已有 token
//  4. SN 不存在：生成新 token，插入数据库，返回新 token
//
// 参数 req *types.DeviceRegisterReq: 设备注册请求
// 返回 *types.DeviceRegisterResp: 设备注册响应（包含 token）
// 返回 error: 注册失败时的错误信息
func (l *DeviceRegisterLogic) DeviceRegister(req *types.DeviceRegisterReq) (*types.DeviceRegisterResp, error) {
	// 1. 校验请求数据格式
	if err := validateDeviceRegisterReq(req); err != nil {
		return nil, fmt.Errorf("数据格式校验失败: %v", err)
	}

	// 2. 查询数据库验证 SN 是否已存在
	existingDevice, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, req.Sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}

	// 3. SN 已存在分支：返回已有 token
	if existingDevice != nil {
		logx.Infof("设备已注册，返回已有 token: %s", req.Sn)
		return &types.DeviceRegisterResp{
			Token: existingDevice.AuthToken,
		}, nil
	}

	// 4. SN 不存在分支（新设备注册）
	logx.Infof("新设备注册: %s", req.Sn)

	// 4.1 生成认证 token
	authToken := l.svcCtx.DeviceRegister.GenerateAuthToken(req.Sn)

	// 4.2 插入数据库
	deviceID, err := l.svcCtx.DeviceRegister.CreateDevice(l.ctx, req.Sn, req.Model, req.FirmwareVersion, authToken)
	if err != nil {
		return nil, fmt.Errorf("创建设备记录失败: %v", err)
	}

	logx.Infof("新设备注册成功: sn=%s, device_id=%d", req.Sn, deviceID)

	// 4.3 返回 token
	return &types.DeviceRegisterResp{
		Token: authToken,
	}, nil
}

// validateDeviceRegisterReq 校验设备注册请求数据格式
// 校验规则：
//   - SN: 16 位字母数字，正则 ^[A-Z0-9]{16}$，不区分大小写
//   - Model: 非空字符串
//   - FirmwareVersion: 非空字符串
//
// 参数 req *types.DeviceRegisterReq: 设备注册请求
// 返回 error: 校验失败时的错误信息
func validateDeviceRegisterReq(req *types.DeviceRegisterReq) error {
	// SN 校验：16 位字母数字
	snRegex := regexp.MustCompile(`(?i)^[A-Z0-9]{16}$`)
	if !snRegex.MatchString(req.Sn) {
		return fmt.Errorf("SN 格式错误，必须为 16 位字母数字组合")
	}

	// Model 校验：非空字符串
	if req.Model == "" {
		return fmt.Errorf("设备型号不能为空")
	}

	// FirmwareVersion 校验：非空字符串
	if req.FirmwareVersion == "" {
		return fmt.Errorf("固件版本号不能为空")
	}

	return nil
}
