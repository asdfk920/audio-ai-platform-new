package repository

import (
	"context"
	"time"

	"database/sql"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
)

// DeviceSongDownloadRepo 设备歌曲下载记录仓储
type DeviceSongDownloadRepo struct {
	db *sql.DB
}

func NewDeviceSongDownloadRepo(db *sql.DB) *DeviceSongDownloadRepo {
	return &DeviceSongDownloadRepo{db: db}
}

// Create 创建下载记录（下发指令后预存一条「下载中」记录；instruction_id 可选）
func (r *DeviceSongDownloadRepo) Create(ctx context.Context, record *model.DeviceSongDownload) error {
	var instr sql.NullInt64
	if record.InstructionID != nil {
		instr = sql.NullInt64{Valid: true, Int64: *record.InstructionID}
	}

	query := `INSERT INTO device_song_downloads 
		(task_id, user_id, device_sn, content_id, song_name, download_url, status, progress, instruction_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())`

	_, err := r.db.ExecContext(ctx, query,
		record.TaskID,
		record.UserID,
		record.DeviceSN,
		record.ContentID,
		record.SongName,
		record.DownloadURL,
		record.Status,
		record.Progress,
		instr,
	)
	return err
}

// FindByTaskID 根据任务ID查询
func (r *DeviceSongDownloadRepo) FindByTaskID(ctx context.Context, taskID string) (*model.DeviceSongDownload, error) {
	query := `SELECT id, task_id, user_id, device_sn, content_id, song_name, download_url, 
		status, progress, local_path, file_size, error_msg, duration_ms, instruction_id, 
		created_at, updated_at, finished_at 
		FROM device_song_downloads WHERE task_id = $1`
	
	var record model.DeviceSongDownload
	var instr sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, taskID).Scan(
		&record.ID, &record.TaskID, &record.UserID, &record.DeviceSN, &record.ContentID,
		&record.SongName, &record.DownloadURL, &record.Status, &record.Progress,
		&record.LocalPath, &record.FileSize, &record.ErrorMsg, &record.DurationMs,
		&instr, &record.CreatedAt, &record.UpdatedAt, &record.FinishedAt,
	)
	if err != nil {
		return nil, err
	}
	if instr.Valid {
		v := instr.Int64
		record.InstructionID = &v
	}
	return &record, nil
}

// UpdateStatus 更新状态和进度
func (r *DeviceSongDownloadRepo) UpdateStatus(ctx context.Context, taskID string, status string, progress float64) error {
	query := `UPDATE device_song_downloads SET status = $1, progress = $2, updated_at = NOW() WHERE task_id = $3`
	_, err := r.db.ExecContext(ctx, query, status, progress, taskID)
	return err
}

// UpdateResult 更新下载结果（成功或失败）
func (r *DeviceSongDownloadRepo) UpdateResult(ctx context.Context, taskID string, status, errorMsg, localPath string, fileSize int64, durationMs int64) error {
	now := time.Now()
	query := `UPDATE device_song_downloads SET 
		status = $1, 
		progress = 100, 
		error_msg = $2, 
		local_path = $3, 
		file_size = $4, 
		duration_ms = $5, 
		updated_at = $6`
	
	if status == "success" || status == "failed" {
		query += `, finished_at = $6`
	}
	query += ` WHERE task_id = $7`
	
	_, err := r.db.ExecContext(ctx, query, status, errorMsg, localPath, fileSize, durationMs, now, taskID)
	return err
}

// UpdateInstructionID 更新指令ID
func (r *DeviceSongDownloadRepo) UpdateInstructionID(ctx context.Context, taskID string, instructionID int64) error {
	query := `UPDATE device_song_downloads SET instruction_id = $1, updated_at = NOW() WHERE task_id = $2`
	_, err := r.db.ExecContext(ctx, query, instructionID, taskID)
	return err
}

// ListByUser 查询用户的下载记录列表
func (r *DeviceSongDownloadRepo) ListByUser(ctx context.Context, userID int64, page, pageSize int) ([]*model.DeviceSongDownload, int64, error) {
	countQuery := `SELECT COUNT(*) FROM device_song_downloads WHERE user_id = $1`
	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	
	offset := (page - 1) * pageSize
	query := `SELECT id, task_id, user_id, device_sn, content_id, song_name, download_url, 
		status, progress, local_path, file_size, error_msg, duration_ms, instruction_id, 
		created_at, updated_at, finished_at 
		FROM device_song_downloads WHERE user_id = $1 
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	
	rows, err := r.db.QueryContext(ctx, query, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	
	var records []*model.DeviceSongDownload
	for rows.Next() {
		var record model.DeviceSongDownload
		var instr sql.NullInt64
		scanErr := rows.Scan(
			&record.ID, &record.TaskID, &record.UserID, &record.DeviceSN, &record.ContentID,
			&record.SongName, &record.DownloadURL, &record.Status, &record.Progress,
			&record.LocalPath, &record.FileSize, &record.ErrorMsg, &record.DurationMs,
			&instr, &record.CreatedAt, &record.UpdatedAt, &record.FinishedAt,
		)
		if scanErr != nil {
			continue
		}
		if instr.Valid {
			v := instr.Int64
			record.InstructionID = &v
		}
		records = append(records, &record)
	}
	
	return records, total, nil
}
