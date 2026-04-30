// Package loginsession 基于 Redis 中的 refresh_token 映射实现「当前登录会话」列表与吊销（与 login/refresh 的 key 约定一致）。
package loginsession

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/pkg/redisx"
	"github.com/redis/go-redis/v9"
)

// View 会话列表项（不向客户端暴露原始 refresh_token）。
type View struct {
	SessionID  string
	DeviceID   string
	DeviceName string
	Platform   string
	IP         string
	UserAgent  string
	CreatedAt  int64
	LastSeenAt int64
	Current    bool
}

func refreshTokenKey(token string) string {
	return fmt.Sprintf("user:refresh:%s", token)
}

func userRefreshTokenKey(userID int64) string {
	return fmt.Sprintf("user:%d:refresh", userID)
}

// PublicID 由 refresh_token 派生的不透明会话 ID（客户端用于 list/kick 对应）。
func PublicID(refreshToken string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(refreshToken)))
	return "rt_" + hex.EncodeToString(sum[:8])
}

// List 返回当前用户在 Redis 中的活跃 refresh 会话（当前实现为单会话：与登录/刷新轮换一致）。
func List(ctx context.Context, userID int64, _ int /* limit 预留多会话扩展 */) ([]View, error) {
	tok, err := redisx.Get(ctx, userRefreshTokenKey(userID))
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return nil, nil
	}
	now := time.Now().Unix()
	v := View{
		SessionID:  PublicID(tok),
		DeviceID:   "",
		DeviceName: "当前设备",
		Platform:   "",
		IP:         "",
		UserAgent:  "",
		CreatedAt:  now,
		LastSeenAt: now,
		Current:    true,
	}
	return []View{v}, nil
}

// RevokeByPublicID 吊销与 publicSessionID 对应的 refresh（须与当前 user:{id}:refresh 一致）。
func RevokeByPublicID(ctx context.Context, userID int64, publicSessionID string) error {
	publicSessionID = strings.TrimSpace(publicSessionID)
	if publicSessionID == "" {
		return fmt.Errorf("empty session id")
	}
	tok, err := redisx.Get(ctx, userRefreshTokenKey(userID))
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("会话不存在或已失效")
		}
		return err
	}
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return fmt.Errorf("会话不存在或已失效")
	}
	if PublicID(tok) != publicSessionID {
		return fmt.Errorf("会话不存在或无权操作")
	}
	_ = redisx.Del(ctx, refreshTokenKey(tok))
	_ = redisx.Del(ctx, userRefreshTokenKey(userID))
	return nil
}
