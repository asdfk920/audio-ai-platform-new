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

// RegisterInitInput 设备首次注册/激活时写入影子的初始字段
type RegisterInitInput struct {
	Sn              string
	DeviceID        int64
	ProductKey      string
	Mac             string
	FirmwareVersion string
	HardwareVersion string
	IP              string
}

// InitRegisterShadow 首次注册：写入 Redis Hash（device:shadow:{SN}）及在线 Key
func InitRegisterShadow(ctx context.Context, rdb *redis.Client, ttl time.Duration, in RegisterInitInput) error {
	if rdb == nil {
		return fmt.Errorf("redis client nil")
	}
	snNorm := strings.ToUpper(strings.TrimSpace(in.Sn))
	if snNorm == "" {
		return fmt.Errorf("sn empty")
	}
	if in.DeviceID <= 0 {
		return fmt.Errorf("device_id invalid")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	reported := map[string]interface{}{
		"run_state": "unknown",
	}
	if fw := strings.TrimSpace(in.FirmwareVersion); fw != "" {
		reported["firmware_version"] = fw
	}
	if hw := strings.TrimSpace(in.HardwareVersion); hw != "" {
		reported["hardware_version"] = hw
	}
	if mac := strings.TrimSpace(in.Mac); mac != "" {
		reported["mac"] = mac
	}
	if ip := strings.TrimSpace(in.IP); ip != "" {
		reported["ip"] = ip
	}
	reportedJSON, _ := json.Marshal(reported)
	desiredJSON := []byte("{}")
	deltaJSON := []byte("{}")
	metaJSON := []byte("{}")

	nowMs := time.Now().UnixMilli()
	pk := strings.TrimSpace(in.ProductKey)
	if pk == "" {
		pk = "default"
	}
	mac := strings.TrimSpace(in.Mac)
	fw := strings.TrimSpace(in.FirmwareVersion)
	ip := strings.TrimSpace(in.IP)
	if len(ip) > 45 {
		ip = ip[:45]
	}

	fields := map[string]string{
		FSN:              snNorm,
		FDeviceID:        strconv.FormatInt(in.DeviceID, 10),
		FProductKey:      pk,
		FMac:             mac,
		FOnline:          "0",
		FRunState:        "unknown",
		FFirmwareVersion: fw,
		FBattery:         "0",
		FReportedJSON:    string(reportedJSON),
		FDesiredJSON:     string(desiredJSON),
		FDeltaJSON:       string(deltaJSON),
		FMetadataJSON:    string(metaJSON),
		FVersion:         "0",
		FLastActiveMs:    strconv.FormatInt(nowMs, 10),
		FUpdatedMs:       strconv.FormatInt(nowMs, 10),
	}
	if ip != "" {
		fields[FIP] = ip
	}

	sk, okKey := ShadowKey(snNorm), OnlineKey(snNorm)
	pipe := rdb.Pipeline()
	pipe.HSet(ctx, sk, stringMapToAny(fields))
	pipe.Expire(ctx, sk, ttl)
	pipe.Set(ctx, okKey, "0", ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// RedisShadowEmpty 判断 Redis 影子 Hash 是否尚未初始化
func RedisShadowEmpty(ctx context.Context, rdb *redis.Client, snNorm string) (bool, error) {
	if rdb == nil {
		return true, fmt.Errorf("redis client nil")
	}
	h, err := rdb.HGetAll(ctx, ShadowKey(snNorm)).Result()
	if err != nil {
		return false, err
	}
	return len(h) == 0, nil
}
