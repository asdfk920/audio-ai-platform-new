package jwt

import (
	"context"
	"net/http"
	"strings"

	"github.com/jacklau/audio-ai-platform/pkg/jwtx"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

type contextKey struct{}

// JwtMiddleware JWT 鉴权中间件，解析 token 并将 user_id 注入 context
func JwtMiddleware(secret string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
				httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
					"code": 500,
					"msg":  "JWT secret 未配置",
					"data": nil,
				})
				return
			}

			auth := strings.TrimSpace(r.Header.Get("Authorization"))
			const pfx = "Bearer "
			if !strings.HasPrefix(auth, pfx) {
				httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
					"code": 401,
					"msg":  "请先登录",
					"data": nil,
				})
				return
			}

			tok := strings.TrimSpace(strings.TrimPrefix(auth, pfx))
			if tok == "" {
				httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
					"code": 401,
					"msg":  "token 为空",
					"data": nil,
				})
				return
			}

			claims, err := jwtx.ParseAccessToken(secret, tok)
			if err != nil || claims == nil {
				logx.Errorf("JWT 解析失败: %v", err)
				httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
					"code": 401,
					"msg":  "token 无效或已过期",
					"data": nil,
				})
				return
			}

			if claims.UserID <= 0 {
				httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
					"code": 401,
					"msg":  "token 中缺少用户信息",
					"data": nil,
				})
				return
			}

			ctx := context.WithValue(r.Context(), contextKey{}, claims.UserID)
			next(w, r.WithContext(ctx))
		}
	}
}

// GetUserIdFromContext 从 context 中获取 user_id
func GetUserIdFromContext(ctx context.Context) (int64, bool) {
	userId, ok := ctx.Value(contextKey{}).(int64)
	return userId, ok
}
