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

// DeviceRegister 设备注册/激活接口
//
// 完整流程：
// 1. 接收 SN、密钥、版本等参数并校验
// 2. 查询云端设备；不存在则按请求中的 SN+密钥自动预录入（status=未激活）
// 3. 校验密钥是否匹配
// 4. 已激活 → 直接返回成功；未激活 → 执行首次激活
// 5. 注册完成（不生成 token，需再调 /api/device/auth）
func (l *DeviceRegisterLogic) DeviceRegister(req *types.DeviceRegisterReq) (*types.DeviceRegisterResp, error) {
	logx.Infof("====================================")
	logx.Infof("[Device Register] 📝 开始设备注册/激活流程...")

	sn := normalizeSN(req.Sn)
	logx.Infof("[Device Register] 📋 设备SN: %s", sn)

	if err := l.validateRegisterRequest(sn, req.DeviceSecret); err != nil {
		return nil, fmt.Errorf("请求参数校验失败: %v", err)
	}

	logx.Infof("[Device Register] 🔍 查询云端设备...")

	device, err := l.svcCtx.DeviceRepo.FindBySnForActivation(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	if device == nil || device.ID == 0 {
		device, err = l.ensurePreRegistered(sn, req.DeviceSecret, req.DeviceNameRaw)
		if err != nil {
			return nil, err
		}
	}

	logx.Infof("[Device Register] ✅ 设备就绪: id=%d status=%d", device.ID, device.Status)

	if !verifyDeviceSecretAgainstStored(req.DeviceSecret, device.DeviceSecret) {
		logx.Errorf("[Device Register] ❌ 密钥不匹配: sn=%s", sn)
		return nil, fmt.Errorf("认证失败：设备密钥不正确")
	}

	logx.Infof("[Device Register] ✅ 密钥验证通过")

	now := time.Now()

	if device.Status == model.DeviceStatusNormal {
		logx.Infof("[Device Register] ✅ 设备已是激活状态: sn=%s (无需重复激活)", sn)
		if nameRaw := truncateDeviceNameRaw(req.DeviceNameRaw); nameRaw != "" {
			if err := l.svcCtx.DeviceRepo.UpdateDeviceNameRaw(l.ctx, device.ID, nameRaw); err != nil {
				logx.Errorf("[Device Register] 更新设备原始名称失败: sn=%s err=%v", sn, err)
			}
		}

		return &types.DeviceRegisterResp{
			Sn:          sn,
			DeviceID:    device.ID,
			Status:      "already_activated",
			ActivatedAt: formatTimePtr(device.LastActiveAt),
			Message:     "设备已激活，无需重复注册",
		}, nil
	}

	logx.Infof("[Device Register] 🔄 执行首次激活流程: sn=%d", device.ID)

	clientIP := l.getClientIP()
	deviceNameRaw := truncateDeviceNameRaw(req.DeviceNameRaw)
	logx.Infof("[Device Register] 📍 记录激活信息: ip=%s firmware=%s hardware=%s mac=%s device_name_raw=%s",
		clientIP,
		req.FirmwareVersion,
		req.HardwareVersion,
		req.Mac,
		deviceNameRaw,
	)

	if err := l.svcCtx.DeviceRepo.ActivateDevice(
		l.ctx,
		device.ID,
		clientIP,
		req.FirmwareVersion,
		req.HardwareVersion,
		req.Mac,
		deviceNameRaw,
	); err != nil {
		logx.Errorf("[Device Register] ❌ 激活设备失败: %v", err)
		return nil, fmt.Errorf("激活设备失败: %w", err)
	}
	if deviceNameRaw != "" {
		if err := l.svcCtx.DeviceRepo.UpdateDeviceNameRaw(l.ctx, device.ID, deviceNameRaw); err != nil {
			logx.Errorf("[Device Register] 更新设备原始名称失败: sn=%s err=%v", sn, err)
		}
	}

	l.initDeviceShadowOnFirstRegister(device.ID, sn, device.ProductKey, clientIP, req)

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

	if err := validateReqSN(sn); err != nil {
		return err
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

// ensurePreRegistered 云端无可用设备记录时自动预录入（或恢复软删除设备）
func (l *DeviceRegisterLogic) ensurePreRegistered(sn, deviceSecret, deviceNameRaw string) (*model.Device, error) {
	secret := strings.TrimSpace(deviceSecret)
	nameRaw := truncateDeviceNameRaw(deviceNameRaw)

	existing, err := l.svcCtx.DeviceRepo.FindBySnIncludingDeleted(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	if existing != nil && existing.ID > 0 {
		if existing.DeletedAt != nil {
			if err := l.svcCtx.DeviceRepo.ClearDeletedAt(l.ctx, existing.ID); err != nil {
				return nil, fmt.Errorf("恢复设备失败: %w", err)
			}
			logx.Infof("[Device Register] ♻️ 恢复已删除设备: sn=%s id=%d", sn, existing.ID)
		} else if existing.Status != model.DeviceStatusUnregistered &&
			existing.Status != model.DeviceStatusNormal &&
			existing.Status != model.DeviceStatusDefault &&
			existing.Status != model.DeviceStatusUnauthenticated {
			return nil, fmt.Errorf("设备状态不允许注册: status=%d", existing.Status)
		}

		if err := l.svcCtx.DeviceRepo.ResetForPreRegistration(l.ctx, existing.ID, secret, nameRaw); err != nil {
			return nil, fmt.Errorf("更新预录入信息失败: %w", err)
		}
		logx.Infof("[Device Register] 📝 已更新预录入设备: sn=%s id=%d", sn, existing.ID)
	} else {
		id, err := l.svcCtx.DeviceRepo.CreateWithFullInfo(l.ctx, &model.Device{
			Sn:            sn,
			ProductKey:    "default",
			DeviceSecret:  secret,
			DeviceNameRaw: nameRaw,
			Status:        model.DeviceStatusUnregistered,
			OnlineStatus:  model.DeviceOnlineStatusOffline,
		})
		if err != nil {
			return nil, fmt.Errorf("自动预录入失败: %w", err)
		}
		logx.Infof("[Device Register] 🆕 自动预录入新设备: sn=%s id=%d", sn, id)
	}

	device, err := l.svcCtx.DeviceRepo.FindBySnForActivation(l.ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("查询预录入设备失败: %w", err)
	}
	if device == nil || device.ID == 0 {
		return nil, fmt.Errorf("自动预录入后仍无法找到设备")
	}
	return device, nil
}

func truncateDeviceNameRaw(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	runes := []rune(name)
	if len(runes) > 100 {
		return string(runes[:100])
	}
	return name
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

func formatTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
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
