// Package shadow 设备影子域：Redis Hash、在线 TTL、可选 Set 与 Pub/Sub、事件载荷。
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

// ReportPatch 设备上报的可选字段（空值表示不修改已有影子）。
type ReportPatch struct {
	Online          *bool
	RunState        string
	FirmwareVersion string
	Battery         *int32
	IP              string
	DeviceID        int64
	ProductKey      string
	Mac             string
}

// Snapshot 影子对外视图（与 GET /status 对齐）。
type Snapshot struct {
	Online          string
	SN              string
	ProductKey      string
	Mac             string
	RunState        string
	FirmwareVersion string
	Battery         int32
	LastUpdateAt    int64
	IP              string
	DeviceID        int64
}

func parseSnapshot(m map[string]string, onlineKeyPresent bool, onlineVal string) Snapshot {
	var s Snapshot
	if onlineKeyPresent && onlineVal == "1" {
		s.Online = "online"
	} else {
		s.Online = "offline"
	}
	s.SN = m[FSN]
	s.ProductKey = m[FProductKey]
	s.Mac = m[FMac]
	s.RunState = m[FRunState]
	if s.RunState == "" {
		s.RunState = "unknown"
	}
	s.FirmwareVersion = m[FFirmwareVersion]
	s.Battery = atoi32(m[FBattery])
	s.LastUpdateAt = atoi64(m[FUpdatedMs])
	if s.LastUpdateAt == 0 {
		s.LastUpdateAt = atoi64(m[FLastActiveMs])
	}
	s.IP = m[FIP]
	s.DeviceID = atoi64(m[FDeviceID])
	return s
}

func atoi32(v string) int32 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	n, _ := strconv.ParseInt(v, 10, 32)
	return int32(n)
}

func atoi64(v string) int64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	n, _ := strconv.ParseInt(v, 10, 64)
	return n
}

func mergePatch(snNorm string, prev map[string]string, p ReportPatch, nowMs int64, clientIP string) map[string]string {
	out := make(map[string]string)
	for k, v := range prev {
		out[k] = v
	}
	if out[FSN] == "" {
		out[FSN] = snNorm
	}
	if p.DeviceID > 0 {
		out[FDeviceID] = strconv.FormatInt(p.DeviceID, 10)
	}
	if strings.TrimSpace(p.ProductKey) != "" {
		out[FProductKey] = strings.TrimSpace(p.ProductKey)
	}
	if strings.TrimSpace(p.Mac) != "" {
		out[FMac] = strings.TrimSpace(p.Mac)
	}
	if p.Online != nil {
		if *p.Online {
			out[FOnline] = "1"
		} else {
			out[FOnline] = "0"
		}
	} else if out[FOnline] == "" {
		out[FOnline] = "1"
	}
	if rs := strings.TrimSpace(p.RunState); rs != "" {
		out[FRunState] = rs
	} else if out[FRunState] == "" {
		out[FRunState] = "normal"
	}
	if fw := strings.TrimSpace(p.FirmwareVersion); fw != "" {
		out[FFirmwareVersion] = fw
	}
	if p.Battery != nil {
		out[FBattery] = strconv.FormatInt(int64(*p.Battery), 10)
	}
	ip := strings.TrimSpace(p.IP)
	if ip == "" {
		ip = strings.TrimSpace(clientIP)
	}
	if ip != "" {
		if len(ip) > 45 {
			ip = ip[:45]
		}
		out[FIP] = ip
	}
	out[FLastActiveMs] = strconv.FormatInt(nowMs, 10)
	out[FUpdatedMs] = strconv.FormatInt(nowMs, 10)
	return out
}

func shadowChanged(prev, next map[string]string) bool {
	keys := []string{FOnline, FRunState, FFirmwareVersion, FBattery}
	for _, k := range keys {
		if strings.TrimSpace(prev[k]) != strings.TrimSpace(next[k]) {
			return true
		}
	}
	return false
}

