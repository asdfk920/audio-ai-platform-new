package dao

import "time"

// ContentDownloadStats 内容下载统计表（对应设备服务的 content_download_stats 表）
// 用于记录每个内容的下载统计数据和最后一次下载状态
type ContentDownloadStats struct {
	ID             int64      `gorm:"column:id;primaryKey" json:"id"`
	ContentID      int64      `gorm:"column:content_id;uniqueIndex;not null" json:"content_id"`
	TotalDownloads int64      `gorm:"column:total_downloads;default:0" json:"total_downloads"`
	TodayDownloads int64      `gorm:"column:today_downloads;default:0" json:"today_downloads"`
	WeekDownloads  int64      `gorm:"column:week_downloads;default:0" json:"week_downloads"`
	LastStatus     string     `gorm:"column:last_status;default:''" json:"last_status"`
	LastErrorMsg   string     `gorm:"column:last_error_msg;default:''" json:"last_error_msg"`
	LastLocalPath  string     `gorm:"column:last_local_path;default:''" json:"last_local_path"`
	LastFileSize   int64      `gorm:"column:last_file_size;default:0" json:"last_file_size"`
	LastDurationMs int64      `gorm:"column:last_duration_ms;default:0" json:"last_duration_ms"`
	LastDownloadAt *time.Time `gorm:"column:last_download_at" json:"last_download_at,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ContentDownloadStats) TableName() string {
	return "content_download_stats"
}
