// Package auth JWT 接入鉴权与登出黑名单（middleware/auth）。
package auth

import (
	"net/http"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/jwtx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/logger"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Middleware 校验 Authorization 中 access JWT 的 jti 是否被登出拉黑（Login.JWTBlacklistDisabled=true 时跳过）。
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
			auth := strings.TrimSpace(r.Header.Get("Authorization"))
			const pfx = "Bearer "
			if !strings.HasPrefix(auth, pfx) {
				next(w, r)
				return
			}
			tok := strings.TrimSpace(strings.TrimPrefix(auth, pfx))
			if tok == "" {
				next(w, r)
				return
			}
			claims, err := jwtx.ParseAccessToken(secret, tok)
			// #region agent log
			parseErrSnip := ""
			if err != nil {
				parseErrSnip = err.Error()
				if len(parseErrSnip) > 180 {
					parseErrSnip = parseErrSnip[:180]
				}
			}
			logger.AgentNDJSON("H4", "auth.Middleware:afterParse", "parse access + blacklist gate", map[string]any{
				"path":            r.URL.Path,
				"tokenLen":        len(tok),
				"looksLikeJWT":    strings.Count(tok, ".") == 2,
				"parseErrType":    logger.ErrType(err),
				"parseErrSnippet": parseErrSnip,
				"claimsNil":       claims == nil,
				"jtiEmpty":        claims == nil || claims.ID == "",
				"userId":          func() int64 { if claims == nil { return 0 }; return claims.UserID }(),
				"expUnix":         func() int64 { if claims == nil || claims.ExpiresAt == nil { return 0 }; return claims.ExpiresAt.Time.Unix() }(),
				"iatUnix":         func() int64 { if claims == nil || claims.IssuedAt == nil { return 0 }; return claims.IssuedAt.Time.Unix() }(),
			})
			// #endregion
			if err != nil || claims == nil || claims.ID == "" {
				next(w, r)
				return
			}
			blocked, err := IsBlacklisted(r.Context(), claims.ID)
			if err != nil {
				httpx.WriteJsonCtx(r.Context(), w, http.StatusInternalServerError, map[string]any{
					"code": errorx.CodeRedisError,
					"msg":  "服务暂不可用",
				})
				return
			}
			if blocked {
				// #region agent log
				logger.AgentNDJSON("H5", "auth.Middleware:blacklist", "jti blacklisted", map[string]any{
					"path": r.URL.Path,
				})
				// #endregion
				httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, map[string]any{
					"code": errorx.CodeTokenInvalid,
					"msg":  "登录已失效",
				})
				return
			}
			next(w, r)
		}
	}
}