// ApplyReport 更新影子 Hash、在线 Key TTL、可选写入在线集合与 Pub/Sub。
func ApplyReport(ctx context.Context, rdb *redis.Client, ttl time.Duration, enableOnlineSet, pubEnabled bool, pubPrefix string,
	snNorm string, patch ReportPatch, clientIP string,
) (changed bool, snap Snapshot, err error) {
	if rdb == nil {
		return false, Snapshot{}, fmt.Errorf("redis client nil")
	}
	nowMs := time.Now().UnixMilli()
	sk, okKey := ShadowKey(snNorm), OnlineKey(snNorm)

	prev, err := rdb.HGetAll(ctx, sk).Result()
	if err != nil {
		return false, Snapshot{}, err
	}

	next := mergePatch(snNorm, prev, patch, nowMs, clientIP)
	changed = len(prev) == 0 || shadowChanged(prev, next)

	pipe := rdb.Pipeline()
	pipe.HSet(ctx, sk, stringMapToAny(next))
	pipe.Expire(ctx, sk, ttl)
	online := strings.TrimSpace(next[FOnline]) == "1"
	if online {
		pipe.Set(ctx, okKey, "1", ttl)
		if enableOnlineSet {
			pipe.SAdd(ctx, KeyOnlineAll, snNorm)
		}
	} else {
		pipe.Set(ctx, okKey, "0", ttl)
		if enableOnlineSet {
			pipe.SRem(ctx, KeyOnlineAll, snNorm)
		}
	}
	if _, err = pipe.Exec(ctx); err != nil {
		return false, Snapshot{}, err
	}

	if pubEnabled && changed {
		bat := atoi32(next[FBattery])
		payload, _ := json.Marshal(map[string]any{
			"type":             "device_shadow_changed",
			"sn":               snNorm,
			"at":               nowMs,
			"last_update_at":   nowMs,
			"online":           next[FOnline] == "1",
			"run_state":        strings.TrimSpace(next[FRunState]),
			"firmware_version": strings.TrimSpace(next[FFirmwareVersion]),
			"battery":          bat,
			"ip":               strings.TrimSpace(next[FIP]),
			"device_id":        atoi64(next[FDeviceID]),
		})
		_ = rdb.Publish(ctx, EventChannel(pubPrefix, snNorm), payload).Err()
	}

	vOnline, _ := rdb.Get(ctx, okKey).Result()
	snap = parseSnapshot(next, true, vOnline)
	return changed, snap, nil
}

func stringMapToAny(m map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// LoadSnapshot 读取影子；onlineKey 缺失时视为离线。
func LoadSnapshot(ctx context.Context, rdb *redis.Client, snNorm string) (Snapshot, error) {
	if rdb == nil {
		return Snapshot{}, fmt.Errorf("redis client nil")
	}
	sk, okKey := ShadowKey(snNorm), OnlineKey(snNorm)
	m, err := rdb.HGetAll(ctx, sk).Result()
	if err != nil {
		return Snapshot{}, err
	}
	vOnline, err := rdb.Get(ctx, okKey).Result()
	if err == redis.Nil {
		return parseSnapshot(m, false, "0"), nil
	}
	if err != nil {
		return Snapshot{}, err
	}
	return parseSnapshot(m, true, vOnline), nil
}

// SeedShadow 将数据库基准写入 Redis（离线缓存）。
func SeedShadow(ctx context.Context, rdb *redis.Client, ttl time.Duration, snNorm, pk, mac, fw, ip string, deviceID int64) error {
	if rdb == nil {
		return fmt.Errorf("redis client nil")
	}
	nowMs := time.Now().UnixMilli()
	sk, okKey := ShadowKey(snNorm), OnlineKey(snNorm)
	fields := map[string]string{
		FSN:              snNorm,
		FProductKey:      pk,
		FMac:             mac,
		FOnline:          "0",
		FRunState:        "unknown",
		FFirmwareVersion: fw,
		FBattery:         "0",
		FLastActiveMs:    strconv.FormatInt(nowMs, 10),
		FUpdatedMs:       strconv.FormatInt(nowMs, 10),
		FDeviceID:        strconv.FormatInt(deviceID, 10),
	}
	if ip != "" {
		if len(ip) > 45 {
			ip = ip[:45]
		}
		fields[FIP] = ip
	}
	pipe := rdb.Pipeline()
	pipe.HSet(ctx, sk, stringMapToAny(fields))
	pipe.Expire(ctx, sk, ttl)
	pipe.Set(ctx, okKey, "0", ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// LastActivityEpochMs 影子中用于判断在线的最近时间戳（毫秒），优先 updated_ms。
func LastActivityEpochMs(m map[string]string) int64 {
	if m == nil {
		return 0
	}
	if ms := atoi64(m[FUpdatedMs]); ms > 0 {
		return ms
	}
	return atoi64(m[FLastActiveMs])
}
