package auth

import (
	"net/http"
	"strings"

	"github.com/jacklau/audio-ai-platform/pkg/jwtx"
)

// BearerContext 登录用户上下文
type BearerContext struct {
	UserID int64
}

// ParseBearer 解析 Authorization Bearer 或 query access_token
func ParseBearer(r *http.Request, secret string) BearerContext {
	tok := extractToken(r)
	if tok == "" || strings.TrimSpace(secret) == "" {
		return BearerContext{}
	}
	claims, err := jwtx.ParseAccessToken(secret, tok)
	if err != nil || claims == nil {
		return BearerContext{}
	}
	return BearerContext{UserID: claims.UserID}
}

func extractToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(authz) > 7 && strings.EqualFold(authz[:7], "Bearer ") {
		return strings.TrimSpace(authz[7:])
	}
	return strings.TrimSpace(r.URL.Query().Get("access_token"))
}
