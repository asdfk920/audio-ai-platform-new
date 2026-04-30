package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
)

type AudioResourceRepo struct {
	db *sql.DB
}

func NewAudioResourceRepo(db *sql.DB) *AudioResourceRepo {
	return &AudioResourceRepo{db: db}
}

// FindByIds 批量查询音频资源
func (r *AudioResourceRepo) FindByIds(ctx context.Context, ids []int64) (map[int64]*model.AudioResource, error) {
	if len(ids) == 0 {
		return make(map[int64]*model.AudioResource), nil
	}

	// 构建IN查询参数
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT 
			id, title, duration, play_url, cover_url, status, created_at, updated_at, deleted_at,
			is_encrypted, audio_key, aasp_url
		FROM audio_resource 
		WHERE id IN (%s) AND status = $%d AND deleted_at IS NULL
	`, strings.Join(placeholders, ", "), len(ids)+1)

	args = append(args, model.AudioResourceStatusNormal)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("批量查询音频资源失败: %v", err)
	}
	defer rows.Close()

	audioMap := make(map[int64]*model.AudioResource)
	for rows.Next() {
		var audio model.AudioResource
		err := rows.Scan(
			&audio.ID, &audio.Title, &audio.Duration, &audio.PlayUrl, &audio.CoverUrl,
			&audio.Status, &audio.CreatedAt, &audio.UpdatedAt, &audio.DeletedAt,
			&audio.IsEncrypted, &audio.AudioKey, &audio.AaspUrl,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描音频资源失败: %v", err)
		}

		audioMap[audio.ID] = &audio
	}

	return audioMap, nil
}

// FindById 根据ID查询音频资源
func (r *AudioResourceRepo) FindById(ctx context.Context, id int64) (*model.AudioResource, error) {
	query := `
		SELECT 
			id, title, duration, play_url, cover_url, status, created_at, updated_at, deleted_at,
			is_encrypted, audio_key, aasp_url
		FROM audio_resource 
		WHERE id = $1 AND status = $2 AND deleted_at IS NULL
	`

	var audio model.AudioResource
	err := r.db.QueryRowContext(ctx, query, id, model.AudioResourceStatusNormal).Scan(
		&audio.ID, &audio.Title, &audio.Duration, &audio.PlayUrl, &audio.CoverUrl,
		&audio.Status, &audio.CreatedAt, &audio.UpdatedAt, &audio.DeletedAt,
		&audio.IsEncrypted, &audio.AudioKey, &audio.AaspUrl,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询音频资源失败: %v", err)
	}

	return &audio, nil
}

// Create 创建音频资源
func (r *AudioResourceRepo) Create(ctx context.Context, audio *model.AudioResource) error {
	query := `
		INSERT INTO audio_resource (title, duration, play_url, cover_url, status, is_encrypted, audio_key, aasp_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING id
	`

	err := r.db.QueryRowContext(ctx, query, audio.Title, audio.Duration, audio.PlayUrl, audio.CoverUrl, audio.Status, audio.IsEncrypted, audio.AudioKey, audio.AaspUrl).Scan(&audio.ID)
	if err != nil {
		return fmt.Errorf("创建音频资源失败: %v", err)
	}

	return nil
}

// Update 更新音频资源
func (r *AudioResourceRepo) Update(ctx context.Context, audio *model.AudioResource) error {
	query := `
		UPDATE audio_resource 
		SET title = $1, duration = $2, play_url = $3, cover_url = $4, status = $5, 
		    is_encrypted = $6, audio_key = $7, aasp_url = $8, updated_at = NOW()
		WHERE id = $9
	`

	_, err := r.db.ExecContext(ctx, query, audio.Title, audio.Duration, audio.PlayUrl, audio.CoverUrl, audio.Status, audio.IsEncrypted, audio.AudioKey, audio.AaspUrl, audio.ID)
	if err != nil {
		return fmt.Errorf("更新音频资源失败: %v", err)
	}

	return nil
}
