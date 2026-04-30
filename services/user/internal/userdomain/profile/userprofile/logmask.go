package userprofile

import (
	"net/url"
	"strings"
)

// MaskNicknameAudit 审计日志中的昵称脱敏。
func MaskNicknameAudit(p *string) string {
	if p == nil {
		return "<null>"
	}
	v := *p
	if v == "" {
		return ""
	}
	r := []rune(v)
	if len(r) <= 1 {
		return "*"
	}
	if len(r) == 2 {
		return string(r[0]) + "*"
	}
	return string(r[0]) + "***" + string(r[len(r)-1])
}

// MaskAvatarAudit 审计中仅保留 URL 的 scheme + host，避免日志存完整长链。
func MaskAvatarAudit(p *string) string {
	if p == nil {
		return "<null>"
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return ""
	}
	u, err := url.Parse(v)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "(unparsed)"
	}
	return u.Scheme + "://" + u.Host + "/…"
}
