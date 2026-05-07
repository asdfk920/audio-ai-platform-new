package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/services/ota/internal/model"
	"github.com/jacklau/audio-ai-platform/services/ota/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/ota/internal/types"
	"github.com/jacklau/audio-ai-platform/services/ota/internal/util"
)

type BatchCheckVersionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBatchCheckVersionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchCheckVersionLogic {
	return &BatchCheckVersionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BatchCheckVersionLogic) BatchCheckVersion(userID int64, req *types.BatchCheckVersionReq) (*types.BatchCheckVersionResp, error) {
	sns := make([]string, 0, len(req.Devices))
	for _, d := range req.Devices {
		sns = append(sns, d.DeviceSn)
	}

	ownedMap, err := l.svcCtx.DeviceRepo.FindDeviceIDsByUserAndSNs(userID, sns)
	if err != nil {
		return nil, fmt.Errorf("查询设备归属失败: %w", err)
	}

	results := make([]types.DeviceCheckResult, 0, len(req.Devices))
	now := time.Now().Unix()

	for _, device := range req.Devices {
		if _, ok := ownedMap[device.DeviceSn]; !ok {
			results = append(results, types.DeviceCheckResult{
				DeviceSn:     device.DeviceSn,
				Model:        device.Model,
				CurrentVer:   device.FirmwareVer,
				CheckTime:    now,
				ErrorMessage: "设备未绑定或无权操作",
			})
			continue
		}
		result := l.checkSingleDevice(device, now)
		results = append(results, result)
	}

	return &types.BatchCheckVersionResp{
		Total:     len(results),
		Results:   results,
		CheckTime: now,
	}, nil
}

func (l *BatchCheckVersionLogic) checkSingleDevice(device types.DeviceVersionInfo, checkTime int64) types.DeviceCheckResult {
	result := types.DeviceCheckResult{
		DeviceSn:   device.DeviceSn,
		Model:      device.Model,
		CurrentVer: device.FirmwareVer,
		CheckTime:  checkTime,
	}

	fmt.Printf("[OTA] Checking device: model=%s, deviceType=%s, currentVersion=%s\n",
		device.Model, device.DeviceType, device.FirmwareVer)

	latestFirmware, err := l.svcCtx.OtaFirmwareRepo.FindLatestVersion(device.Model, device.DeviceType)
	if err != nil {
		fmt.Printf("[OTA] Error querying firmware: %v\n", err)
		result.ErrorMessage = fmt.Sprintf("查询固件版本失败: %v", err)
		return result
	}

	if latestFirmware == nil {
		fmt.Printf("[OTA] No firmware found for model=%s, deviceType=%s\n", device.Model, device.DeviceType)
		return result
	}

	fmt.Printf("[OTA] Found firmware: version=%s, published=%d\n", latestFirmware.Version, latestFirmware.Published)

	cmp := util.CompareVersion(latestFirmware.Version, device.FirmwareVer)
	if cmp <= 0 {
		fmt.Printf("[OTA] No upgrade needed: latest=%s, current=%s\n", latestFirmware.Version, device.FirmwareVer)
		return result
	}

	if latestFirmware.GrayScale == model.GrayScaleYes {
		if !util.IsInGrayScale(device.DeviceSn, latestFirmware.GrayPercent) {
			fmt.Printf("[OTA] Device not in gray scale: percent=%d\n", latestFirmware.GrayPercent)
			return result
		}
		result.GrayScale = true
		result.GrayPercent = latestFirmware.GrayPercent
	}

	result.NeedUpgrade = true
	result.LatestVer = latestFirmware.Version
	result.UpgradeType = latestFirmware.UpgradeType
	result.Changelog = latestFirmware.Changelog
	result.DownloadURL = latestFirmware.DownloadURL
	result.FileSize = latestFirmware.FileSize
	result.FileMD5 = latestFirmware.FileMD5
	result.ForceUpgrade = (latestFirmware.UpgradeType == model.UpgradeTypeForce)

	if result.ForceUpgrade {
		result.UpgradeTips = "发现强制升级版本 " + latestFirmware.Version + "，请尽快升级以保证设备正常运行"
	} else {
		result.UpgradeTips = "发现可选升级版本 " + latestFirmware.Version + "，建议升级以获得更好的使用体验"
	}

	fmt.Printf("[OTA] Upgrade available: %s -> %s\n", device.FirmwareVer, latestFirmware.Version)
	return result
}
