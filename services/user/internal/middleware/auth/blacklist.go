package auth

import (
	"context"
	"time"

	"github.com/jacklau/audio-ai-platform/pkg/redisx"
)

const redisKeyPrefix = "jwt:blacklist:"

func redisKey(jti string) string {
	return redisKeyPrefix + jti
}

// Blacklist 将 access token 的 jti 加入黑名单，TTL 与 access 剩余有效期对齐（登出后当前 token 立即失效）。
func Blacklist(ctx context.Context, jti string, ttl time.Duration) error {
	if ctx == nil || jti == "" {
		return nil
	}
	if ttl < time.Second {
		ttl = time.Second
	}
	return redisx.Set(ctx, redisKey(jti), "1", ttl)
}

// IsBlacklisted 判断 jti 是否已被登出吊销。
func IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	if ctx == nil || jti == "" {
		return false, nil
	}
	n, err := redisx.Exists(ctx, redisKey(jti))
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
