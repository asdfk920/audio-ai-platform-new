package logic

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// initDeviceShadowOnFirstRegister 设备首次激活成功后：确保 device_shadow 表存在、写入 PG、初始化 Redis Hash
func (l *DeviceRegisterLogic) initDeviceShadowOnFirstRegister(deviceID int64, sn, productKey, clientIP string, req *types.DeviceRegisterReq) {
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		logx.Infof("[Device Register] 跳过影子初始化: DB 未配置")
		return
	}
	snNorm := strings.ToUpper(strings.TrimSpace(sn))
	if snNorm == "" || deviceID <= 0 {
		return
	}

	if err := l.svcCtx.DeviceShadowRepo.EnsureSchema(l.ctx); err != nil {
		logx.Errorf("[Device Register] 创建 device_shadow 表失败: %v", err)
		return
	}

	reported := map[string]interface{}{
		"run_state": "unknown",
	}
	if req != nil {
		if fw := strings.TrimSpace(req.FirmwareVersion); fw != "" {
			reported["firmware_version"] = fw
		}
		if hw := strings.TrimSpace(req.HardwareVersion); hw != "" {
			reported["hardware_version"] = hw
		}
		if mac := strings.TrimSpace(req.Mac); mac != "" {
			reported["mac"] = mac
		}
	}
	if ip := strings.TrimSpace(clientIP); ip != "" {
		reported["ip"] = ip
	}
	reportedBytes, _ := json.Marshal(reported)
	desiredBytes := json.RawMessage(`{}`)
	metaBytes := json.RawMessage(`{}`)

	inserted, err := l.svcCtx.DeviceShadowRepo.InsertInitialIfAbsent(l.ctx, deviceID, snNorm, reportedBytes, desiredBytes, metaBytes)
	if err != nil {
		logx.Errorf("[Device Register] 写入 device_shadow 失败 sn=%s: %v", snNorm, err)
	} else if inserted {
		logx.Infof("[Device Register] 已写入 device_shadow 表 sn=%s device_id=%d", snNorm, deviceID)
	}

	if l.svcCtx.Redis == nil {
		logx.Infof("[Device Register] 跳过 Redis 影子: Redis 未配置")
		return
	}

	empty, err := shadow.RedisShadowEmpty(l.ctx, l.svcCtx.Redis, snNorm)
	if err != nil {
		logx.Errorf("[Device Register] 检查 Redis 影子失败 sn=%s: %v", snNorm, err)
		return
	}
	if !empty && !inserted {
		return
	}

	ttl := shadowTTL(l.svcCtx.Config.DeviceShadow.SeedTTLSeconds, l.svcCtx.Config.DeviceShadow.HeartbeatTTLSeconds)
	in := shadow.RegisterInitInput{
		Sn:              snNorm,
		DeviceID:        deviceID,
		ProductKey:      productKey,
		Mac:             "",
		FirmwareVersion: "",
		HardwareVersion: "",
		IP:              clientIP,
	}
	if req != nil {
		in.Mac = strings.TrimSpace(req.Mac)
		in.FirmwareVersion = strings.TrimSpace(req.FirmwareVersion)
		in.HardwareVersion = strings.TrimSpace(req.HardwareVersion)
	}
	if err := shadow.InitRegisterShadow(l.ctx, l.svcCtx.Redis, ttl, in); err != nil {
		logx.Errorf("[Device Register] 初始化 Redis 影子 Hash 失败 sn=%s: %v", snNorm, err)
		return
	}
	logx.Infof("[Device Register] 已初始化 Redis 影子 Hash sn=%s key=%s", snNorm, shadow.ShadowKey(snNorm))
}

func shadowTTL(seedSec, heartbeatSec int) time.Duration {
	if seedSec > 0 {
		return time.Duration(seedSec) * time.Second
	}
	if heartbeatSec > 0 {
		return time.Duration(heartbeatSec) * time.Second
	}
	return 24 * time.Hour
}
