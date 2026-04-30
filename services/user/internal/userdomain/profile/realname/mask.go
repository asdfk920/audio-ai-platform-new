package realname

import (
	"strings"
)

// MaskRealName 姓名脱敏：保留首尾字符，中间 *。
func MaskRealName(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return ""
	}
	r := []rune(s)
	if len(r) <= 1 {
		return "*"
	}
	if len(r) == 2 {
		return string(r[0]) + "*"
	}
	return string(r[0]) + strings.Repeat("*", len(r)-2) + string(r[len(r)-1])
}

// IDLast4 证件号后若干位（展示/审计）。
func IDLast4(id string) string {
	s := strings.TrimSpace(strings.ToUpper(id))
	if len(s) >= 4 {
		return s[len(s)-4:]
	}
	return "****"
}

// MaskIdNumber 身份证号脱敏：前 6 位 + **** + 后 4 位
func MaskIdNumber(id string) string {
	s := strings.TrimSpace(strings.ToUpper(id))
	if len(s) < 10 {
		return "******"
	}
	return s[:6] + "****" + s[len(s)-4:]
}
