package dao

import "time"

// ContentCatalog 对应迁移 045 public.content
type ContentCatalog struct {
	ID          int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Title       string    `gorm:"column:title"`
	CoverURL    string    `gorm:"column:cover_url"`
	AudioURL    string    `gorm:"column:audio_url"`
	DurationSec int       `gorm:"column:duration_sec"`
	Artist      string    `gorm:"column:artist"`
	ArtistID    int64     `gorm:"column:artist_id"`
	CategoryID  *int64    `gorm:"column:category_id"`
	VipLevel    int16     `gorm:"column:vip_level"`
	SizeBytes   *int64    `gorm:"column:size_bytes"`
	Format      string    `gorm:"column:format"`
	SortOrder   int       `gorm:"column:sort_order"`
	Status      int16     `gorm:"column:status"`
	IsDeleted   int16     `gorm:"column:is_deleted;default:0"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (ContentCatalog) TableName() string {
	return "content"
}
