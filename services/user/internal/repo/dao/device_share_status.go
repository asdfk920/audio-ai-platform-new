package dao

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
)

// DeviceShareStatus 与库表 user_device_share.status 一致：文本枚举（迁移后），仍兼容历史 smallint 扫描。
type DeviceShareStatus string

const (
	DeviceShareStatusPending  DeviceShareStatus = "pending"
	DeviceShareStatusActive   DeviceShareStatus = "active"
	DeviceShareStatusRejected DeviceShareStatus = "rejected"
	DeviceShareStatusRevoked  DeviceShareStatus = "revoked"
	DeviceShareStatusExpired  DeviceShareStatus = "expired"
	DeviceShareStatusQuit     DeviceShareStatus = "quit"
)

func (s DeviceShareStatus) String() string { return string(s) }

func (s DeviceShareStatus) Value() (driver.Value, error) {
	// 写入库：优先兼容仍为 SMALLINT 的 status；VARCHAR 列通常会将整数转为如 '1'，Scan 侧已兼容。
	return int64(DeviceShareStatusToLegacyInt(s)), nil
}

func (s *DeviceShareStatus) Scan(src interface{}) error {
	if src == nil {
		*s = ""
		return nil
	}
	switch v := src.(type) {
	case int64:
		*s = deviceShareStatusFromInt(int(v))
	case int32:
		*s = deviceShareStatusFromInt(int(v))
	case int16:
		*s = deviceShareStatusFromInt(int(v))
	case []byte:
		return s.Scan(string(v))
	case string:
		*s = DeviceShareStatus(normalizeDeviceShareStatusString(v))
	default:
		return fmt.Errorf("device share status: scan unsupported type %T", v)
	}
	return nil
}

func deviceShareStatusFromInt(n int) DeviceShareStatus {
	switch n {
	case 0:
		return DeviceShareStatusPending
	case 1:
		return DeviceShareStatusActive
	case 2:
		return DeviceShareStatusRejected
	case 3:
		return DeviceShareStatusRevoked
	case 4:
		return DeviceShareStatusExpired
	case 5:
		return DeviceShareStatusQuit
	default:
		return DeviceShareStatus(strconv.Itoa(n))
	}
}

func normalizeDeviceShareStatusString(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return string(DeviceShareStatusPending)
	}
	switch v {
	case string(DeviceShareStatusPending), string(DeviceShareStatusActive),
		string(DeviceShareStatusRejected), string(DeviceShareStatusRevoked),
		string(DeviceShareStatusExpired), string(DeviceShareStatusQuit):
		return v
	case "0":
		return string(DeviceShareStatusPending)
	case "1":
		return string(DeviceShareStatusActive)
	case "2":
		return string(DeviceShareStatusRejected)
	case "3":
		return string(DeviceShareStatusRevoked)
	case "4":
		return string(DeviceShareStatusExpired)
	case "5":
		return string(DeviceShareStatusQuit)
	default:
		return v
	}
}

// DeviceShareStatusToLegacyInt 兼容前端/历史 JSON：0=pending … 5=quit。
func DeviceShareStatusToLegacyInt(s DeviceShareStatus) int16 {
	switch normalizeDeviceShareStatusString(string(s)) {
	case string(DeviceShareStatusPending):
		return 0
	case string(DeviceShareStatusActive):
		return 1
	case string(DeviceShareStatusRejected):
		return 2
	case string(DeviceShareStatusRevoked):
		return 3
	case string(DeviceShareStatusExpired):
		return 4
	case string(DeviceShareStatusQuit):
		return 5
	default:
		return 0
	}
}
