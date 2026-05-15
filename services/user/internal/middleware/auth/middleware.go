// Package auth JWT 接入鉴权与登出黑名单（middleware/auth）。
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/jwtx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/logger"
)

type contextKey string

const userIdKey contextKey = "userId"

// Middleware 校验 Authorization 中 access JWT 的 jti 是否被登出拉黑（Login.JWTBlacklistDisabled=true 时跳过）。
// 执行流程：
// 1. 提取并验证Authorization头格式
// 2. 解析JWT Token，提取claims（包含jti, userId等）
// 3. 检查Token是否在黑名单中
// 4. 如果在黑名单中，返回401 "登录已失效"
// 5. 如果Token无效或格式错误，返回401 "请先登录"
func Middleware(secret string, disabled bool) rest.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if disabled {
				next(w, r)
				return
			}
			if secret == "" {
				next(w, r)
				return
			}

			tok, err := extractToken(r)
			if err != nil {
				logger.AgentNDJSON("H4", "auth.Middleware:extractFailed", "token extraction failed", map[string]any{
					"path":  r.URL.Path,
					"error": err.Error(),
				})
				httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, map[string]any{
					"code": errorx.CodeTokenInvalid,
					"msg":  "请先登录",
				})
				return
			}

			claims, parseErr := jwtx.ParseAccessToken(secret, tok)

			logger.AgentNDJSON("H4", "auth.Middleware:afterParse", "parse access + blacklist gate", map[string]any{
				"path":         r.URL.Path,
				"tokenLen":     len(tok),
				"looksLikeJWT": strings.Count(tok, ".") == 2,
				"parseErrType": logger.ErrType(parseErr),
				"parseErrSnippet": func() string {
					if parseErr != nil {
						snip := parseErr.Error()
						if len(snip) > 180 {
							snip = snip[:180]
						}
						return snip
					}
					return ""
				}(),
				"claimsNil": claims == nil,
				"jtiEmpty":  claims == nil || claims.ID == "",
				"userId": func() int64 {
					if claims == nil {
						return 0
					}
					return claims.UserID
				}(),
				"expUnix": func() int64 {
					if claims == nil || claims.ExpiresAt == nil {
						return 0
					}
					return claims.ExpiresAt.Time.Unix()
				}(),
				"iatUnix": func() int64 {
					if claims == nil || claims.IssuedAt == nil {
						return 0
					}
					return claims.IssuedAt.Time.Unix()
				}(),
			})

			if parseErr != nil || claims == nil || claims.ID == "" {
				logger.AgentNDJSON("H4", "auth.Middleware:invalidToken", "token invalid or missing jti", map[string]any{
					"path":     r.URL.Path,
					"hasError": parseErr != nil,
					"error": func() string {
						if parseErr != nil {
							return parseErr.Error()
						}
						return ""
					}(),
					"claimsNil": claims == nil,
					"jtiEmpty":  claims == nil || claims.ID == "",
				})

				msg := "Token 无效，请重新登录"
				if parseErr != nil {
					if strings.Contains(parseErr.Error(), "expired") {
						msg = "Token 已过期，请重新登录"
					} else if strings.Contains(parseErr.Error(), "token type") {
						msg = "Token 类型无效，请重新登录"
					}
				}

				httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, map[string]any{
					"code": errorx.CodeTokenInvalid,
					"msg":  msg,
				})
				return
			}

			blocked, err := IsBlacklisted(r.Context(), claims.ID)
			if err != nil {
				logger.AgentNDJSON("H5", "auth.Middleware:blacklistCheckError", "redis error when checking blacklist", map[string]any{
					"path":  r.URL.Path,
					"jti":   claims.ID,
					"error": err.Error(),
				})
				httpx.WriteJsonCtx(r.Context(), w, http.StatusInternalServerError, map[string]any{
					"code": errorx.CodeRedisError,
					"msg":  "服务暂不可用，请稍后重试",
				})
				return
			}

			if blocked {
				logger.AgentNDJSON("H5", "auth.Middleware:blacklisted", "jti blacklisted - token revoked", map[string]any{
					"path":   r.URL.Path,
					"jti":    claims.ID,
					"userId": claims.UserID,
				})
				httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, map[string]any{
					"code": errorx.CodeTokenInvalid,
					"msg":  "登录已失效，请重新登录",
				})
				return
			}

			ctx := context.WithValue(r.Context(), userIdKey, claims.UserID)
			next(w, r.WithContext(ctx))
		}
	}
}

func extractToken(r *http.Request) (string, error) {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	const pfx = "Bearer "

	if !strings.HasPrefix(auth, pfx) {
		return "", fmt.Errorf("missing or invalid Authorization header format")
	}

	tok := strings.TrimSpace(strings.TrimPrefix(auth, pfx))
	if tok == "" {
		return "", fmt.Errorf("empty token after Bearer prefix")
	}

	return tok, nil
}
