package devicesn

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/jacklau/audio-ai-platform/common/errorx"
)

// 与设备服务注册规则一致，避免用户端绑定与 device 表 sn 规则不一致。
var reSN = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{6,62}[A-Za-z0-9]$|^[A-Za-z0-9]{8}$`)

// ValidateAndNormalize 去首尾空格后校验 SN；合法则返回规范化串（原样，仅 TrimSpace）。
func ValidateAndNormalize(sn string) (string, error) {
	sn = strings.TrimSpace(sn)
	n := utf8.RuneCountInString(sn)
	if n < 8 || n > 64 {
		return "", errorx.NewCodeError(errorx.CodeDeviceSnInvalid, "设备序列号长度须为 8～64 个字符")
	}
	if !reSN.MatchString(sn) {
		return "", errorx.NewCodeError(errorx.CodeDeviceSnInvalid, "设备序列号仅允许字母、数字、下划线与连字符")
	}
	return sn, nil
}
