package repo

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/jacklau/audio-ai-platform/services/content/internal/repo/dao"
)

// DeviceSongDownloadRepo 设备歌曲下载记录仓储
type DeviceSongDownloadRepo struct {
	db *gorm.DB
}

func NewDeviceSongDownloadRepo(db *gorm.DB) *DeviceSongDownloadRepo {
	return &DeviceSongDownloadRepo{db: db}
}

// Create 创建下载记录
func (r *DeviceSongDownloadRepo) Create(ctx context.Context, record *dao.DeviceSongDownload) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// FindByTaskID 根据任务ID查询
func (r *DeviceSongDownloadRepo) FindByTaskID(ctx context.Context, taskID string) (*dao.DeviceSongDownload, error) {
	var record dao.DeviceSongDownload
	err := r.db.WithContext(ctx).Where("task_id = ?", taskID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateStatus 更新状态和进度
func (r *DeviceSongDownloadRepo) UpdateStatus(ctx context.Context, taskID string, status string, progress float64) error {
	updates := map[string]interface{}{
		"status":     status,
		"progress":   progress,
		"updated_at": time.Now(),
	}
	return r.db.WithContext(ctx).Model(&dao.DeviceSongDownload{}).Where("task_id = ?", taskID).Updates(updates).Error
}

// UpdateResult 更新下载结果（成功或失败）
func (r *DeviceSongDownloadRepo) UpdateResult(ctx context.Context, taskID string, status, errorMsg, localPath string, fileSize int64, durationMs int64) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":      status,
		"progress":    100,
		"error_msg":   errorMsg,
		"local_path":  localPath,
		"file_size":   fileSize,
		"duration_ms": durationMs,
		"updated_at":  now,
	}

	if status == "success" || status == "failed" {
		updates["finished_at"] = now
	}

	return r.db.WithContext(ctx).Model(&dao.DeviceSongDownload{}).Where("task_id = ?", taskID).Updates(updates).Error
}

// UpdateInstructionID 更新指令ID
func (r *DeviceSongDownloadRepo) UpdateInstructionID(ctx context.Context, taskID string, instructionID int64) error {
	return r.db.WithContext(ctx).Model(&dao.DeviceSongDownload{}).
		Where("task_id = ?", taskID).
		Update("instruction_id", instructionID).Error
}

