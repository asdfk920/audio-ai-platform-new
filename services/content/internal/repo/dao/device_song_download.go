package dao

import "time"

// DeviceSongDownload 设备歌曲下载记录
type DeviceSongDownload struct {
	ID            int64      `gorm:"column:id;primaryKey" json:"id"`
	TaskID        string     `gorm:"column:task_id;uniqueIndex;size:64;not null" json:"task_id"`
	UserID        int64      `gorm:"column:user_id;index;not null" json:"user_id"`
	DeviceSN      string     `gorm:"column:device_sn;index;size:16;not null" json:"device_sn"`
	ContentID     int64      `gorm:"column:content_id;index;not null" json:"content_id"`
	SongName      string     `gorm:"column:song_name;size:200" json:"song_name"`
	DownloadURL   string     `gorm:"column:download_url;type:text" json:"download_url"`
	Status        string     `gorm:"column:status;size:20;default:pending;index" json:"status"`
	Progress      float64    `gorm:"column:progress;default:0" json:"progress"`
	LocalPath     string     `gorm:"column:local_path;size:500" json:"local_path"`
	FileSize      int64      `gorm:"column:file_size;default:0" json:"file_size"`
	ErrorMsg      string     `gorm:"column:error_msg;type:text" json:"error_msg"`
	DurationMs    int64      `gorm:"column:duration_ms;default:0" json:"duration_ms"`
	InstructionID *int64     `gorm:"column:instruction_id" json:"instruction_id"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	FinishedAt    *time.Time `gorm:"column:finished_at" json:"finished_at"`
}

func (DeviceSongDownload) TableName() string {
	return "device_song_downloads"
}
