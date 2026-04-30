package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
)

type ContentFileRepo struct {
	db *sql.DB
}

func NewContentFileRepo(db *sql.DB) *ContentFileRepo {
	return &ContentFileRepo{db: db}
}

func (r *ContentFileRepo) Create(ctx context.Context, file *model.ContentFile) error {
	query := `
		INSERT INTO content_files (url, key_hash, file_type, original_name, original_size, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id
	`

	err := r.db.QueryRowContext(ctx, query, file.URL, file.KeyHash, file.FileType, file.OriginalName, file.OriginalSize).Scan(&file.ID)
	if err != nil {
		return fmt.Errorf("创建文件记录失败: %v", err)
	}

	return nil
}

func (r *ContentFileRepo) FindById(ctx context.Context, id int64) (*model.ContentFile, error) {
	query := `
		SELECT id, url, key_hash, file_type, original_name, original_size, created_at
		FROM content_files
		WHERE id = $1
	`

	var file model.ContentFile
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&file.ID, &file.URL, &file.KeyHash, &file.FileType,
		&file.OriginalName, &file.OriginalSize, &file.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询文件记录失败: %v", err)
	}

	return &file, nil
}

type ContentFileIDs struct {
	AudioID sql.NullInt64
	CoverID sql.NullInt64
}

func (r *ContentFileRepo) GetContentFileIDs(ctx context.Context, contentID int64) (*ContentFileIDs, error) {
	query := `
		SELECT audio_id, cover_id FROM content WHERE id = $1
	`

	var result ContentFileIDs
	err := r.db.QueryRowContext(ctx, query, contentID).Scan(&result.AudioID, &result.CoverID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询内容文件关联失败: %v", err)
	}

	return &result, nil
}
