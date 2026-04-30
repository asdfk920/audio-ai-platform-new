package service

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OTA 升级任务状态：1 等待 2 升级中 3 成功 4 失败
const (
	otaTaskStatusWaiting = 1
	otaTaskStatusRunning = 2
)

var (
	ErrFirmwareMustDisable        = errors.New("请先禁用该固件后再删除")
	ErrFirmwareMustUnpublish      = errors.New("请先撤销发布后再删除该固件")
	ErrFirmwareTaskInProgress     = errors.New("该固件关联进行中的升级任务，请先取消")
	ErrFirmwareDeviceCache        = errors.New("有设备已下载该固件，请等待缓存过期或使用强制删除")
	ErrFirmwareDeleteConfirm      = errors.New("请确认删除操作（confirm 须为 true）")
	ErrFirmwareDeleteNoTarget     = errors.New("请提供 firmware_id 或 product_key 与 version")
	ErrFirmwareDeleteStorage      = errors.New("备份固件文件失败，请重试")
	ErrFirmwareDeleteNoPermission = errors.New("您没有权限删除该固件")
)

// FirmwareDeleteIn 固件删除入参
type FirmwareDeleteIn struct {
	FirmwareID int64
	ProductKey string
	Version    string
	Confirm    bool
	Force      bool // 忽略「已下载」提示，仍执行删除
	Reason     string
	Operator   string
}

// FirmwareBackupInfo 备份信息
type FirmwareBackupInfo struct {
	BackupPath string `json:"backup_path"`
	ExpiresAt  string `json:"expires_at"`
}

// FirmwareDeleteTaskInfo 关联任务（进行中时返回）
type FirmwareDeleteTaskInfo struct {
	PendingOrRunningCount int64   `json:"pending_or_running_count"`
	TaskIDs               []int64 `json:"task_ids,omitempty"`
}

// FirmwareDeleteOut 删除响应
type FirmwareDeleteOut struct {
	Success    bool                    `json:"success"`
	Message    string                  `json:"message"`
	BackupInfo *FirmwareBackupInfo     `json:"backup_info,omitempty"`
	TaskInfo   *FirmwareDeleteTaskInfo `json:"task_info,omitempty"`
}

func (e *PlatformDeviceService) loadActiveFirmwareByIDOrVersion(id int64, productKey, version string) (*firmwareRow, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	q := e.Orm.Table("ota_firmware AS f").Where("f.deleted_at IS NULL")
	if id > 0 {
		q = q.Where("f.id = ?", id)
	} else if pk := strings.TrimSpace(productKey); pk != "" && strings.TrimSpace(version) != "" {
		q = q.Where("f.product_key = ? AND f.version = ?", pk, strings.TrimSpace(version))
	} else {
		return nil, ErrFirmwareDeleteNoTarget
	}
	var r firmwareRow
	err := q.Select(`f.id, f.product_key, f.version, f.version_code, f.file_url, f.file_size, f.md5, f.upgrade_type, f.publish_status,
		f.release_note, f.min_sys_version, f.device_models, f.download_count, f.fw_status, f.creator, f.created_at, f.updated_at`).
		Take(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFirmwareNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func countBlockingUpgradeTasks(tx *gorm.DB, firmwareID int64) (int64, []int64, error) {
	var ids []int64
	err := tx.Table("ota_upgrade_task").
		Where("firmware_id = ? AND status IN (?, ?)", firmwareID, otaTaskStatusWaiting, otaTaskStatusRunning).
		Order("id ASC").
		Limit(100).
		Pluck("id", &ids).Error
	if err != nil {
		return 0, nil, err
	}
	return int64(len(ids)), ids, nil
}

// backupFirmwareFile 将本地静态文件移至备份目录，保留约 30 天由运维或定时任务清理
func backupFirmwareFile(firmwareID int64, fileURL string) (relBackup string, expiresAt time.Time, err error) {
	expiresAt = time.Now().Add(30 * 24 * time.Hour)
	u := strings.TrimSpace(fileURL)
	if u == "" {
		return "", expiresAt, nil
	}
	low := strings.ToLower(u)
	if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") {
		return "", expiresAt, nil
	}
	rel := strings.TrimPrefix(u, "/")
	rel = filepath.ToSlash(rel)
	if rel == "" {
		return "", expiresAt, nil
	}
	src := filepath.FromSlash(rel)
	if _, st := os.Stat(src); os.IsNotExist(st) {
		return "", expiresAt, nil
	}
	dir := filepath.Join("static", "uploadfile", "firmware_backup")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", time.Time{}, fmt.Errorf("%w: %v", ErrFirmwareDeleteStorage, err)
	}
	ext := filepath.Ext(src)
	if ext == "" {
		ext = ".bin"
	}
	dstName := fmt.Sprintf("%d_%s%s", firmwareID, uuid.New().String(), ext)
	dst := filepath.Join(dir, dstName)
	if err := moveFileOrCopy(src, dst); err != nil {
		return "", time.Time{}, fmt.Errorf("%w: %v", ErrFirmwareDeleteStorage, err)
	}
	return filepath.ToSlash(filepath.Join("static", "uploadfile", "firmware_backup", dstName)), expiresAt, nil
}

