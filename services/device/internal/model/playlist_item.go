package model

import (
	"time"
)

// PlaylistItem 播放列表项
type PlaylistItem struct {
	ID         int64     `db:"id"`
	PlaylistID int64     `db:"playlist_id"`
	AudioID    int64     `db:"audio_id"`
	SortOrder  int32     `db:"sort_order"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
	DeletedAt  *time.Time `db:"deleted_at"`
}