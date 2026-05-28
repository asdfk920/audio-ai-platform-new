package shadowv2

import (
	"encoding/json"
	"time"
)

const (
	ShadowKeyPrefix = "device:shadow:"
	DefaultVersion  = int64(1)
)

type DeviceStatus string

const (
	StatusOnline   DeviceStatus = "online"
	StatusOffline  DeviceStatus = "offline"
	StatusAbnormal DeviceStatus = "abnormal"
)

type ShadowFields struct {
	Reported   string
	Desired    string
	Version    string
	UpdateTime string
	Status     string
}

func GetShadowFields() ShadowFields {
	return ShadowFields{
		Reported:   "reported",
		Desired:    "desired",
		Version:    "version",
		UpdateTime: "update_time",
		Status:     "status",
	}
}

func BuildShadowKey(deviceSN string) string {
	return ShadowKeyPrefix + deviceSN
}

type DeviceShadow struct {
	DeviceSN   string                 `json:"device_sn"`
	Reported   map[string]interface{} `json:"reported"`
	Desired    map[string]interface{} `json:"desired"`
	Version    int64                  `json:"version"`
	UpdateTime int64                  `json:"update_time"`
	Status     DeviceStatus           `json:"status"`
}

type InitShadowReq struct {
	DeviceSN string `json:"device_sn" validate:"required"`
}

type InitShadowResp struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type UpdateReportedReq struct {
	DeviceSN string                 `json:"device_sn" validate:"required"`
	Reported map[string]interface{} `json:"reported" validate:"required"`
}

type UpdateDesiredReq struct {
	DeviceSN string                 `json:"device_sn" validate:"required"`
	Desired  map[string]interface{} `json:"desired" validate:"required"`
}

type UpdateResp struct {
	DeviceSN   string                 `json:"device_sn"`
	Reported   map[string]interface{} `json:"reported"`
	Desired    map[string]interface{} `json:"desired"`
	Version    int64                  `json:"version"`
	UpdateTime int64                  `json:"update_time"`
	Status     DeviceStatus           `json:"status"`
}

type QueryShadowReq struct {
	DeviceSN string `json:"device_sn" validate:"required"`
}

type QueryShadowResp struct {
	DeviceSN   string                 `json:"device_sn"`
	Reported   map[string]interface{} `json:"reported"`
	Desired    map[string]interface{} `json:"desired"`
	Version    int64                  `json:"version"`
	UpdateTime int64                  `json:"update_time"`
	Status     DeviceStatus           `json:"status"`
	Message    string                 `json:"message,omitempty"`
}

type UpdateStatusReq struct {
	DeviceSN string       `json:"device_sn" validate:"required"`
	Status   DeviceStatus `json:"status" validate:"required,oneof=online offline abnormal"`
}

type CASUpdateReq struct {
	DeviceSN      string                 `json:"device_sn" validate:"required"`
	ExpectVersion int64                  `json:"expect_version" validate:"required,min=1"`
	Reported      map[string]interface{} `json:"reported,omitempty"`
	Desired       map[string]interface{} `json:"desired,omitempty"`
	Status        *DeviceStatus          `json:"status,omitempty"`
}

type CASUpdateResp struct {
	Success bool          `json:"success"`
	Message string        `json:"message"`
	Shadow  *DeviceShadow `json:"shadow,omitempty"`
	Error   string        `json:"error,omitempty"`
}

func (s *DeviceShadow) ToMap() map[string]string {
	reportedJSON, _ := json.Marshal(s.Reported)
	desiredJSON, _ := json.Marshal(s.Desired)
	fields := GetShadowFields()

	return map[string]string{
		fields.Reported:   string(reportedJSON),
		fields.Desired:    string(desiredJSON),
		fields.Version:    formatInt64(s.Version),
		fields.UpdateTime: formatInt64(s.UpdateTime),
		fields.Status:     string(s.Status),
	}
}

func FromMap(deviceSN string, m map[string]string) *DeviceShadow {
	fields := GetShadowFields()
	reported := make(map[string]interface{})
	desired := make(map[string]interface{})

	if v, ok := m[fields.Reported]; ok && v != "" {
		_ = json.Unmarshal([]byte(v), &reported)
	}
	// 兼容 shadow v1 Hash 字段 reported_json / desired_json（批量上报、shadowsvc 同步）
	if len(reported) == 0 {
		if v, ok := m["reported_json"]; ok && v != "" {
			_ = json.Unmarshal([]byte(v), &reported)
		}
	}

	if v, ok := m[fields.Desired]; ok && v != "" {
		_ = json.Unmarshal([]byte(v), &desired)
	}
	if len(desired) == 0 {
		if v, ok := m["desired_json"]; ok && v != "" {
			_ = json.Unmarshal([]byte(v), &desired)
		}
	}

	version := parseInt64(m[fields.Version])
	updateTime := parseInt64(m[fields.UpdateTime])
	status := DeviceStatus(m[fields.Status])

	if status == "" {
		status = StatusOffline
	}

	if version == 0 {
		version = DefaultVersion
	}

	if updateTime == 0 {
		updateTime = time.Now().Unix()
	}

	return &DeviceShadow{
		DeviceSN:   deviceSN,
		Reported:   reported,
		Desired:    desired,
		Version:    version,
		UpdateTime: updateTime,
		Status:     status,
	}
}

func formatInt64(n int64) string {
	return time.Unix(n, 0).Format("2006-01-02T15:04:05Z")
}

func parseInt64(s string) int64 {
	var t time.Time
	err := t.UnmarshalText([]byte(s))
	if err != nil {
		return 0
	}
	return t.Unix()
}
