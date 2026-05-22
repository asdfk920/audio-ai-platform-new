package model

import "time"

// UserDownload 用户下载记录表
type UserDownload struct {
	ID           int64     `gorm:"column:id;primaryKey" json:"id"`
	UserID       int64     `gorm:"column:user_id;not null;index" json:"user_id"`
	ContentID    int64     `gorm:"column:content_id;not null;index" json:"content_id"`
	ContentTitle string    `gorm:"column:content_title;not null" json:"content_title"`
	FileURL      string    `gorm:"column:file_url" json:"file_url"`
	Status       string    `gorm:"column:status;default:'downloading'" json:"status"` // downloading/success/failed
	DownloadTime time.Time `gorm:"column:download_time" json:"download_time"`
}

func (UserDownload) TableName() string {
	return "user_downloads"
}
