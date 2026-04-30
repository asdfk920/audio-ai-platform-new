package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
)

type PlaylistItemRepo struct {
	db *sql.DB
}

func NewPlaylistItemRepo(db *sql.DB) *PlaylistItemRepo {
	return &PlaylistItemRepo{db: db}
}

// FindByPlaylistId 根据播放列表ID查询所有音频项
func (r *PlaylistItemRepo) FindByPlaylistId(ctx context.Context, playlistId int64) ([]*model.PlaylistItem, error) {
	query := `
		SELECT 
			id, playlist_id, audio_id, sort_order, created_at, updated_at, deleted_at
		FROM playlist_item 
		WHERE playlist_id = $1 AND deleted_at IS NULL
		ORDER BY sort_order ASC, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, playlistId)
	if err != nil {
		return nil, fmt.Errorf("查询播放列表项失败: %v", err)
	}
	defer rows.Close()

	var items []*model.PlaylistItem
	for rows.Next() {
		var item model.PlaylistItem
		err := rows.Scan(
			&item.ID, &item.PlaylistID, &item.AudioID, &item.SortOrder,
			&item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描播放列表项失败: %v", err)
		}

		items = append(items, &item)
	}

	return items, nil
}

// AddItem 添加音频到播放列表
func (r *PlaylistItemRepo) AddItem(ctx context.Context, item *model.PlaylistItem) error {
	query := `
		INSERT INTO playlist_item (playlist_id, audio_id, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id
	`

	err := r.db.QueryRowContext(ctx, query, item.PlaylistID, item.AudioID, item.SortOrder).Scan(&item.ID)
	if err != nil {
		return fmt.Errorf("添加播放列表项失败: %v", err)
	}

	return nil
}

// RemoveItem 从播放列表移除音频
func (r *PlaylistItemRepo) RemoveItem(ctx context.Context, playlistId, audioId int64) error {
	query := `
		UPDATE playlist_item 
		SET deleted_at = NOW()
		WHERE playlist_id = $1 AND audio_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, playlistId, audioId)
	if err != nil {
		return fmt.Errorf("移除播放列表项失败: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("播放列表项不存在")
	}

	return nil
}