package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go-admin/app/admin/device/models"

	"gorm.io/gorm"
)

// isReportTypeColumnMissing 判断是否为「缺少 report_type 列」（未执行 078 迁移时常出现）。
func isReportTypeColumnMissing(err error) bool {
	if err == nil {
		return false
	}
	raw := err.Error()
	sl := strings.ToLower(raw)
	if strings.Contains(sl, "42703") && strings.Contains(sl, "report_type") {
		return true
	}
	if strings.Contains(sl, "report_type") && strings.Contains(sl, "does not exist") {
		return true
	}
	if strings.Contains(raw, "report_type") && strings.Contains(raw, "不存在") {
		return true
	}
	return false
}

// DeviceStatusLogItem 管理端列表行
type DeviceStatusLogItem struct {
	Id                 int64     `json:"id"`
	DeviceId           int64     `json:"deviceId"`
	Sn                 string    `json:"sn"`
	BatteryLevel       int       `json:"batteryLevel"`
	StorageUsed        int64     `json:"storageUsed"`
	StorageTotal       int64     `json:"storageTotal"`
	SpeakerCount       int       `json:"speakerCount"`
	UwbX               *float64  `json:"uwbX"`
	UwbY               *float64  `json:"uwbY"`
	UwbZ               *float64  `json:"uwbZ"`
	AcousticCalibrated int16     `json:"acousticCalibrated"`
	AcousticOffset     *float64  `json:"acousticOffset"`
	ReportType         string    `json:"reportType"`
	ReportedAt         time.Time `json:"reportedAt"`
	CreatedAt          time.Time `json:"createdAt"`
}

// ListDeviceStatusLogs 按设备分页查询状态上报历史（created_at 降序）。
// 优先 deviceID（与详情页 device.id 一致，避免 SN/软删条件与 GetDeviceDetail 不一致）；否则按 SN 大小写不敏感匹配（与 GetDeviceDetail 相同，均不额外过滤 deleted_at，与 Table 查询行为一致）。
func (e *PlatformDeviceService) ListDeviceStatusLogs(sn string, deviceID int64, page, pageSize int, createdFrom, createdTo *time.Time) ([]DeviceStatusLogItem, int64, error) {
	if e.Orm == nil {
		return nil, 0, errors.New("orm nil")
	}
	var did int64
	if deviceID > 0 {
		var dev struct {
			ID int64 `gorm:"column:id"`
		}
		if err := e.Orm.Table("device").Select("id").Where("id = ?", deviceID).Take(&dev).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, 0, ErrPlatformDeviceNotFound
			}
			return nil, 0, err
		}
		did = dev.ID
	} else {
		sn = strings.TrimSpace(sn)
		if sn == "" {
			return nil, 0, ErrPlatformDeviceInvalid
		}
		snNorm := strings.ToUpper(sn)
		var dev struct {
			ID int64 `gorm:"column:id"`
		}
		if err := e.Orm.Table("device").
			Select("id").
			Where("UPPER(TRIM(sn)) = ?", snNorm).
			Take(&dev).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, 0, ErrPlatformDeviceNotFound
			}
			return nil, 0, err
		}
		did = dev.ID
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	offset := (page - 1) * pageSize

	q := e.Orm.Model(&models.DeviceStatusLog{}).Where("device_id = ?", did)
	if createdFrom != nil {
		q = q.Where("created_at >= ?", *createdFrom)
	}
	if createdTo != nil {
		q = q.Where("created_at <= ?", *createdTo)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []models.DeviceStatusLog
	listQ := e.Orm.Model(&models.DeviceStatusLog{}).Where("device_id = ?", did)
	if createdFrom != nil {
		listQ = listQ.Where("created_at >= ?", *createdFrom)
	}
	if createdTo != nil {
		listQ = listQ.Where("created_at <= ?", *createdTo)
	}
	if err := listQ.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		if !isReportTypeColumnMissing(err) {
			return nil, 0, err
		}
		listQ2 := e.Orm.Model(&models.DeviceStatusLog{}).Omit("ReportType").Where("device_id = ?", did)
		if createdFrom != nil {
			listQ2 = listQ2.Where("created_at >= ?", *createdFrom)
		}
		if createdTo != nil {
			listQ2 = listQ2.Where("created_at <= ?", *createdTo)
		}
		if err := listQ2.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
			return nil, 0, err
		}
	}

	out := make([]DeviceStatusLogItem, 0, len(rows))
	for i := range rows {
		r := rows[i]
		out = append(out, DeviceStatusLogItem{
			Id:                 r.Id,
			DeviceId:           r.DeviceId,
			Sn:                 r.Sn,
			BatteryLevel:       r.BatteryLevel,
			StorageUsed:        r.StorageUsed,
			StorageTotal:       r.StorageTotal,
			SpeakerCount:       r.SpeakerCount,
			UwbX:               r.UwbX,
			UwbY:               r.UwbY,
			UwbZ:               r.UwbZ,
			AcousticCalibrated: r.AcousticCalibrated,
			AcousticOffset:     r.AcousticOffset,
			ReportType:         r.ReportType,
			ReportedAt:         r.ReportedAt,
			CreatedAt:          r.CreatedAt,
		})
	}
	return out, total, nil
}

