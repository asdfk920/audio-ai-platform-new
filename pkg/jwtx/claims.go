package jwtx

import (
	"errors"

	"github.com/golang-jwt/jwt/v4"
)

const (
	// TokenTypeAccess 访问令牌，仅用于 API 携带（与 refresh 随机串区分，防混用）。
	TokenTypeAccess = "access"
	// TokenTypeRefresh 若未来 refresh 也 JWT 化时使用；当前 refresh 为 Redis 随机串，此常量预留。
	TokenTypeRefresh = "refresh"
)

// 默认标准声明，符合 JWT 常见实践；可按 SignAccessOptions 覆盖。
const (
	DefaultIssuer  = "user-service"
	DefaultSubject = "user-auth"
)

// AccessClaims 结构化 access token 声明（替代 MapClaims），便于类型安全解析与审计。
// JSON 仍含 userId，与 go-zero rest.WithJwt 中间件（MapClaims 注入 ctx）兼容。
type AccessClaims struct {
	UserID    int64  `json:"userId"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// Valid 校验过期等标准字段，并确保 token_type 为 access。
func (c *AccessClaims) Valid() error {
	if err := c.RegisteredClaims.Valid(); err != nil {
		return err
	}
	if c.TokenType != "" && c.TokenType != TokenTypeAccess {
		return errors.New("token 类型无效，需要 access token")
	}
	return nil
}
