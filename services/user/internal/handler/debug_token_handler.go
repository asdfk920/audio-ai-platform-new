package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/pkg/jwtx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
)

// DebugTokenHandler 调试Token有效性（仅开发环境使用）
// GET /api/v1/debug/token?token=xxx
// 用途：诊断Token为什么无效，返回详细的解析结果
//
// ⚠️ 注意：此接口仅供开发调试，生产环境应该禁用！
func DebugTokenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, map[string]any{
				"code": 400,
				"msg":  "请提供token参数或在Authorization头中携带",
				"data": map[string]any{
					"usage": "GET /api/v1/debug/token?token=your_jwt_token",
				},
			})
			return
		}

		debugInfo := analyzeToken(svcCtx, token)

		httpx.OkJsonCtx(r.Context(), w, debugInfo)
	}
}

// TokenDebugInfo Token调试信息结构
type TokenDebugInfo struct {
	Valid        bool   `json:"valid"`                // 是否有效
	Error        string `json:"error,omitempty"`      // 错误信息（如果有）
	ErrorType    string `json:"error_type,omitempty"` // 错误类型
	RawToken     string `json:"raw_token"`            // 原始token（截断显示）
	TokenLength  int    `json:"token_length"`         // token长度
	LooksLikeJWT bool   `json:"looks_like_jwt"`       // 是否像JWT格式
	DotCount     int    `json:"dot_count"`            // 点号数量（标准JWT应该有2个）

	// 解析后的Claims信息
	Claims    map[string]interface{} `json:"claims,omitempty"`     // Claims内容（脱敏）
	UserID    int64                  `json:"user_id,omitempty"`    // 用户ID
	TokenType string                 `json:"token_type,omitempty"` // Token类型
	JTI       string                 `json:"jti,omitempty"`        // JWT ID
	Issuer    string                 `json:"issuer,omitempty"`     // 签发者
	Subject   string                 `json:"subject,omitempty"`    // 主题
	IssuedAt  *int64                 `json:"issued_at,omitempty"`  // 签发时间（Unix时间戳）
	ExpiresAt *int64                 `json:"expires_at,omitempty"` // 过期时间（Unix时间戳）
	Expired   bool                   `json:"expired"`              // 是否已过期
	ExpiresIn int64                  `json:"expires_in,omitempty"` // 剩余有效时间（秒）

	// 配置信息
	Config map[string]interface{} `json:"config,omitempty"` // 相关配置
}

// analyzeToken 分析Token的有效性
func analyzeToken(svcCtx *svc.ServiceContext, token string) *TokenDebugInfo {
	info := &TokenDebugInfo{
		Valid:        false,
		RawToken:     maskToken(token),
		TokenLength:  len(token),
		LooksLikeJWT: strings.Count(token, ".") == 2,
		DotCount:     strings.Count(token, "."),
		Config: map[string]interface{}{
			"access_secret":  maskSecret(svcCtx.Config.Auth.AccessSecret),
			"access_expire":  svcCtx.Config.Auth.AccessExpire,
			"expire_seconds": svcCtx.Config.Auth.AccessExpire,
		},
	}

	if !info.LooksLikeJWT {
		info.Error = "Token格式错误：不是标准的JWT格式（应该包含2个点号）"
		info.ErrorType = "INVALID_FORMAT"
		return info
	}

	secret := svcCtx.Config.Auth.AccessSecret
	if secret == "" {
		info.Error = "配置错误：AccessSecret为空"
		info.ErrorType = "CONFIG_ERROR"
		return info
	}

	claims, parseErr := jwtx.ParseAccessToken(secret, token)
	if parseErr != nil {
		errMsg := parseErr.Error()
		info.Error = errMsg
		info.RawToken = token // 解析失败时显示完整token便于排查

		if strings.Contains(errMsg, "expired") {
			info.ErrorType = "TOKEN_EXPIRED"
			info.Expired = true
		} else if strings.Contains(errMsg, "signature") {
			info.ErrorType = "INVALID_SIGNATURE"
		} else if strings.Contains(errMsg, "token type") {
			info.ErrorType = "INVALID_TOKEN_TYPE"
		} else {
			info.ErrorType = "PARSE_ERROR"
		}

		return info
	}

	if claims == nil {
		info.Error = "解析成功但Claims为空"
		info.ErrorType = "NIL_CLAIMS"
		return info
	}

	now := getCurrentTimestamp()

	info.Valid = true
	info.UserID = claims.UserID
	info.TokenType = claims.TokenType
	info.JTI = claims.ID
	info.Issuer = claims.Issuer
	info.Subject = claims.Subject

	if claims.IssuedAt != nil {
		iat := claims.IssuedAt.Time.Unix()
		info.IssuedAt = &iat
	}

	if claims.ExpiresAt != nil {
		exp := claims.ExpiresAt.Time.Unix()
		info.ExpiresAt = &exp
		info.Expired = now > exp
		if !info.Expired {
			info.ExpiresIn = exp - now
		}
	}

	info.Claims = map[string]interface{}{
		"userId":     claims.UserID,
		"token_type": claims.TokenType,
		"jti":        claims.ID,
		"issuer":     claims.Issuer,
		"subject":    claims.Subject,
	}

	return info
}

// maskToken 对Token进行脱敏处理（只显示前20位和后10位）
func maskToken(token string) string {
	if len(token) <= 30 {
		return token
	}
	return token[:20] + "..." + token[len(token)-10:]
}

// maskSecret 对Secret进行脱敏处理
func maskSecret(secret string) string {
	if secret == "" {
		return "(empty)"
	}
	if len(secret) <= 8 {
		return "***"
	}
	return secret[:4] + "****" + secret[len(secret)-4:]
}

// getCurrentTimestamp 获取当前Unix时间戳
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