// ManualDeviceStatusReportIn 管理员手动录入一条状态（写入 device_status_logs，report_type=manual）
type ManualDeviceStatusReportIn struct {
	DeviceID int64
	Sn       string

	BatteryLevel       int
	StorageUsed        int64
	StorageTotal       int64
	SpeakerCount       int
	UwbX               *float64
	UwbY               *float64
	UwbZ               *float64
	AcousticCalibrated int16
	AcousticOffset     *float64
	ReportedAt         time.Time
}

// ManualInsertDeviceStatusReport 管理员录入状态上报（不经过设备 HTTP 鉴权，用于补录/演示）
func (e *PlatformDeviceService) ManualInsertDeviceStatusReport(in *ManualDeviceStatusReportIn) (*DeviceStatusLogItem, error) {
	if e.Orm == nil {
		return nil, errors.New("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}

	var dev struct {
		ID int64  `gorm:"column:id"`
		Sn string `gorm:"column:sn"`
	}
	if in.DeviceID > 0 {
		if err := e.Orm.Table("device").Select("id, sn").Where("id = ?", in.DeviceID).Take(&dev).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrPlatformDeviceNotFound
			}
			return nil, err
		}
	} else {
		sn := strings.TrimSpace(in.Sn)
		if sn == "" {
			return nil, ErrPlatformDeviceInvalid
		}
		snNorm := strings.ToUpper(sn)
		if err := e.Orm.Table("device").Select("id, sn").Where("UPPER(TRIM(sn)) = ?", snNorm).Take(&dev).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrPlatformDeviceNotFound
			}
			return nil, err
		}
	}

	if in.BatteryLevel < 0 || in.BatteryLevel > 100 {
		return nil, fmt.Errorf("battery_level 须在 0-100: %w", ErrPlatformDeviceInvalid)
	}
	if in.StorageUsed < 0 || in.StorageTotal < 0 {
		return nil, fmt.Errorf("storage 不能为负: %w", ErrPlatformDeviceInvalid)
	}
	if in.SpeakerCount < 0 {
		return nil, fmt.Errorf("speaker_count 不能为负: %w", ErrPlatformDeviceInvalid)
	}
	if in.AcousticCalibrated != 0 && in.AcousticCalibrated != 1 {
		return nil, fmt.Errorf("acoustic_calibrated 仅支持 0 或 1: %w", ErrPlatformDeviceInvalid)
	}

	snRow := strings.ToUpper(strings.TrimSpace(dev.Sn))
	if snRow == "" {
		return nil, fmt.Errorf("设备 SN 无效: %w", ErrPlatformDeviceInvalid)
	}

	now := time.Now()
	row := models.DeviceStatusLog{
		DeviceId:           dev.ID,
		Sn:                 snRow,
		BatteryLevel:       in.BatteryLevel,
		StorageUsed:        in.StorageUsed,
		StorageTotal:       in.StorageTotal,
		SpeakerCount:       in.SpeakerCount,
		UwbX:               in.UwbX,
		UwbY:               in.UwbY,
		UwbZ:               in.UwbZ,
		AcousticCalibrated: in.AcousticCalibrated,
		AcousticOffset:     in.AcousticOffset,
		ReportType:         "manual",
		ReportedAt:         in.ReportedAt,
		CreatedAt:          now,
	}

	if err := e.Orm.Create(&row).Error; err != nil {
		if !isReportTypeColumnMissing(err) {
			return nil, err
		}
		// 库仅 077、未跑 078 时无 report_type 列：省略该字段再插入
		if err2 := e.Orm.Omit("ReportType").Create(&row).Error; err2 != nil {
			return nil, err2
		}
		row.ReportType = "manual"
	}

	out := &DeviceStatusLogItem{
		Id:                 row.Id,
		DeviceId:           row.DeviceId,
		Sn:                 row.Sn,
		BatteryLevel:       row.BatteryLevel,
		StorageUsed:        row.StorageUsed,
		StorageTotal:       row.StorageTotal,
		SpeakerCount:       row.SpeakerCount,
		UwbX:               row.UwbX,
		UwbY:               row.UwbY,
		UwbZ:               row.UwbZ,
		AcousticCalibrated: row.AcousticCalibrated,
		AcousticOffset:     row.AcousticOffset,
		ReportType:         row.ReportType,
		ReportedAt:         row.ReportedAt,
		CreatedAt:          row.CreatedAt,
	}
	return out, nil
}
