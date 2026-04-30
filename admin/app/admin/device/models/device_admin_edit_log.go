package models

import "time"

// DeviceAdminEditLog 管理员修改设备扩展信息审计
type DeviceAdminEditLog struct {
	Id            int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DeviceId      int64     `gorm:"column:device_id" json:"deviceId"`
	Sn            string    `gorm:"column:sn;size:64" json:"sn"`
	AdminUserId   int64     `gorm:"column:admin_user_id" json:"adminUserId"`
	AdminAccount  string    `gorm:"column:admin_account;size:128" json:"adminAccount"`
	BeforeData    string    `gorm:"column:before_data;type:jsonb" json:"beforeData"`
	AfterData     string    `gorm:"column:after_data;type:jsonb" json:"afterData"`
	UpdatedFields string    `gorm:"column:updated_fields;type:jsonb" json:"updatedFields"`
	IpAddress     string    `gorm:"column:ip_address;size:64" json:"ipAddress"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"createdAt"`
}

func (DeviceAdminEditLog) TableName() string {
	return "device_admin_edit_log"
}
