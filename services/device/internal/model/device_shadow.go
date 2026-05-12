package model

import (
	"encoding/json"
	"time"
)

const (
	ShadowOnlineStatusOffline int16 = 0 // 离线
	ShadowOnlineStatusOnline  int16 = 1 // 在线

	DisconnectTypeNormal     string = "normal"      // 正常断开
	DisconnectTypeWill       string = "will"        // WILL消息触发
	DisconnectTypeKeepalive  string = "keepalive_timeout" // 心跳超时
	DisconnectTypeKicked     string = "kicked"      // 被踢出
	DisconnectTypeUnknown    string = "unknown"     // 未知原因
)

type DeviceShadow struct {
	ID              int64           `db:"id"`
	Sn              string          `db:"sn"`
	OnlineStatus    int16           `db:"online_status"`
	FirmwareVersion string          `db:"firmware_version"`
	BatteryLevel    *int            `db:"battery_level"`
	LastReportTime  *time.Time      `db:"last_report_time"`
	LastCommandTime *time.Time      `db:"last_command_time"`
	Reported        json.RawMessage `db:"reported"`         // 设备上报的属性
	Desired         json.RawMessage `db:"desired"`          // 云端期望的属性
	Metadata        json.RawMessage `db:"metadata"`         // 影子元数据
	Version         int64           `db:"version"`          // 版本号（乐观锁）
	DisconnectType  string          `db:"disconnect_type"`  // 断开类型
	DisconnectAt    *time.Time      `db:"disconnect_at"`    // 断开时间
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

type WillMessagePayload struct {
	SN      string `json:"sn"`
	Status  string `json:"status"`  // "offline"
	Time    string `json:"time"`    // ISO8601时间戳
	Reason  string `json:"reason,omitempty"` // 可选：断开原因
}

func (s *DeviceShadow) IsOnline() bool {
	return s.OnlineStatus == ShadowOnlineStatusOnline
}

func (s *DeviceShadow) SetOnline() {
	s.OnlineStatus = ShadowOnlineStatusOnline
	s.DisconnectType = ""
	s.DisconnectAt = nil
}

func (s *DeviceShadow) SetOffline(disconnectType string) {
	now := time.Now()
	s.OnlineStatus = ShadowOnlineStatusOffline
	s.DisconnectType = disconnectType
	s.DisconnectAt = &now
}