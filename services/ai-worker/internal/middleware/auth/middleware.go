// Package auth JWT 认证中间件
package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/jwtx"
)

// Middleware JWT 认证中间件
// 验证 Authorization Header 中的 Bearer Token
func Middleware(secret string) rest.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// 如果 secret 为空，跳过认证
			if secret == "" {
				next(w, r)
				return
			}

			// 获取 Authorization Header
			auth := strings.TrimSpace(r.Header.Get("Authorization"))
			const prefix = "Bearer "

			// 验证 Bearer Token 格式
			if !strings.HasPrefix(auth, prefix) {
				httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, map[string]any{
					"code": errorx.CodeTokenInvalid,
					"msg":  "未授权访问",
				})
				return
			}

			token := strings.TrimSpace(strings.TrimPrefix(auth, prefix))
			if token == "" {
				httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, map[string]any{
					"code": errorx.CodeTokenInvalid,
					"msg":  "Token 不能为空",
				})
				return
			}

			// 解析 Token
			claims, err := jwtx.ParseAccessToken(secret, token)
			if err != nil || claims == nil || claims.UserID == 0 {
				httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, map[string]any{
					"code": errorx.CodeTokenInvalid,
					"msg":  "Token 无效或已过期",
				})
				return
			}

			// 将用户 ID 注入到 Context 中
			ctx := context.WithValue(r.Context(), "userId", claims.UserID)
			*r = *r.WithContext(ctx)

			// 继续处理请求
			next(w, r)
		}
	}
}

// GetUserIDFromContext 从 Context 中获取用户 ID
func GetUserIDFromContext(ctx context.Context) int64 {
	if userID, ok := ctx.Value("userId").(int64); ok {
		return userID
	}
	return 0
}
