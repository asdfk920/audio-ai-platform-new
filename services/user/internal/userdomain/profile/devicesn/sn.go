package devicesn

import (
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
)

// ValidateAndNormalize 去首尾空格后校验 SN；合法则返回规范化串（原样，仅 TrimSpace）。
func ValidateAndNormalize(sn string) (string, error) {
	sn = strings.TrimSpace(sn)
	if sn == "" {
		return "", errorx.NewCodeError(errorx.CodeInvalidParam, "设备序列号不能为空")
	}
	return sn, nil
}
