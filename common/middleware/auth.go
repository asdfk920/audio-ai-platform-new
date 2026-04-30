package middleware

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type AuthMiddleware struct {
	Secret string
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
	return &AuthMiddleware{
		Secret: secret,
	}
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: 实现 JWT 验证逻辑
		// 1. 从 Header 获取 token
		// 2. 验证 token
		// 3. 解析用户信息
		// 4. 将用户信息放入 context

		// 临时实现：直接放行
		next(w, r)
	}
}

func UnauthorizedResponse(w http.ResponseWriter, err error) {
	httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
		"code": 401,
		"msg":  "unauthorized",
		"data": nil,
	})
}
