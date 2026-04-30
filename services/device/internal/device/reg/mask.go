package reg

import "unicode/utf8"

// MaskSN 日志脱敏：保留头尾各 2 个可见字符。
func MaskSN(s string) string {
	return maskMiddle(s, 2, 2, 8)
}

// MaskMAC 日志脱敏。
func MaskMAC(s string) string {
	return maskMiddle(s, 2, 2, 6)
}

// MaskProductKey 日志脱敏。
func MaskProductKey(s string) string {
	return maskMiddle(s, 2, 2, 6)
}

func maskMiddle(s string, head, tail, minLen int) string {
	n := utf8.RuneCountInString(s)
	if n <= minLen {
		return "***"
	}
	rs := []rune(s)
	if n <= head+tail {
		return "***"
	}
	var b []rune
	b = append(b, rs[:head]...)
	b = append(b, []rune("***")...)
	b = append(b, rs[n-tail:]...)
	return string(b)
}
