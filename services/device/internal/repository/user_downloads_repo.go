package repository

import (
	"context"
	"time"

	"database/sql"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
)

// UserDownloadsRepo 用户下载记录仓储
type UserDownloadsRepo struct {
	db *sql.DB
}

func NewUserDownloadsRepo(db *sql.DB) *UserDownloadsRepo {
	return &UserDownloadsRepo{db: db}
}

// Create 创建下载记录（状态：downloading）
func (r *UserDownloadsRepo) Create(ctx context.Context, record *model.UserDownload) error {
	query := `INSERT INTO user_downloads (user_id, content_id, content_title, file_url, status, download_time)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query,
		record.UserID,
		record.ContentID,
		record.ContentTitle,
		record.FileURL,
		record.Status,
		time.Now(),
	)
	return err
}

// UpdateStatus 更新下载状态（回调使用）
func (r *UserDownloadsRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	query := `UPDATE user_downloads SET status = $1, download_time = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

// FindByContentIDAndUserID 根据内容ID和用户ID查询下载记录
func (r *UserDownloadsRepo) FindByContentIDAndUserID(ctx context.Context, contentID int64, userID int64) (*model.UserDownload, error) {
	query := `SELECT id, user_id, content_id, content_title, file_url, status, download_time
		FROM user_downloads WHERE content_id = $1 AND user_id = $2 ORDER BY id DESC LIMIT 1`

	var record model.UserDownload
	err := r.db.QueryRowContext(ctx, query, contentID, userID).Scan(
		&record.ID,
		&record.UserID,
		&record.ContentID,
		&record.ContentTitle,
		&record.FileURL,
		&record.Status,
		&record.DownloadTime,
	)
	if err != nil {
		return nil, err
	}
	return &record, nil
}
