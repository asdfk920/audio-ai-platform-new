package models

import "time"

// DeviceShadowProfile 设备影子规范化主表（1:1 device）
type DeviceShadowProfile struct {
	DeviceId        int64      `gorm:"column:device_id;primaryKey" json:"deviceId"`
	FirmwareVersion string     `gorm:"column:firmware_version;size:32" json:"firmwareVersion"`
	HardwareVersion string     `gorm:"column:hardware_version;size:32" json:"hardwareVersion"`
	OnlineStatus    int16      `gorm:"column:online_status;default:0" json:"onlineStatus"`
	OfflineAt       *time.Time `gorm:"column:offline_at" json:"offlineAt"`
	LastActiveAt    *time.Time `gorm:"column:last_active_at" json:"lastActiveAt"`
	FwUpgradedAt    *time.Time `gorm:"column:fw_upgraded_at" json:"fwUpgradedAt"`
	NetworkType     string     `gorm:"column:network_type;size:32" json:"networkType"`
	Rssi            *int       `gorm:"column:rssi" json:"rssi"`
	ProductKey      string     `gorm:"column:product_key;size:64" json:"productKey"`
	FirstOnlineAt   *time.Time `gorm:"column:first_online_at" json:"firstOnlineAt"`
	OfflineReason   string     `gorm:"column:offline_reason;size:128" json:"offlineReason"`
	ReportedAt      *time.Time `gorm:"column:reported_at" json:"reportedAt"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (DeviceShadowProfile) TableName() string {
	return "device_shadow_profile"
}

// DeviceShadowBattery 设备影子电量快照
type DeviceShadowBattery struct {
	DeviceId        int64     `gorm:"column:device_id;primaryKey" json:"deviceId"`
	MainPercent     *int16    `gorm:"column:main_percent" json:"mainPercent"`
	SpeakerPercent  *int16    `gorm:"column:speaker_percent" json:"speakerPercent"`
	Charging        int16     `gorm:"column:charging;default:0" json:"charging"`
	EstRemainingSec *int64    `gorm:"column:est_remaining_sec" json:"estRemainingSec"`
	LowThreshold    *int16    `gorm:"column:low_threshold" json:"lowThreshold"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (DeviceShadowBattery) TableName() string {
	return "device_shadow_battery"
}

// DeviceShadowLocation 设备影子位置快照
type DeviceShadowLocation struct {
	DeviceId       int64     `gorm:"column:device_id;primaryKey" json:"deviceId"`
	Latitude       *float64  `gorm:"column:latitude" json:"latitude"`
	Longitude      *float64  `gorm:"column:longitude" json:"longitude"`
	LocationMode   string    `gorm:"column:location_mode;size:32" json:"locationMode"`
	AccuracyM      *float64  `gorm:"column:accuracy_m" json:"accuracyM"`
	GeofenceStatus string    `gorm:"column:geofence_status;size:32" json:"geofenceStatus"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (DeviceShadowLocation) TableName() string {
	return "device_shadow_location"
}

// DeviceShadowConfig 设备影子配置（多行按 config_type）
type DeviceShadowConfig struct {
	Id         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DeviceId   int64     `gorm:"column:device_id" json:"deviceId"`
	ConfigType string    `gorm:"column:config_type;size:64" json:"configType"`
	Desired    string    `gorm:"column:desired;type:text" json:"desired"`
	Reported   string    `gorm:"column:reported;type:text" json:"reported"`
	SyncStatus int16     `gorm:"column:sync_status;default:0" json:"syncStatus"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (DeviceShadowConfig) TableName() string {
	return "device_shadow_config"
}
