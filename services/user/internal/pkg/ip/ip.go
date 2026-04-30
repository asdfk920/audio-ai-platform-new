package ip

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type ctxKey struct{}

// WithContext 将解析后的客户端 IP 写入 context，供 logic/dao 记录 register_ip / last_login_ip。
func WithContext(ctx context.Context, r *http.Request) context.Context {
	if r == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, FromRequest(r))
}

// FromContext 读取 WithContext 写入的 IP；未设置时返回空串。
func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v := ctx.Value(ctxKey{})
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// FromRequest 优先 X-Forwarded-For 首段、X-Real-IP，否则 RemoteAddr。
func FromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			if ip := strings.TrimSpace(parts[0]); ip != "" {
				return truncateIP(ip)
			}
		}
	}
	if xr := strings.TrimSpace(r.Header.Get("X-Real-IP")); xr != "" {
		return truncateIP(xr)
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return truncateIP(strings.TrimSpace(r.RemoteAddr))
	}
	return truncateIP(host)
}

func truncateIP(s string) string {
	if len(s) > 45 {
		return s[:45]
	}
	return s
}
