// Package shadow 与设备微服务 Redis 键约定对齐（见 services/device/internal/device/shadow）。
package shadow

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	KeyPrefixShadow = "device:shadow:"
	KeyPrefixOnline = "device:online:"
	// Hash 字段（与微服务一致）
	FOnline          = "online"
	FSN              = "sn"
	FProductKey      = "product_key"
	FMac             = "mac"
	FRunState        = "run_state"
	FFirmwareVersion = "firmware_version"
	FBattery         = "battery"
	FVolume          = "volume"
	FLastActiveMs    = "last_active_ms"
	FUpdatedMs       = "updated_ms"
	FIP              = "ip"
	FDeviceID        = "device_id"
	// 扩展：JSON 字符串，供管理端与设备侧同步 desired/delta
	FReportedJSON = "reported_json"
	FDesiredJSON  = "desired_json"
	FDeltaJSON    = "delta_json"
)

var (
	rdbOnce sync.Once
	rdb     *redis.Client
)

// Client 返回 Redis 客户端；设置 DEVICE_REDIS_DISABLED=1 时不连接。
func Client() *redis.Client {
	rdbOnce.Do(func() {
		if strings.TrimSpace(os.Getenv("DEVICE_REDIS_DISABLED")) == "1" {
			return
		}
		addr := strings.TrimSpace(os.Getenv("DEVICE_REDIS_ADDR"))
		if addr == "" {
			addr = "127.0.0.1:6379"
		}
		pass := os.Getenv("DEVICE_REDIS_PASSWORD")
		db := 0
		if s := strings.TrimSpace(os.Getenv("DEVICE_REDIS_DB")); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				db = n
			}
		}
		rdb = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: pass,
			DB:       db,
		})
	})
	return rdb
}

func ShadowKey(snNorm string) string {
	return KeyPrefixShadow + strings.ToUpper(strings.TrimSpace(snNorm))
}

func OnlineKey(snNorm string) string {
	return KeyPrefixOnline + strings.ToUpper(strings.TrimSpace(snNorm))
}

func DefaultShadowTTL() time.Duration {
	if s := strings.TrimSpace(os.Getenv("DEVICE_SHADOW_TTL_SECONDS")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 30 * time.Minute
}
