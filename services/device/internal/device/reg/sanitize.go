// Package reg 设备注册域：SN 规范化、MAC 格式、黑白名单与注册侧限流辅助。
package reg

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jacklau/audio-ai-platform/common/errorx"
)

const defaultVersion = "unknown"

// SanitizeVisibleString 去除首尾空白，并拒绝控制类与格式类不可见字符（防隐蔽注入）。
func SanitizeVisibleString(field, s string, maxRunes int) (string, error) {
	s = strings.TrimSpace(s)
	if maxRunes > 0 && utf8.RuneCountInString(s) > maxRunes {
		return "", errorx.NewCodeError(errorx.CodeInvalidParam, field+"过长")
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return "", errorx.NewCodeError(errorx.CodeInvalidParam, field+"包含非法不可见字符")
		}
	}
	return s, nil
}

// NormalizeSN 统一大写存储与查询，减少大小写重复注册。
func NormalizeSN(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// DefaultVersion 固件 / 硬件版本缺省时占位。
func DefaultVersion(s string) string {
	if strings.TrimSpace(s) == "" {
		return defaultVersion
	}
	return strings.TrimSpace(s)
}
