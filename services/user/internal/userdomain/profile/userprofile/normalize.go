package userprofile

import (
	"strings"
	"unicode"
)

// NormalizeUpdateText 去掉首尾空白、全角空格转半角、剥离零宽字符与控制字符（昵称/头像 URL 前处理）。
func NormalizeUpdateText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\u3000", " ")
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\u200B', '\u200C', '\u200D', '\uFEFF':
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}
