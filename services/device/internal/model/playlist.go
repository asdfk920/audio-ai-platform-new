package model

import (
	"time"
)

// Playlist 播放列表主表
type Playlist struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Name      string    `db:"name"`
	Type      int16     `db:"type"`
	Status    int16     `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

// PlaylistType 播放列表类型常量
const (
	PlaylistTypeDefault int16 = 1 // 默认列表
	PlaylistTypeFavorite int16 = 2 // 收藏列表
	PlaylistTypeCustom  int16 = 3 // 自建列表
)

// PlaylistStatus 播放列表状态常量
const (
	PlaylistStatusNormal int16 = 1 // 正常
	PlaylistStatusHidden int16 = 0 // 隐藏
)