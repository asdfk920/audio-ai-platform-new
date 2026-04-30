package jwtx

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// SignAccessOptions 签发 access JWT 的入参（iat 由包内 time.Now 生成，避免调用方传错时间）。
type SignAccessOptions struct {
	Secret     string
	TTLSeconds int64
	UserID     int64
	Issuer     string // 为空则用 DefaultIssuer
	Subject    string // 为空则用 DefaultSubject
}

// SignAccessToken 使用 HS256 签发 access token：含 iss/sub/exp/iat/jti、userId、token_type。
// jti 为 16 字节随机十六进制，便于日志关联与后续黑名单扩展。
func SignAccessToken(opt SignAccessOptions) (string, error) {
	if opt.Secret == "" {
		return "", errors.New("JWT 密钥不能为空")
	}
	if opt.TTLSeconds <= 0 {
		return "", errors.New("access token 有效期须大于 0 秒")
	}
	if opt.UserID <= 0 {
		return "", errors.New("用户 ID 无效")
	}

	iss := opt.Issuer
	if iss == "" {
		iss = DefaultIssuer
	}
	sub := opt.Subject
	if sub == "" {
		sub = DefaultSubject
	}

	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", fmt.Errorf("生成令牌唯一标识失败: %w", err)
	}
	jti := hex.EncodeToString(jtiBytes)

	now := time.Now()
	claims := AccessClaims{
		UserID:    opt.UserID,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    iss,
			Subject:   sub,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(opt.TTLSeconds) * time.Second)),
			ID:        jti,
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	return t.SignedString([]byte(opt.Secret))
}

// SignAccessTokenHS256 兼容旧签名：iat 参数已忽略，由包内自动生成。
// 请优先使用 SignAccessToken。
func SignAccessTokenHS256(secret string, _ /* iatUnix ignored */, ttlSeconds, userID int64) (string, error) {
	return SignAccessToken(SignAccessOptions{
		Secret:     secret,
		TTLSeconds: ttlSeconds,
		UserID:     userID,
	})
}