func moveFileOrCopy(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		_ = os.Remove(dst)
		return err
	}
	_ = in.Close()
	_ = out.Close()
	return os.Remove(src)
}

// DeleteFirmware 软删除固件：校验 → 备份文件 → 标记删除
func (e *PlatformDeviceService) DeleteFirmware(in *FirmwareDeleteIn) (*FirmwareDeleteOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	if !in.Confirm {
		return nil, ErrFirmwareDeleteConfirm
	}

	row, err := e.loadActiveFirmwareByIDOrVersion(in.FirmwareID, in.ProductKey, in.Version)
	if err != nil {
		return nil, err
	}

	if row.FwStatus != 2 {
		return nil, ErrFirmwareMustDisable
	}
	if row.PublishStatus == 2 {
		return nil, ErrFirmwareMustUnpublish
	}

	if row.DownloadCount > 0 && !in.Force {
		return nil, ErrFirmwareDeviceCache
	}

	var blockCount int64
	var blockIDs []int64
	err = e.Orm.Transaction(func(tx *gorm.DB) error {
		c, ids, err := countBlockingUpgradeTasks(tx, row.ID)
		if err != nil {
			return err
		}
		blockCount, blockIDs = c, ids
		return nil
	})
	if err != nil {
		return nil, err
	}
	if blockCount > 0 {
		return &FirmwareDeleteOut{
			Success: false,
			Message: ErrFirmwareTaskInProgress.Error(),
			TaskInfo: &FirmwareDeleteTaskInfo{
				PendingOrRunningCount: blockCount,
				TaskIDs:               blockIDs,
			},
		}, nil
	}

	backupRel, expAt, berr := backupFirmwareFile(row.ID, row.FileURL)
	if berr != nil {
		return nil, berr
	}

	origURL := strings.TrimSpace(row.FileURL)
	restoreSrc := backupRel
	restoreDst := localPathFromFileURL(origURL)
	needRestore := restoreSrc != "" && restoreDst != ""
	if needRestore {
		defer func() {
			if needRestore {
				_ = moveFileOrCopy(filepath.FromSlash(restoreSrc), restoreDst)
			}
		}()
	}

	reason := strings.TrimSpace(in.Reason)
	by := strings.TrimSpace(in.Operator)
	if by == "" {
		by = "admin"
	}

	var lateBlockCount int64
	var lateBlockIDs []int64
	err = e.Orm.Transaction(func(tx *gorm.DB) error {
		c, ids, err := countBlockingUpgradeTasks(tx, row.ID)
		if err != nil {
			return err
		}
		lateBlockCount, lateBlockIDs = c, ids
		if c > 0 {
			return ErrFirmwareTaskInProgress
		}

		var rel interface{}
		if reason != "" {
			rel = reason
		}

		res := tx.Exec(`
UPDATE ota_firmware SET
  deleted_at = CURRENT_TIMESTAMP,
  deleted_by = ?,
  delete_reason = ?,
  backup_path = ?,
  backup_expires_at = ?,
  file_url = '',
  updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND deleted_at IS NULL`,
			by,
			rel,
			backupRel,
			expAt,
			row.ID,
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrFirmwareNotFound
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrFirmwareTaskInProgress) {
			return &FirmwareDeleteOut{
				Success: false,
				Message: ErrFirmwareTaskInProgress.Error(),
				TaskInfo: &FirmwareDeleteTaskInfo{
					PendingOrRunningCount: lateBlockCount,
					TaskIDs:               lateBlockIDs,
				},
			}, nil
		}
		return nil, err
	}

	needRestore = false

	bi := &FirmwareBackupInfo{
		BackupPath: backupRel,
		ExpiresAt:  expAt.Format(time.RFC3339),
	}

	return &FirmwareDeleteOut{
		Success:    true,
		Message:    "固件已软删除，文件已移至备份目录（约 30 天后可清理）",
		BackupInfo: bi,
	}, nil
}

func localPathFromFileURL(fileURL string) string {
	u := strings.TrimSpace(fileURL)
	if u == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(u), "http://") || strings.HasPrefix(strings.ToLower(u), "https://") {
		return ""
	}
	rel := strings.TrimPrefix(u, "/")
	if rel == "" {
		return ""
	}
	return filepath.FromSlash(rel)
}
