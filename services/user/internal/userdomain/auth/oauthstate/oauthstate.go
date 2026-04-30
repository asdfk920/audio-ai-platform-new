package oauthstate

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/redis/go-redis/v9"
)

// ErrInvalidState state 缺失、不匹配或已过期。
var ErrInvalidState = errors.New("oauth state invalid")

const keyPrefix = "oauth:state:"
const ttl = 10 * time.Minute

func redisKey(state string) string {
	return keyPrefix + state
}

// New 生成随机 state 并写入 Redis，值为 platform（wechat|google）。
func New(ctx context.Context, platform string) (state string, err error) {
	platform = strings.TrimSpace(strings.ToLower(platform))
	if platform == "" {
		return "", errors.New("empty platform")
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state = hex.EncodeToString(b)
	if err := redisx.Set(ctx, redisKey(state), platform, ttl); err != nil {
		return "", err
	}
	return state, nil
}

// Validate 校验 state 是否与期望平台一致，成功则删除键（一次性）。
func Validate(ctx context.Context, state, expectPlatform string) error {
	state = strings.TrimSpace(state)
	expectPlatform = strings.TrimSpace(strings.ToLower(expectPlatform))
	if state == "" {
		return ErrInvalidState
	}
	got, err := redisx.Get(ctx, redisKey(state))
	if err != nil {
		if err == redis.Nil {
			return ErrInvalidState
		}
		return err
	}
	if strings.ToLower(strings.TrimSpace(got)) != expectPlatform {
		return ErrInvalidState
	}
	_ = redisx.Del(ctx, redisKey(state))
	return nil
}
