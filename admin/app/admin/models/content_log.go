package models

import "time"

// ContentLog 内容操作审计（表 content_log，若库中无表需自行迁移）
type ContentLog struct {
	ID            int64     `json:"id" gorm:"primaryKey;autoIncrement;comment:主键"`
	ContentId     int64     `json:"content_id" gorm:"column:content_id;index;comment:内容ID"`
	Operator      string    `json:"operator" gorm:"size:64;comment:操作人"`
	Operation     string    `json:"operation" gorm:"size:32;comment:操作类型"`
	ChangedFields string    `json:"changed_fields" gorm:"type:text;comment:变更JSON"`
	IpAddress     string    `json:"ip_address" gorm:"size:64;comment:IP"`
	UserAgent     string    `json:"user_agent" gorm:"size:512;comment:UA"`
	CreateBy      int       `json:"create_by" gorm:"column:create_by;comment:操作人ID"`
	CreatedAt     time.Time `json:"created_at" gorm:"comment:创建时间"`
}

func (ContentLog) TableName() string {
	return "content_log"
}
