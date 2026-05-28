package logic

import (
	"github.com/jacklau/audio-ai-platform/services/device/internal/util"
)

// validateReqSN 统一设备序列号校验（仅非空，无格式限制）
func validateReqSN(sn string) error {
	return util.ValidateDeviceSN(sn)
}
