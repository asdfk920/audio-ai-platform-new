package jwtx

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

// ParseAccessToken 解析并校验 access JWT（类型安全）。供非 go-zero 中间件场景或单元测试使用。
// go-zero 默认仍用 MapClaims 解析；本函数用于需要结构化 Claims 的调用方。
func ParseAccessToken(secret, tokenString string) (*AccessClaims, error) {
	if secret == "" {
		return nil, errors.New("JWT 密钥不能为空")
	}
	if tokenString == "" {
		return nil, errors.New("令牌不能为空")
	}

	var claims AccessClaims
	tok, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("签名算法无效: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("解析令牌失败: %w", err)
	}
	if !tok.Valid {
		return nil, errors.New("令牌无效")
	}
	if err := claims.Valid(); err != nil {
		return nil, err
	}
	return &claims, nil
}
