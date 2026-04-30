package httpmeta

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP 解析客户端 IP（优先 X-Forwarded-For 首段、X-Real-IP，否则 RemoteAddr）。
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			if ip := strings.TrimSpace(parts[0]); ip != "" {
				return ip
			}
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	addr := strings.TrimSpace(r.RemoteAddr)
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
