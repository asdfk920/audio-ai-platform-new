package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

type DeviceRegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeviceRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeviceRegisterLogic {
	return &DeviceRegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeviceRegister 设备注册/激活接口（预录入+首次激活模式）
//
// 完整流程：
// 1. 接收SN、密钥、版本等参数
// 2. 校验参数合法性（非空、格式）
// 3. 根据SN查询数据库，判断设备是否已预录入
// 4. 设备不存在 → 返回注册失败（非法设备）
// 5. 设备存在 → 校验密钥是否匹配
// 6. 密钥不匹配 → 返回认证失败
// 7. 密钥匹配 → 检查设备当前状态
// 8. 已激活过的设备 → 直接返回成功（不重复激活）
// 9. 未激活的设备 → 更新数据库状态为已激活，记录IP、版本信息
// 10. 注册完成，返回成功（⚠️ 不生成token，设备需调用认证接口获取）
func (l *DeviceRegisterLogic) DeviceRegister(req *types.DeviceRegisterReq) (*types.DeviceRegisterResp, error) {
	logx.Infof("====================================")
	logx.Infof("[Device Register] 📝 开始设备注册/激活流程...")

	sn := normalizeSN(req.Sn)
	logx.Infof("[Device Register] 📋 设备SN: %s", sn)

	if err := l.validateRegisterRequest(sn, req.DeviceSecret); err != nil {
		return nil, fmt.Errorf("请求参数校验失败: %v", err)
	}

	logx.Infof("[Device Register] 🔍 查询预录入设备...")

	device, err := l.svcCtx.DeviceRepo.FindBySnForActivation(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	if device == nil || device.ID == 0 {
		logx.Errorf("[Device Register] ❌ 设备不存在: sn=%s (未在云端预录入)", sn)
		return nil, fmt.Errorf("非法设备：该SN未在云端预录入")
	}

	logx.Infof("[Device Register] ✅ 找到预录入设备: id=%d status=%d", device.ID, device.Status)

	if !verifyDeviceSecretAgainstStored(req.DeviceSecret, device.DeviceSecret) {
		logx.Errorf("[Device Register] ❌ 密钥不匹配: sn=%s", sn)
		return nil, fmt.Errorf("认证失败：设备密钥不正确")
	}

	logx.Infof("[Device Register] ✅ 密钥验证通过")

	now := time.Now()

	if device.Status == model.DeviceStatusNormal {
		logx.Infof("[Device Register] ✅ 设备已是激活状态: sn=%s (无需重复激活)", sn)

		return &types.DeviceRegisterResp{
			Sn:          sn,
			DeviceID:    device.ID,
			Status:      "already_activated",
			ActivatedAt: device.LastActiveAt.Format(time.RFC3339),
			Message:     "设备已激活，无需重复注册",
		}, nil
	}

	logx.Infof("[Device Register] 🔄 执行首次激活流程: sn=%d", device.ID)

	clientIP := l.getClientIP()
	logx.Infof("[Device Register] 📍 记录激活信息: ip=%s firmware=%s hardware=%s mac=%s",
		clientIP,
		req.FirmwareVersion,
		req.HardwareVersion,
		req.Mac,
	)

	if err := l.svcCtx.DeviceRepo.ActivateDevice(
		l.ctx,
		device.ID,
		clientIP,
		req.FirmwareVersion,
		req.HardwareVersion,
		req.Mac,
	); err != nil {
		logx.Errorf("[Device Register] ❌ 激活设备失败: %v", err)
		return nil, fmt.Errorf("激活设备失败: %w", err)
	}

	logx.Infof("\n====================================")
	logx.Infof("🎉 [Device Register] ✅✅✅ 设备激活成功! ✅✅✅")
	logx.Infof("====================================")

	return &types.DeviceRegisterResp{
		Sn:          sn,
		DeviceID:    device.ID,
		Status:      "activated",
		ActivatedAt: now.Format(time.RFC3339),
		Message:     "设备首次激活成功，请调用 /api/device/auth 获取访问凭证",
	}, nil
}

func (l *DeviceRegisterLogic) validateRegisterRequest(sn string, deviceSecret string) error {
	if sn == "" {
		return fmt.Errorf("设备序列号不能为空")
	}

	sn = strings.TrimSpace(sn)
	if len(sn) != 16 {
		return fmt.Errorf("设备序列号长度错误: 期望16位, 实际%d位", len(sn))
	}

	for _, c := range sn {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z')) {
			return fmt.Errorf("设备序列号格式错误: 只允许字母和数字")
		}
	}

	if strings.TrimSpace(deviceSecret) == "" {
		return fmt.Errorf("设备密钥不能为空")
	}

	if len(strings.TrimSpace(deviceSecret)) < 16 {
		return fmt.Errorf("设备密钥长度不足: 至少需要16位")
	}

	return nil
}

func (l *DeviceRegisterLogic) getClientIP() string {
	return ""
}

func normalizeSN(sn string) string {
	result := make([]byte, 0, len(sn))
	for i := 0; i < len(sn); i++ {
		c := sn[i]
		if c >= 'a' && c <= 'z' {
			result = append(result, c-32)
		} else if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result = append(result, c)
		}
	}
	return string(result)
}

func verifyDeviceSecretAgainstStored(requestPlaintext, stored string) bool {
	requestPlaintext = strings.TrimSpace(requestPlaintext)
	stored = strings.TrimSpace(stored)
	if requestPlaintext == "" || stored == "" {
		return false
	}
	if requestPlaintext == stored {
		return true
	}
	return false
}