// ListByUser 查询用户的下载记录列表
func (r *DeviceSongDownloadRepo) ListByUser(ctx context.Context, userID int64, page, pageSize int) ([]*dao.DeviceSongDownload, int64, error) {
	var total int64
	var records []*dao.DeviceSongDownload

	query := r.db.WithContext(ctx).Model(&dao.DeviceSongDownload{}).Where("user_id = ?", userID)

	countErr := query.Count(&total).Error
	if countErr != nil {
		return nil, 0, countErr
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&records).Error
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// DownloadListItem 带完整信息的设备下载列表项（device_song_downloads + content，可选聚合统计 cds）
type DownloadListItem struct {
	ID             int64  `json:"download_id"`                                            // 下载记录ID（主键）
	ContentID      int64  `json:"content_id"`                                             // 内容ID（歌曲ID）
	TotalDownloads int64  `json:"total_downloads"`                                        // 总下载次数（来自 content_download_stats，可空）
	TodayDownloads int64  `json:"today_downloads"`                                        // 今日下载次数
	WeekDownloads  int64  `json:"week_downloads"`                                         // 本周下载次数
	LastStatus     string `json:"status"`                                                 // 任务状态
	LastErrorMsg   string `json:"error_msg,omitempty"`                                    // 最后错误信息
	LastLocalPath  string `json:"local_path,omitempty"`                                   // 设备本地路径
	LastFileSize   int64  `json:"file_size"`                                              // 文件大小（字节）
	LastDurationMs int64  `json:"duration_ms"`                                            // 下载耗时（毫秒）
	LastDownloadAt string `json:"download_time,omitempty" gorm:"column:last_download_at"` // 最近时间

	SongName    string `json:"song_name"`           // 歌曲名称
	Artist      string `json:"artist"`              // 艺术家名称
	CoverURL    string `json:"cover_url,omitempty"` // 封面图地址
	DurationSec int    `json:"duration" gorm:"column:duration_sec"`
	SortTs      string `json:"-" gorm:"column:sort_ts"`
}

// ListByUserWithFilter 合并 device_song_downloads（设备下发）与 user_downloads（应用下载历史），同一用户分页去重排序。
func (r *DeviceSongDownloadRepo) ListByUserWithFilter(
	ctx context.Context,
	userID int64,
	page, pageSize int,
	songName, artist, deviceSN, status string,
) ([]*DownloadListItem, int64, error) {
	var total int64
	var records []*DownloadListItem

	mergeCTE := `
WITH unioned AS (
  SELECT
    dsd.id AS id,
    dsd.content_id AS content_id,
    COALESCE(cds.total_downloads, 0) AS total_downloads,
    COALESCE(cds.today_downloads, 0) AS today_downloads,
    COALESCE(cds.week_downloads, 0) AS week_downloads,
    COALESCE(dsd.status, '') AS last_status,
    COALESCE(dsd.error_msg, '') AS last_error_msg,
    COALESCE(dsd.local_path, '') AS last_local_path,
    COALESCE(dsd.file_size, 0) AS last_file_size,
    COALESCE(dsd.duration_ms, 0) AS last_duration_ms,
    to_char(COALESCE(dsd.finished_at, dsd.updated_at, dsd.created_at), 'YYYY-MM-DD HH24:MI:SS') AS last_download_at,
    COALESCE(NULLIF(TRIM(dsd.song_name),''), c.title, '') AS song_name,
    COALESCE(c.artist, '') AS artist,
    COALESCE(c.cover_url, '') AS cover_url,
    COALESCE(c.duration_sec, 0) AS duration_sec,
    COALESCE(TRIM(dsd.device_sn), '') AS device_hint,
    COALESCE(EXTRACT(epoch FROM COALESCE(dsd.finished_at, dsd.updated_at, dsd.created_at))::bigint, 0) AS sort_epoch
  FROM device_song_downloads dsd
  LEFT JOIN content c ON dsd.content_id = c.id
  LEFT JOIN content_download_stats cds ON cds.content_id = dsd.content_id
  WHERE dsd.user_id = ?

  UNION ALL

  SELECT
    ud.id AS id,
    ud.content_id AS content_id,
    COALESCE(cds2.total_downloads, 0) AS total_downloads,
    COALESCE(cds2.today_downloads, 0) AS today_downloads,
    COALESCE(cds2.week_downloads, 0) AS week_downloads,
    (
      CASE COALESCE(ud.status, -1)
        WHEN 0 THEN 'pending'
        WHEN 1 THEN 'downloading'
        WHEN 2 THEN 'downloading'
        WHEN 3 THEN 'success'
        ELSE 'unknown'
      END
    ) AS last_status,
    '' AS last_error_msg,
    COALESCE(ud.file_url, '') AS last_local_path,
    COALESCE(ud.file_size, 0) AS last_file_size,
    0 AS last_duration_ms,
    to_char(ud.download_time, 'YYYY-MM-DD HH24:MI:SS') AS last_download_at,
    COALESCE(NULLIF(TRIM(ud.content_title),''), c2.title, '') AS song_name,
    COALESCE(c2.artist, '') AS artist,
    COALESCE(c2.cover_url, '') AS cover_url,
    COALESCE(c2.duration_sec, 0) AS duration_sec,
    CAST('' AS VARCHAR(512)) AS device_hint,
    COALESCE(EXTRACT(epoch FROM ud.download_time)::bigint, 0) AS sort_epoch
  FROM user_downloads ud
  LEFT JOIN content c2 ON ud.content_id = c2.id
  LEFT JOIN content_download_stats cds2 ON cds2.content_id = ud.content_id
  WHERE ud.user_id = ?
)
`
	argsHead := []interface{}{userID, userID}

	filterBody := ""
	var filterArgs []interface{}

	if songName != "" {
		pat := "%" + songName + "%"
		filterBody += " AND COALESCE(m.song_name,'') ILIKE ?"
		filterArgs = append(filterArgs, pat)
	}
	if artist != "" {
		filterBody += " AND COALESCE(m.artist,'') ILIKE ?"
		filterArgs = append(filterArgs, "%"+artist+"%")
	}
	if snTrim := strings.TrimSpace(deviceSN); snTrim != "" {
		filterBody += " AND COALESCE(m.device_hint,'') ILIKE ?"
		filterArgs = append(filterArgs, "%"+snTrim+"%")
	}
	if st := normalizeMergedDownloadStatusFilter(status); st.clause != "" {
		filterBody += st.clause
		filterArgs = append(filterArgs, st.args...)
	}

	countSQL := mergeCTE + "SELECT COUNT(*) FROM unioned m WHERE 1=1" + filterBody
	countArgs := append(append([]interface{}{}, argsHead...), filterArgs...)
	err := r.db.WithContext(ctx).Raw(countSQL, countArgs...).Scan(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	dataSQL := mergeCTE + `
SELECT
  m.id,
  m.content_id,
  m.total_downloads,
  m.today_downloads,
  m.week_downloads,
  m.last_status,
  m.last_error_msg,
  m.last_local_path,
  m.last_file_size,
  m.last_duration_ms,
  m.last_download_at,
  m.song_name,
  m.artist,
  m.cover_url,
  m.duration_sec,
  CAST(m.sort_epoch AS TEXT) AS sort_ts
FROM unioned m
WHERE 1=1 ` + filterBody + `
ORDER BY m.sort_epoch DESC NULLS LAST, m.id DESC
LIMIT ? OFFSET ?`
	dataArgs := append(append([]interface{}{}, argsHead...), filterArgs...)
	dataArgs = append(dataArgs, pageSize, offset)

	err = r.db.WithContext(ctx).Raw(dataSQL, dataArgs...).Scan(&records).Error
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

type statusFilterSQL struct {
	clause string
	args   []interface{}
}

func normalizeMergedDownloadStatusFilter(status string) statusFilterSQL {
	s := strings.ToLower(strings.TrimSpace(status))
	if s == "" {
		return statusFilterSQL{}
	}
	switch s {
	case "pending":
		return statusFilterSQL{clause: ` AND LOWER(TRIM(m.last_status)) IN ('pending','cached')`, args: nil}
	case "downloading":
		return statusFilterSQL{clause: ` AND LOWER(TRIM(m.last_status)) IN ('downloading','sent','delivered','unknown')`, args: nil}
	case "success":
		return statusFilterSQL{clause: ` AND LOWER(TRIM(m.last_status)) IN ('success','completed')`, args: nil}
	case "failed":
		return statusFilterSQL{clause: ` AND LOWER(TRIM(m.last_status)) IN ('failed')`, args: nil}
	default:
		return statusFilterSQL{clause: ` AND LOWER(TRIM(m.last_status)) = ?`, args: []interface{}{s}}
	}
}

// GetActiveDownloadsByDevice 获取设备正在进行的下载任务
func (r *DeviceSongDownloadRepo) GetActiveDownloadsByDevice(ctx context.Context, deviceSN string) ([]*dao.DeviceSongDownload, error) {
	var records []*dao.DeviceSongDownload
	err := r.db.WithContext(ctx).
		Where("device_sn = ? AND status IN (?)", deviceSN, []string{"pending", "downloading"}).
		Order("created_at ASC").
		Find(&records).Error
	return records, err
}
