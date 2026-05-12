package logic

import (
	"context"
	"fmt"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/jacklau/audio-ai-platform/services/device/internal/util"
	"github.com/zeromicro/go-zero/core/logx"
)

// DeviceRegisterLogic 设备注册逻辑
// 处理设备首次注册的业务逻辑（设备端携带预烧录SN，后端分配密钥）
type DeviceRegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewDeviceRegisterLogic 创建设备注册逻辑实例
func NewDeviceRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceRegisterLogic {
	return &DeviceRegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceRegister 设备注册（接收预烧录SN，返回设备密钥）
//
// 完整流程：
//  1. 接收设备SN参数
//  2. 校验SN格式（16位长度 + 校验码验证）
//  3. 查询数据库判断设备是否已存在
//     - 已存在：直接返回已有密钥
//     - 不存在：生成32位随机密钥，插入数据库
//  4. 返回 SN + device_secret 给设备端保存到Flash
//
// 参数 req *types.DeviceRegisterReq: 设备注册请求（包含sn）
// 返回 *types.DeviceRegisterResp: 设备注册响应（包含device_secret）
// 返回 error: 注册失败时的错误信息
func (l *DeviceRegisterLogic) DeviceRegister(req *types.DeviceRegisterReq) (*types.DeviceRegisterResp, error) {
	sn := strings.TrimSpace(strings.ToUpper(req.Sn))

	if err := validateSnFormat(sn); err != nil {
		return nil, fmt.Errorf("SN格式校验失败: %v", err)
	}

	logx.Infof("收到设备注册请求: sn=%s (显示格式:%s)", sn, util.FormatSNDisplay(sn))

	existingDevice, err := l.svcCtx.DeviceRegister.FindBySn(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}

	if existingDevice != nil && existingDevice.Secret != "" {
		logx.Infof("设备已注册，返回已有密钥: sn=%s", sn)
		return &types.DeviceRegisterResp{
			Sn:           sn,
			DeviceSecret: existingDevice.Secret,
		}, nil
	}

	deviceSecret := l.svcCtx.DeviceRegister.GenerateDeviceSecret()

	logx.Infof("新设备注册，生成密钥: sn=%s", sn)

	deviceID, err := l.svcCtx.DeviceRegister.CreateDeviceWithSecret(l.ctx, sn, deviceSecret)
	if err != nil {
		return nil, fmt.Errorf("创建设备记录失败: %v", err)
	}

	if err := l.svcCtx.DeviceTopicACLRepo.CreateDefaultRulesForDevice(l.ctx, sn); err != nil {
		logx.Errorf("创建设备默认 Topic 权限规则失败: sn=%s, err=%v (不影响注册结果)", sn, err)
	} else {
		logx.Infof("已为设备创建默认 Topic 白名单规则: sn=%s", sn)
	}

	snDisplay := util.FormatSNDisplay(sn)
	logx.Infof("设备注册成功: sn=%s (显示格式:%s), device_id=%d", sn, snDisplay, deviceID)

	return &types.DeviceRegisterResp{
		Sn:           sn,
		DeviceSecret: deviceSecret,
	}, nil
}

// validateSnFormat 校验设备序列号格式
//
// 校验规则：
//   - 长度必须为16位
//   - 只能包含大写字母和数字
//   - 厂商码必须是已知的（AU/HX等）
//   - 设备类型必须是已知的（SP/HP/SB等）
//   - 校验码必须正确（SHA256哈希校验）
//
// 参数 sn string: 待校验的设备序列号
// 返回 error: 校验失败时的错误信息，nil表示校验通过
func validateSnFormat(sn string) error {
	if sn == "" {
		return fmt.Errorf("设备序列号不能为空")
	}

	if !util.ValidateSNFormat(sn) {
		parsed := util.ParseSN(sn)
		if errMsg, ok := parsed["error"].(string); ok {
			return fmt.Errorf("%s", errMsg)
		}
		return fmt.Errorf("SN格式错误或校验码无效")
	}

	return nil
}
