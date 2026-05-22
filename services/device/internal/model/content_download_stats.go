package model

import "time"

// ContentDownloadStats 内容下载统计表
type ContentDownloadStats struct {
	ID        int64 `gorm:"column:id;primaryKey" json:"id"`
	ContentID int64 `gorm:"column:content_id;uniqueIndex;not null" json:"content_id"`

	// 下载次数统计
	TotalDownloads int64 `gorm:"column:total_downloads;default:0" json:"total_downloads"`
	TodayDownloads int64 `gorm:"column:today_downloads;default:0" json:"today_downloads"`
	WeekDownloads  int64 `gorm:"column:week_downloads;default:0" json:"week_downloads"`

	// 最后一次下载详情（用于回调更新）
	LastStatus     string     `gorm:"column:last_status;default:''" json:"last_status"`          // 最后状态: success/failed/downloading
	LastErrorMsg   string     `gorm:"column:last_error_msg;default:''" json:"last_error_msg"`    // 最后错误信息
	LastLocalPath  string     `gorm:"column:last_local_path;default:''" json:"last_local_path"`  // 最后本地路径
	LastFileSize   int64      `gorm:"column:last_file_size;default:0" json:"last_file_size"`     // 最后文件大小（字节）
	LastDurationMs int64      `gorm:"column:last_duration_ms;default:0" json:"last_duration_ms"` // 最后下载耗时（毫秒）
	LastDownloadAt *time.Time `gorm:"column:last_download_at" json:"last_download_at,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ContentDownloadStats) TableName() string {
	return "content_download_stats"
}
