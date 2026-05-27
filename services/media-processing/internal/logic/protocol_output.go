package logic

import "strings"

// NormalizeOutputProtocol 将简写规整为 canonical 值。
func NormalizeOutputProtocol(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "chunked", "http":
		return "http_chunked"
	case "ws":
		return "websocket"
	default:
		return strings.ToLower(strings.TrimSpace(p))
	}
}

// ParseTruthyQuery 是否为真（嵌入式开关等）。
func ParseTruthyQuery(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
