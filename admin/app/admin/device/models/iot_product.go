package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// IotProduct 后台产品线（与 ota_firmware.product_key / device.product_key 对齐）
type IotProduct struct {
	Id            int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	ProductKey    string         `gorm:"column:product_key;size:64;not null;comment:产品标识" json:"productKey"`
	Name          string         `gorm:"column:name;size:128;not null" json:"name"`
	Category      string         `gorm:"column:category;size:64" json:"category"`
	Description   string         `gorm:"column:description;type:text" json:"description"`
	Communication json.RawMessage `gorm:"column:communication;type:jsonb" json:"communication"`
	DeviceType    string         `gorm:"column:device_type;size:64" json:"deviceType"`
	Status        string         `gorm:"column:status;size:32;not null;default:draft" json:"status"` // draft|published|disabled
	CreateBy      int64          `gorm:"column:created_by" json:"-"`
	UpdateBy      int64          `gorm:"column:updated_by" json:"-"`
	CreatedAt     time.Time      `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt     time.Time      `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (IotProduct) TableName() string {
	return "iot_product"
}
