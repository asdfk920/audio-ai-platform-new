// Package redisexpire 监听 Redis 键过期事件（device:online:{sn}），驱动影子离线与应用层通知。
package redisexpire

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/config"
	"github.com/jacklau/audio-ai-platform/services/device/internal/device/reg"
	"github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
	"github.com/jacklau/audio-ai-platform/services/device/internal/repo"
	"github.com/jacklau/audio-ai-platform/services/device/internal/statuspersist"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// StartOnlineKeyExpiryListener 使用独立 Redis 连接订阅 __keyevent@db__:expired。
// 需在 redis.conf 中启用 notify-keyspace-events 包含 E（或 Ex），否则收不到事件。
func StartOnlineKeyExpiryListener(ctx context.Context, sub *redis.Client, db *sql.DB, rdb *redis.Client, persist *statuspersist.Pool, c config.Config) {
	if sub == nil || rdb == nil || !c.RedisKeyspace.Enabled {
		return
	}
	dbi := c.Redis.DB
	if dbi < 0 {
		dbi = 0
	}
	channel := fmt.Sprintf("__keyevent@%d__:expired", dbi)
	log := logx.WithContext(ctx)
	go func() {
		pubsub := sub.Subscribe(ctx, channel)
		defer func() { _ = pubsub.Close() }()
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok || msg == nil {
					return
				}
				key := strings.TrimSpace(msg.Payload)
				if key == "" {
					continue
				}
				if !strings.HasPrefix(key, shadow.KeyPrefixOnline) || key == shadow.KeyOnlineAll {
					continue
				}
				snNorm := strings.ToUpper(strings.TrimPrefix(key, shadow.KeyPrefixOnline))
				if len(snNorm) < 8 {
					continue
				}
				handleOnlineExpired(context.Background(), log, db, rdb, persist, c, snNorm)
			}
		}
	}()
	log.Infof("redis keyspace listener subscribed: %s (device:online:* expiry → shadow offline)", channel)
}

func handleOnlineExpired(ctx context.Context, log logx.Logger, db *sql.DB, rdb *redis.Client, persist *statuspersist.Pool, c config.Config, snNorm string) {
	cfg := c.DeviceShadow
	ttlSec := cfg.HeartbeatTTLSeconds
	if ttlSec <= 0 {
		ttlSec = 60
	}
	shadowTTL := time.Duration(ttlSec) * time.Second
	prefix := cfg.EventChannelPrefix
	if strings.TrimSpace(prefix) == "" {
		prefix = "device:event:"
	}
	if err := shadow.MarkOfflineAfterOnlineTTL(ctx, rdb, shadowTTL, cfg.EnableOnlineSet, cfg.PubSubEnabled, prefix, snNorm); err != nil {
		log.Errorf("MarkOfflineAfterOnlineTTL sn=%s: %v", reg.MaskSN(snNorm), err)
		return
	}
	if persist == nil || db == nil {
		return
	}
	qctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	row, err := repo.GetDeviceForMQTTAuth(qctx, db, snNorm)
	if err != nil || row == nil {
		return
	}
	persist.Offer(statuspersist.Job{
		DeviceID:     row.ID,
		SN:           snNorm,
		RunState:     "",
		OnlineStatus: 0,
		Source:       "redis_expire",
	})
}
