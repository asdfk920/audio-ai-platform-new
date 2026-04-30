package models

import "time"

// DeviceStatusLog 设备定时状态上报日志（对应 device_status_logs）
type DeviceStatusLog struct {
	Id                 int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DeviceId           int64     `gorm:"column:device_id" json:"deviceId"`
	Sn                 string    `gorm:"column:sn;size:64" json:"sn"`
	BatteryLevel       int       `gorm:"column:battery_level" json:"batteryLevel"`
	StorageUsed        int64     `gorm:"column:storage_used" json:"storageUsed"`
	StorageTotal       int64     `gorm:"column:storage_total" json:"storageTotal"`
	SpeakerCount       int       `gorm:"column:speaker_count" json:"speakerCount"`
	UwbX               *float64  `gorm:"column:uwb_x" json:"uwbX"`
	UwbY               *float64  `gorm:"column:uwb_y" json:"uwbY"`
	UwbZ               *float64  `gorm:"column:uwb_z" json:"uwbZ"`
	AcousticCalibrated int16     `gorm:"column:acoustic_calibrated" json:"acousticCalibrated"`
	AcousticOffset     *float64  `gorm:"column:acoustic_offset" json:"acousticOffset"`
	ReportType         string    `gorm:"column:report_type;size:16" json:"reportType"`
	ReportedAt         time.Time `gorm:"column:reported_at" json:"reportedAt"`
	CreatedAt          time.Time `gorm:"column:created_at" json:"createdAt"`
}

func (DeviceStatusLog) TableName() string {
	return "device_status_logs"
}
