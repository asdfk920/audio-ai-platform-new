package model

import (
	"time"
)

// AudioResource 音频资源表
type AudioResource struct {
	ID          int64     `db:"id"`
	Title       string    `db:"title"`
	Duration    int32     `db:"duration"`
	PlayUrl     string    `db:"play_url"`
	CoverUrl    string    `db:"cover_url"`
	Status      int16     `db:"status"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
	// 私有格式字段
	IsEncrypted int16  `db:"is_encrypted"` // 是否加密: 0-否, 1-是
	AudioKey    string `db:"audio_key"`    // 音频加密密钥（仅创建时返回，不落库明文）
	AaspUrl     string `db:"aasp_url"`     // 私有格式文件URL
}

// AudioResourceStatus 音频资源状态常量
const (
	AudioResourceStatusNormal int16 = 1 // 正常
	AudioResourceStatusHidden int16 = 0 // 隐藏
)