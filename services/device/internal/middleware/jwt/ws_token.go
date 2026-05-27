package jwt

import (
	"net/http"
	"strings"
)

// ExtractAppWSBearerToken App 用户 WebSocket 握手时提取 JWT：
// 优先 Authorization: Bearer，其次 URL ?token= / ?access_token= / ?accessToken=
func ExtractAppWSBearerToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(h), "bearer ") && len(h) > 7 {
		t := strings.TrimSpace(h[7:])
		if t != "" {
			return t
		}
	}
	q := r.URL.Query()
	for _, key := range []string{"token", "access_token", "accessToken"} {
		if v := strings.TrimSpace(q.Get(key)); v != "" {
			return v
		}
	}
	return ""
}
