package shadow

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// MarkOfflineAfterOnlineTTL 在 device:online:{sn} 因 TTL 过期后调用：影子标记离线、写回在线键为 0（短 TTL 与 Hash 一致）、可选 Pub/Sub。
func MarkOfflineAfterOnlineTTL(ctx context.Context, rdb *redis.Client, shadowTTL time.Duration, enableOnlineSet, pubEnabled bool, pubPrefix, snNorm string) error {
	if rdb == nil {
		return fmt.Errorf("redis client nil")
	}
	snNorm = strings.ToUpper(strings.TrimSpace(snNorm))
	if snNorm == "" {
		return nil
	}
	if shadowTTL <= 0 {
		shadowTTL = 60 * time.Second
	}
	nowMs := time.Now().UnixMilli()
	sk, okKey := ShadowKey(snNorm), OnlineKey(snNorm)

	prev, err := rdb.HGetAll(ctx, sk).Result()
	if err != nil {
		return err
	}
	next := make(map[string]string)
	for k, v := range prev {
		next[k] = v
	}
	if next[FSN] == "" {
		next[FSN] = snNorm
	}
	wasOnline := strings.TrimSpace(next[FOnline]) == "1"
	next[FOnline] = "0"
	next[FUpdatedMs] = strconv.FormatInt(nowMs, 10)
	next[FLastActiveMs] = strconv.FormatInt(nowMs, 10)
	if strings.TrimSpace(next[FRunState]) == "" {
		next[FRunState] = "unknown"
	}

	pipe := rdb.Pipeline()
	pipe.HSet(ctx, sk, stringMapToAny(next))
	pipe.Expire(ctx, sk, shadowTTL)
	pipe.Set(ctx, okKey, "0", shadowTTL)
	if enableOnlineSet {
		pipe.SRem(ctx, KeyOnlineAll, snNorm)
	}
	if _, err = pipe.Exec(ctx); err != nil {
		return err
	}

	if pubEnabled && wasOnline {
		bat := atoi32(next[FBattery])
		payload, _ := json.Marshal(map[string]any{
			"type":             "device_offline",
			"sn":               snNorm,
			"at":               nowMs,
			"last_update_at":   nowMs,
			"online":           false,
			"run_state":        strings.TrimSpace(next[FRunState]),
			"firmware_version": strings.TrimSpace(next[FFirmwareVersion]),
			"battery":          bat,
			"ip":               strings.TrimSpace(next[FIP]),
			"device_id":        atoi64(next[FDeviceID]),
			"reason":           "online_key_expired",
		})
		_ = rdb.Publish(ctx, EventChannel(pubPrefix, snNorm), payload).Err()
	}
	return nil
}
