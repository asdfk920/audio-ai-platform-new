package repository

import (
	"context"
	"time"

	"database/sql"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
)

// ContentDownloadStatsRepo 内容下载统计仓储
type ContentDownloadStatsRepo struct {
	db *sql.DB
}

func NewContentDownloadStatsRepo(db *sql.DB) *ContentDownloadStatsRepo {
	return &ContentDownloadStatsRepo{db: db}
}

// FindByContentID 根据内容ID查询统计记录
func (r *ContentDownloadStatsRepo) FindByContentID(ctx context.Context, contentID int64) (*model.ContentDownloadStats, error) {
	query := `SELECT id, content_id, total_downloads, today_downloads, week_downloads,
		last_status, last_error_msg, last_local_path, last_file_size, last_duration_ms,
		last_download_at, created_at, updated_at
		FROM content_download_stats WHERE content_id = $1`

	var record model.ContentDownloadStats
	err := r.db.QueryRowContext(ctx, query, contentID).Scan(
		&record.ID,
		&record.ContentID,
		&record.TotalDownloads,
		&record.TodayDownloads,
		&record.WeekDownloads,
		&record.LastStatus,
		&record.LastErrorMsg,
		&record.LastLocalPath,
		&record.LastFileSize,
		&record.LastDurationMs,
		&record.LastDownloadAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// IncrementDownloads 增加下载次数（如果不存在则创建，状态设为 downloading）
func (r *ContentDownloadStatsRepo) IncrementDownloads(ctx context.Context, contentID int64) error {
	now := time.Now()

	query := `INSERT INTO content_download_stats (content_id, total_downloads, today_downloads, week_downloads,
			last_status, last_download_at, created_at, updated_at)
		VALUES ($1, 1, 1, 1, 'downloading', $2, NOW(), NOW())
		ON CONFLICT (content_id) DO UPDATE SET
			total_downloads = content_download_stats.total_downloads + 1,
			today_downloads = content_download_stats.today_downloads + 1,
			week_downloads = content_download_stats.week_downloads + 1,
			last_status = 'downloading',
			last_error_msg = '',
			last_download_at = $2,
			updated_at = NOW()`

	_, err := r.db.ExecContext(ctx, query, contentID, now)
	return err
}

// UpdateDownloadResult 更新下载结果（回调接口使用）
// status: success 或 failed
func (r *ContentDownloadStatsRepo) UpdateDownloadResult(
	ctx context.Context,
	contentID int64,
	status string,
	errorMsg string,
	localPath string,
	fileSize int64,
	durationMs int64,
) error {
	query := `UPDATE content_download_stats SET
		last_status = $1,
		last_error_msg = COALESCE(NULLIF($2, ''), last_error_msg),
		last_local_path = COALESCE(NULLIF($3, ''), last_local_path),
		last_file_size = CASE WHEN $4 > 0 THEN $4 ELSE last_file_size END,
		last_duration_ms = CASE WHEN $5 > 0 THEN $5 ELSE last_duration_ms END,
		updated_at = NOW()
	WHERE content_id = $6`

	_, err := r.db.ExecContext(ctx, query, status, errorMsg, localPath, fileSize, durationMs, contentID)
	return err
}
