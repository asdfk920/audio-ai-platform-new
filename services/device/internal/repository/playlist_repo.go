package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
)

type PlaylistRepo struct {
	db *sql.DB
}

func NewPlaylistRepo(db *sql.DB) *PlaylistRepo {
	return &PlaylistRepo{db: db}
}

// FindByUserId 根据用户ID和类型查询播放列表
func (r *PlaylistRepo) FindByUserId(ctx context.Context, userId int64, playlistType int16) (*model.Playlist, error) {
	query := `
		SELECT 
			id, user_id, name, type, status, created_at, updated_at, deleted_at
		FROM playlist 
		WHERE user_id = $1 AND type = $2 AND status = $3 AND deleted_at IS NULL
		LIMIT 1
	`

	var playlist model.Playlist
	err := r.db.QueryRowContext(ctx, query, userId, playlistType, model.PlaylistStatusNormal).Scan(
		&playlist.ID, &playlist.UserID, &playlist.Name, &playlist.Type, &playlist.Status,
		&playlist.CreatedAt, &playlist.UpdatedAt, &playlist.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询播放列表失败: %v", err)
	}

	return &playlist, nil
}

// Create 创建播放列表
func (r *PlaylistRepo) Create(ctx context.Context, playlist *model.Playlist) error {
	query := `
		INSERT INTO playlist (user_id, name, type, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id
	`

	err := r.db.QueryRowContext(ctx, query, playlist.UserID, playlist.Name, playlist.Type, playlist.Status).Scan(&playlist.ID)
	if err != nil {
		return fmt.Errorf("创建播放列表失败: %v", err)
	}

	return nil
}

// Update 更新播放列表
func (r *PlaylistRepo) Update(ctx context.Context, playlist *model.Playlist) error {
	query := `
		UPDATE playlist 
		SET name = $1, status = $2, updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, playlist.Name, playlist.Status, playlist.ID)
	if err != nil {
		return fmt.Errorf("更新播放列表失败: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("播放列表不存在或已被删除")
	}

	return nil
}