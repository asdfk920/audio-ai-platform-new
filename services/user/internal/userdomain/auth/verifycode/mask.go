package verifycode

import (
	"strings"
)

// MaskTarget 日志脱敏：邮箱保留首尾少量字符，手机保留前 3 后 2 位，避免生产环境泄露完整账号。
// MaskTargetLoose 仅根据 target 形态脱敏（无 channel 时用于日志，按是否含 @ 区分邮箱/手机）。
func MaskTargetLoose(target string) string {
	t := strings.TrimSpace(target)
	if t == "" {
		return ""
	}
	if strings.Contains(t, "@") {
		return maskEmail(t)
	}
	return maskMobile(t)
}

func MaskTarget(channel, target string) string {
	t := strings.TrimSpace(target)
	if t == "" {
		return ""
	}
	switch channel {
	case ChannelEmail:
		return maskEmail(t)
	case ChannelMobile:
		return maskMobile(t)
	default:
		return "***"
	}
}

func maskEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at <= 0 || at >= len(email)-1 {
		return "***"
	}
	local, domain := email[:at], email[at+1:]
	if len(local) <= 1 {
		return "*" + "@" + maskRunes(domain, 1, 1)
	}
	return string(local[0]) + "***" + string(local[len(local)-1]) + "@" + maskRunes(domain, 1, 1)
}

func maskMobile(m string) string {
	r := []rune(m)
	if len(r) <= 5 {
		return "***"
	}
	return string(r[:3]) + "****" + string(r[len(r)-2:])
}

func maskRunes(s string, head, tail int) string {
	r := []rune(s)
	if len(r) <= head+tail {
		return "***"
	}
	return string(r[:head]) + "***" + string(r[len(r)-tail:])
}
