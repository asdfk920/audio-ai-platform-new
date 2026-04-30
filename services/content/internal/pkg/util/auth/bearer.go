package auth

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

// BearerContext 可选登录解析结果：无 Token / 非法时 UserID=0，按游客（vip=0）处理。
type BearerContext struct {
	UserID      int64
	VipLevel    int32 // 当 HasVipClaim 为 true 时来自 JWT；否则由业务查库
	HasVipClaim bool
}

// ParseBearer 解析 Authorization: Bearer，读取 userId 及可选的 vip_level / member_level 声明。
func ParseBearer(r *http.Request, accessSecret string) BearerContext {
	out := BearerContext{}
	if accessSecret == "" {
		return out
	}
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	const prefix = "Bearer "
	if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return out
	}
	tokenStr := strings.TrimSpace(h[len(prefix):])
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(accessSecret), nil
	})
	if err != nil || !token.Valid {
		return out
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return out
	}
	out.UserID = claimInt64(claims["userId"])
	if v, ok := claims["vip_level"]; ok {
		out.VipLevel = toInt32(v)
		out.HasVipClaim = true
		return out
	}
	if v, ok := claims["member_level"]; ok {
		out.VipLevel = toInt32(v)
		out.HasVipClaim = true
	}
	return out
}

func claimInt64(v interface{}) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case int:
		return int64(x)
	default:
		return 0
	}
}

func toInt32(v interface{}) int32 {
	switch x := v.(type) {
	case float64:
		return int32(x)
	case int64:
		return int32(x)
	case int:
		return int32(x)
	default:
		return 0
	}
}
