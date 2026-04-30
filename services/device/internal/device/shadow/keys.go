package shadow

import (
	"strconv"
	"strings"
)

const (
	KeyPrefixShadow = "device:shadow:"
	KeyPrefixOnline = "device:online:"
	KeyOnlineAll    = "device:online:all"
	// KeyPrefixStatusNeg 空值缓存：绑定不存在时短期占位，减轻 DB 穿透（按 user+sn）。
	KeyPrefixStatusNeg = "device:status:neg:"
	KeySeedLock        = "device:shadow:seedlock:"
)

// StatusNegCacheKey 查询无绑定时的负缓存键。
func StatusNegCacheKey(userID int64, snNorm string) string {
	return KeyPrefixStatusNeg + strings.ToUpper(strings.TrimSpace(snNorm)) + ":" + strconv.FormatInt(userID, 10)
}

// SeedLockKey 回种影子互斥（多实例下减少重复 Seed）。
func SeedLockKey(snNorm string) string {
	return KeySeedLock + strings.ToUpper(strings.TrimSpace(snNorm))
}

// Hash 字段（全 string，便于 HGETALL）
const (
	FOnline          = "online"
	FSN              = "sn"
	FProductKey      = "product_key"
	FMac             = "mac"
	FRunState        = "run_state"
	FFirmwareVersion = "firmware_version"
	FBattery         = "battery"
	FLastActiveMs    = "last_active_ms"
	FUpdatedMs       = "updated_ms"
	FIP              = "ip"
	FDeviceID        = "device_id"
	FReportedJSON    = "reported_json"
	FDesiredJSON     = "desired_json"
	FDeltaJSON       = "delta_json"
	FMetadataJSON    = "metadata_json"
	FVersion         = "version"
)

func ShadowKey(snNorm string) string {
	return KeyPrefixShadow + strings.ToUpper(strings.TrimSpace(snNorm))
}

func OnlineKey(snNorm string) string {
	return KeyPrefixOnline + strings.ToUpper(strings.TrimSpace(snNorm))
}

func EventChannel(prefix, snNorm string) string {
	p := strings.TrimSpace(prefix)
	if p == "" {
		p = "device:event:"
	}
	return p + strings.ToUpper(strings.TrimSpace(snNorm))
}
