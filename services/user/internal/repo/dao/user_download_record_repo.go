package dao

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type UserDownloadRecord struct {
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	ContentID      int64     `json:"content_id"`
	FileID         int64     `json:"file_id"`
	FileName       string    `json:"file_name"`
	FileSize       int64     `json:"file_size"`
	DownloadedSize int64     `json:"downloaded_size"`
	Status         string    `json:"status"`
	LocalPath      string    `json:"local_path"`
	IsDeleted      bool      `json:"is_deleted"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UserDownloadRecordRepo struct {
	db *sql.DB
}

func NewUserDownloadRecordRepo(db *sql.DB) *UserDownloadRecordRepo {
	return &UserDownloadRecordRepo{db: db}
}

type ListDownloadRecordsReq struct {
	UserID   int64
	Status   string
	Keyword  string
	Page     int
	PageSize int
}

type ListDownloadRecordsResp struct {
	List     []*UserDownloadRecord
	Total    int64
	Page     int
	PageSize int
}

func (r *UserDownloadRecordRepo) List(ctx context.Context, req *ListDownloadRecordsReq) (*ListDownloadRecordsResp, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	where := "WHERE user_id = $1 AND is_deleted = FALSE"
	args := []interface{}{req.UserID}
	argIdx := 2

	if req.Status != "" {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, req.Status)
		argIdx++
	}

	if req.Keyword != "" {
		where += fmt.Sprintf(" AND file_name LIKE $%d", argIdx)
		args = append(args, "%"+req.Keyword+"%")
		argIdx++
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM user_download_records %s", where)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("查询下载记录总数失败: %v", err)
	}

	offset := (req.Page - 1) * req.PageSize
	query := fmt.Sprintf(`
		SELECT id, user_id, content_id, file_id, file_name, file_size, downloaded_size, 
		       status, local_path, is_deleted, created_at, updated_at
		FROM user_download_records
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, req.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询下载记录列表失败: %v", err)
	}
	defer rows.Close()

	var list []*UserDownloadRecord
	for rows.Next() {
		var record UserDownloadRecord
		if err := rows.Scan(
			&record.ID, &record.UserID, &record.ContentID, &record.FileID,
			&record.FileName, &record.FileSize, &record.DownloadedSize,
			&record.Status, &record.LocalPath, &record.IsDeleted,
			&record.CreatedAt, &record.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描下载记录失败: %v", err)
		}
		list = append(list, &record)
	}

	return &ListDownloadRecordsResp{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (r *UserDownloadRecordRepo) Create(ctx context.Context, record *UserDownloadRecord) error {
	query := `
		INSERT INTO user_download_records (user_id, content_id, file_id, file_name, file_size, downloaded_size, status, local_path, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING id
	`
	return r.db.QueryRowContext(ctx, query,
		record.UserID, record.ContentID, record.FileID, record.FileName,
		record.FileSize, record.DownloadedSize, record.Status, record.LocalPath,
	).Scan(&record.ID)
}

func (r *UserDownloadRecordRepo) UpdateStatus(ctx context.Context, id int64, status string, downloadedSize int64) error {
	query := `
		UPDATE user_download_records
		SET status = $1, downloaded_size = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, status, downloadedSize, id)
	return err
}

func (r *UserDownloadRecordRepo) Delete(ctx context.Context, id int64, userID int64) error {
	query := `
		UPDATE user_download_records
		SET is_deleted = TRUE, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`
	_, err := r.db.ExecContext(ctx, query, id, userID)
	return err
}

func (r *UserDownloadRecordRepo) FindByContentID(ctx context.Context, userID int64, contentID int64) (*UserDownloadRecord, error) {
	query := `
		SELECT id, user_id, content_id, file_id, file_name, file_size, downloaded_size, 
		       status, local_path, is_deleted, created_at, updated_at
		FROM user_download_records
		WHERE user_id = $1 AND content_id = $2 AND is_deleted = FALSE
		ORDER BY created_at DESC
		LIMIT 1
	`
	var record UserDownloadRecord
	err := r.db.QueryRowContext(ctx, query, userID, contentID).Scan(
		&record.ID, &record.UserID, &record.ContentID, &record.FileID,
		&record.FileName, &record.FileSize, &record.DownloadedSize,
		&record.Status, &record.LocalPath, &record.IsDeleted,
		&record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询下载记录失败: %v", err)
	}
	return &record, nil
}

func (r *UserDownloadRecordRepo) FindByID(ctx context.Context, id int64, userID int64) (*UserDownloadRecord, error) {
	query := `
		SELECT id, user_id, content_id, file_id, file_name, file_size, downloaded_size, 
		       status, local_path, is_deleted, created_at, updated_at
		FROM user_download_records
		WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE
	`
	var record UserDownloadRecord
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&record.ID, &record.UserID, &record.ContentID, &record.FileID,
		&record.FileName, &record.FileSize, &record.DownloadedSize,
		&record.Status, &record.LocalPath, &record.IsDeleted,
		&record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询下载记录失败: %v", err)
	}
	return &record, nil
}

func (r *UserDownloadRecordRepo) UpdateDownloadedSize(ctx context.Context, id int64, userID int64, downloadedSize int64) error {
	query := `
		UPDATE user_download_records
		SET downloaded_size = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3
	`
	_, err := r.db.ExecContext(ctx, query, downloadedSize, id, userID)
	return err
}

func (r *UserDownloadRecordRepo) UpdateLocalPath(ctx context.Context, id int64, userID int64, localPath string) error {
	query := `
		UPDATE user_download_records
		SET local_path = $1, status = 'completed', updated_at = NOW()
		WHERE id = $2 AND user_id = $3
	`
	_, err := r.db.ExecContext(ctx, query, localPath, id, userID)
	return err
}

// BatchDeleteByFileIDs 批量软删除下载记录（按 file_id 列表）
// 功能：根据用户 ID 和文件 ID 列表批量软删除下载记录
// 参数：
//   - ctx: 请求上下文
//   - userID: 用户 ID，确保只能删除自己的记录
//   - fileIDs: 文件 ID 列表，要删除的记录对应的文件 ID
//
// 返回：
//   - int64: 实际删除的记录数量
//   - error: 操作失败时返回错误信息
//
// 注意：只软删除未被标记为已删除的记录（is_deleted = FALSE）
func (r *UserDownloadRecordRepo) BatchDeleteByFileIDs(ctx context.Context, userID int64, fileIDs []int64) (int64, error) {
	if len(fileIDs) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(fileIDs))
	args := make([]interface{}, 0, len(fileIDs)+1)
	args = append(args, userID)
	for i, id := range fileIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	query := fmt.Sprintf(`
		UPDATE user_download_records
		SET is_deleted = TRUE, updated_at = NOW()
		WHERE user_id = $1 AND file_id IN (%s) AND is_deleted = FALSE
	`, strings.Join(placeholders, ","))
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("批量删除下载记录失败: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取批量删除影响行数失败: %v", err)
	}
	return rowsAffected, nil
}

// CleanExpiredDownloadRecords 根据会员等级清理过期下载记录（软删除）
// 功能：按不同会员等级的保留期限，批量软删除超期的下载记录
// 参数：
//   - ctx: 请求上下文
//   - freeRetentionDays: 免费版保留天数（默认 7 天）
//   - standardRetentionDays: 标准版保留天数（默认 365 天）
//   - batchSize: 每批次清理的最大记录数（避免一次性删除过多）
//
// 返回：
//   - int64: 实际清理的记录总数
//   - error: 操作失败时返回错误信息
//
// 清理规则：
//   - 免费版（ordinary）：超过 freeRetentionDays 天的记录标记为已删除
//   - 标准版（vip/year_vip）：超过 standardRetentionDays 天的记录标记为已删除
//   - 专业版（svip）：永久保留，不删除
func (r *UserDownloadRecordRepo) CleanExpiredDownloadRecords(ctx context.Context, freeRetentionDays int, standardRetentionDays int, batchSize int) (int64, error) {
	if batchSize <= 0 {
		batchSize = 1000
	}

	now := time.Now()
	freeExpireTime := now.AddDate(0, 0, -freeRetentionDays)
	standardExpireTime := now.AddDate(0, 0, -standardRetentionDays)

	var totalCleaned int64

	// 清理免费版过期记录（ordinary 等级）
	freeQuery := `
		UPDATE user_download_records
		SET is_deleted = TRUE, updated_at = NOW()
		WHERE id IN (
			SELECT id FROM user_download_records
			WHERE is_deleted = FALSE
			AND created_at < $1
			AND user_id IN (
				SELECT user_id FROM user_member
				WHERE (level_code = 'ordinary' OR level_code IS NULL OR level_code = '')
				AND (status = 1 OR status IS NULL)
			)
			ORDER BY created_at ASC
			LIMIT $2
		)
	`

	for {
		result, err := r.db.ExecContext(ctx, freeQuery, freeExpireTime, batchSize)
		if err != nil {
			return totalCleaned, fmt.Errorf("清理免费版过期下载记录失败: %v", err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return totalCleaned, fmt.Errorf("获取清理影响行数失败: %v", err)
		}
		if rowsAffected == 0 {
			break
		}
		totalCleaned += rowsAffected
		if rowsAffected < int64(batchSize) {
			break
		}
	}

	// 清理标准版过期记录（vip/year_vip 等级）
	standardQuery := `
		UPDATE user_download_records
		SET is_deleted = TRUE, updated_at = NOW()
		WHERE id IN (
			SELECT id FROM user_download_records
			WHERE is_deleted = FALSE
			AND created_at < $1
			AND user_id IN (
				SELECT user_id FROM user_member
				WHERE level_code IN ('vip', 'year_vip')
				AND (status = 1 OR status IS NULL)
				AND (is_permanent = 0 OR is_permanent IS NULL)
			)
			ORDER BY created_at ASC
			LIMIT $2
		)
	`

	for {
		result, err := r.db.ExecContext(ctx, standardQuery, standardExpireTime, batchSize)
		if err != nil {
			return totalCleaned, fmt.Errorf("清理标准版过期下载记录失败: %v", err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return totalCleaned, fmt.Errorf("获取清理影响行数失败: %v", err)
		}
		if rowsAffected == 0 {
			break
		}
		totalCleaned += rowsAffected
		if rowsAffected < int64(batchSize) {
			break
		}
	}

	return totalCleaned, nil
}
