package service

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

var ErrFirmwareNotFound = errors.New("固件不存在")

// FirmwareListFilter 固件列表筛选
type FirmwareListFilter struct {
	ProductKey     string
	Version        string // 模糊或精确由 VersionExact 决定
	VersionExact   bool
	VersionCodeMin *int
	VersionCodeMax *int
	DeviceModel    string // 模糊匹配 device_models 列
	Status         *int16 // 1=启用 2=禁用；nil=不限
	Keyword        string // 版本号或说明
	CreatedFrom    *time.Time
	CreatedTo      *time.Time
	SortBy         string // created_at, version_code, download_count, version, device_models
	SortOrder      string // asc, desc
	IncludeDeleted bool   // true 时包含已软删记录（审计）
}

// FirmwareListItem 固件列表行
type FirmwareListItem struct {
	FirmwareID       int64   `json:"firmware_id"`
	ProductKey       string  `json:"product_key"`
	ProductName      string  `json:"product_name"`
	Version          string  `json:"version"`
	VersionCode      int     `json:"version_code"`
	FileSize         int64   `json:"file_size"`
	FileSizeHuman    string  `json:"file_size_human"`
	FileSizeMB       float64 `json:"file_size_mb"`
	FileMd5          string  `json:"file_md5"`
	Description      string  `json:"description"`
	ForceUpdate      bool    `json:"force_update"`
	MinSysVersion    string  `json:"min_sys_version"`
	DeviceModels     string  `json:"device_models"`
	Status           int16   `json:"status"` // fw_status 1/2
	StatusText       string  `json:"status_text"`
	PublishStatus    int16   `json:"publish_status"`
	DownloadCount    int64   `json:"download_count"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at,omitempty"`
	Creator          string  `json:"creator"`
	IsLatest         bool    `json:"is_latest"`
	ProductDeviceCnt int64   `json:"product_device_count"` // 该产品线下设备总数（参考）
	FileURL          string  `json:"file_url,omitempty"`
}

func firmwareStatusText(st int16) string {
	switch st {
	case 1:
		return "启用"
	case 2:
		return "禁用"
	default:
		return "未知"
	}
}

func formatFileHuman(n int64) string {
	if n < 0 {
		n = 0
	}
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	x := float64(n)
	if x < 1024*1024 {
		return fmt.Sprintf("%.2f KB", x/1024)
	}
	if x < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", x/(1024*1024))
	}
	return fmt.Sprintf("%.2f GB", x/(1024*1024*1024))
}

func fileSizeMB(n int64) float64 {
	return math.Round(float64(n)*100/(1024*1024)) / 100
}

func (e *PlatformDeviceService) firmwareListQuery(f FirmwareListFilter) *gorm.DB {
	q := e.Orm.Table("ota_firmware AS f")
	if !f.IncludeDeleted {
		q = q.Where("f.deleted_at IS NULL")
	}
	if pk := strings.TrimSpace(f.ProductKey); pk != "" {
		q = q.Where("f.product_key = ?", pk)
	}
	if v := strings.TrimSpace(f.Version); v != "" {
		if f.VersionExact {
			q = q.Where("f.version = ?", v)
		} else {
			q = q.Where("f.version ILIKE ?", "%"+v+"%")
		}
	}
	if f.VersionCodeMin != nil {
		q = q.Where("f.version_code >= ?", *f.VersionCodeMin)
	}
	if f.VersionCodeMax != nil {
		q = q.Where("f.version_code <= ?", *f.VersionCodeMax)
	}
	if m := strings.TrimSpace(f.DeviceModel); m != "" {
		q = q.Where("f.device_models ILIKE ?", "%"+m+"%")
	}
	if f.Status != nil {
		q = q.Where("f.fw_status = ?", *f.Status)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(f.version ILIKE ? OR COALESCE(f.release_note,'') ILIKE ?)", like, like)
	}
	if f.CreatedFrom != nil {
		q = q.Where("f.created_at >= ?", *f.CreatedFrom)
	}
	if f.CreatedTo != nil {
		t := *f.CreatedTo
		if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
			t = t.Add(24*time.Hour - time.Nanosecond)
		}
		q = q.Where("f.created_at <= ?", t)
	}
	return q
}

func firmwareOrderSQL(f FirmwareListFilter) string {
	col := "f.created_at"
	switch strings.ToLower(strings.TrimSpace(f.SortBy)) {
	case "version_code":
		col = "f.version_code"
	case "download_count":
		col = "f.download_count"
	case "version":
		col = "f.version"
	case "device_models":
		col = "f.device_models"
	case "created_at", "":
		col = "f.created_at"
	}
	ord := "DESC"
	if strings.EqualFold(strings.TrimSpace(f.SortOrder), "asc") {
		ord = "ASC"
	}
	return col + " " + ord
}

type firmwareRow struct {
	ID             int64     `gorm:"column:id"`
	ProductKey     string    `gorm:"column:product_key"`
	Version        string    `gorm:"column:version"`
	VersionCode    int       `gorm:"column:version_code"`
	FileURL        string    `gorm:"column:file_url"`
	FileSize       int64     `gorm:"column:file_size"`
	Md5            string    `gorm:"column:md5"`
	UpgradeType    int16     `gorm:"column:upgrade_type"`
	PublishStatus  int16     `gorm:"column:publish_status"`
	ReleaseNote    *string   `gorm:"column:release_note"`
	MinSysVersion  string    `gorm:"column:min_sys_version"`
	DeviceModels   string    `gorm:"column:device_models"`
	DownloadCount  int64     `gorm:"column:download_count"`
	FwStatus       int16     `gorm:"column:fw_status"`
	Creator        string    `gorm:"column:creator"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

// ListFirmware 固件分页列表
func (e *PlatformDeviceService) ListFirmware(page, pageSize int, f FirmwareListFilter) ([]FirmwareListItem, int64, error) {
	if e.Orm == nil {
		return nil, 0, fmt.Errorf("orm nil")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	base := e.firmwareListQuery(f)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	order := firmwareOrderSQL(f)

	var rows []firmwareRow
	if err := e.firmwareListQuery(f).
		Select(`f.id, f.product_key, f.version, f.version_code, f.file_url, f.file_size, f.md5, f.upgrade_type, f.publish_status,
			f.release_note, f.min_sys_version, f.device_models, f.download_count, f.fw_status, f.creator, f.created_at, f.updated_at`).
		Order(order).
		Offset(offset).Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	// 各 product_key 下最大 version_code（用于 is_latest）
	maxVC := map[string]int{}
	if len(rows) > 0 {
		var pks []string
		seen := map[string]bool{}
		for _, r := range rows {
			pk := strings.TrimSpace(r.ProductKey)
			if pk != "" && !seen[pk] {
				seen[pk] = true
				pks = append(pks, pk)
			}
		}
		if len(pks) > 0 {
			type agg struct {
				PK string `gorm:"column:product_key"`
				M  int    `gorm:"column:m"`
			}
			var aggs []agg
			_ = e.Orm.Table("ota_firmware").Select("product_key, MAX(version_code) AS m").Where("product_key IN ? AND deleted_at IS NULL", pks).Group("product_key").Scan(&aggs).Error
			for _, a := range aggs {
				maxVC[a.PK] = a.M
			}
		}
	}
	maxIDByPK := map[string]int64{}
	if len(rows) > 0 {
		var pks2 []string
		seen := map[string]bool{}
		for _, r := range rows {
			pk := strings.TrimSpace(r.ProductKey)
			if pk != "" && !seen[pk] {
				seen[pk] = true
				pks2 = append(pks2, pk)
			}
		}
		if len(pks2) > 0 {
			type idagg struct {
				PK string `gorm:"column:product_key"`
				Mx int64  `gorm:"column:mx"`
			}
			var idaggs []idagg
			_ = e.Orm.Table("ota_firmware").Select("product_key, MAX(id) AS mx").Where("product_key IN ? AND deleted_at IS NULL", pks2).Group("product_key").Scan(&idaggs).Error
			for _, a := range idaggs {
				maxIDByPK[a.PK] = a.Mx
			}
		}
	}

	// 产品线设备数（参考）
	devCnt := map[string]int64{}
	for _, r := range rows {
		pk := strings.TrimSpace(r.ProductKey)
		if pk == "" {
			continue
		}
		if _, ok := devCnt[pk]; ok {
			continue
		}
		var c int64
		_ = e.Orm.Table("device").Where("product_key = ?", pk).Count(&c).Error
		devCnt[pk] = c
	}

	out := make([]FirmwareListItem, 0, len(rows))
	for _, r := range rows {
		desc := ""
		if r.ReleaseNote != nil {
			desc = *r.ReleaseNote
		}
		displayVC := r.VersionCode
		if displayVC == 0 {
			displayVC = guessVersionCode(r.Version)
		}
		pk := strings.TrimSpace(r.ProductKey)
		isLatest := false
		gmax, hasMax := maxVC[pk]
		if hasMax && gmax > 0 && r.VersionCode > 0 {
			isLatest = r.VersionCode == gmax
		} else if r.VersionCode == 0 && hasMax && gmax == 0 {
			if mx, ok := maxIDByPK[pk]; ok && mx > 0 {
				isLatest = r.ID == mx
			}
		}

		st := r.FwStatus
		if st == 0 {
			st = 1
		}

		out = append(out, FirmwareListItem{
			FirmwareID:       r.ID,
			ProductKey:       r.ProductKey,
			ProductName:      productDisplayName(pk),
			Version:          r.Version,
			VersionCode:      displayVC,
			FileSize:         r.FileSize,
			FileSizeHuman:    formatFileHuman(r.FileSize),
			FileSizeMB:       fileSizeMB(r.FileSize),
			FileURL:          r.FileURL,
			FileMd5:          r.Md5,
			Description:      desc,
			ForceUpdate:      r.UpgradeType == 2,
			MinSysVersion:    strings.TrimSpace(r.MinSysVersion),
			DeviceModels:     strings.TrimSpace(r.DeviceModels),
			Status:           st,
			StatusText:       firmwareStatusText(st),
			PublishStatus:    r.PublishStatus,
			DownloadCount:    r.DownloadCount,
			CreatedAt:        r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        r.UpdatedAt.Format(time.RFC3339),
			Creator:          strings.TrimSpace(r.Creator),
			IsLatest:         isLatest,
			ProductDeviceCnt: devCnt[pk],
		})
	}

	return out, total, nil
}

func productDisplayName(productKey string) string {
	if strings.TrimSpace(productKey) == "" {
		return ""
	}
	// 无独立产品表时，名称默认与 product_key 一致，便于前端展示
	return productKey
}

// guessVersionCode 从 v1.2.3 粗略得到 1002003
func guessVersionCode(ver string) int {
	ver = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ver), "v"))
	ver = strings.TrimPrefix(ver, "V")
	parts := strings.Split(ver, ".")
	if len(parts) == 0 {
		return 0
	}
	multi := []int{1000000, 1000, 1}
	sum := 0
	for i := 0; i < len(parts) && i < 3; i++ {
		n, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil {
			return 0
		}
		if n > 999 {
			n = 999
		}
		sum += n * multi[i]
	}
	return sum
}

// GetFirmwareDetail 固件详情（单条）
func (e *PlatformDeviceService) GetFirmwareDetail(id int64) (*FirmwareListItem, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if id <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}
	var r firmwareRow
	err := e.Orm.Table("ota_firmware AS f").
		Select(`f.id, f.product_key, f.version, f.version_code, f.file_url, f.file_size, f.md5, f.upgrade_type, f.publish_status,
			f.release_note, f.min_sys_version, f.device_models, f.download_count, f.fw_status, f.creator, f.created_at, f.updated_at`).
		Where("f.id = ? AND f.deleted_at IS NULL", id).
		Take(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFirmwareNotFound
	}
	if err != nil {
		return nil, err
	}

	var mx struct {
		M int `gorm:"column:m"`
	}
	_ = e.Orm.Raw(`SELECT COALESCE(MAX(version_code),0) AS m FROM ota_firmware WHERE product_key = ? AND deleted_at IS NULL`, r.ProductKey).Scan(&mx).Error
	maxVC := mx.M
	var maxID int64
	_ = e.Orm.Raw(`SELECT COALESCE(MAX(id),0) FROM ota_firmware WHERE product_key = ? AND deleted_at IS NULL`, r.ProductKey).Scan(&maxID).Error
	isLatest := false
	if maxVC > 0 && r.VersionCode > 0 {
		isLatest = r.VersionCode == maxVC
	} else if r.VersionCode == 0 && maxVC == 0 && r.ID == maxID && maxID > 0 {
		isLatest = true
	}

	desc := ""
	if r.ReleaseNote != nil {
		desc = *r.ReleaseNote
	}
	vc := r.VersionCode
	if vc == 0 {
		vc = guessVersionCode(r.Version)
	}
	st := r.FwStatus
	if st == 0 {
		st = 1
	}
	pk := strings.TrimSpace(r.ProductKey)
	var pc int64
	_ = e.Orm.Table("device").Where("product_key = ?", pk).Count(&pc).Error

	item := &FirmwareListItem{
		FirmwareID:       r.ID,
		ProductKey:       r.ProductKey,
		ProductName:      productDisplayName(pk),
		Version:          r.Version,
		VersionCode:      vc,
		FileSize:         r.FileSize,
		FileSizeHuman:    formatFileHuman(r.FileSize),
		FileSizeMB:       fileSizeMB(r.FileSize),
		FileURL:          r.FileURL,
		FileMd5:          r.Md5,
		Description:      desc,
		ForceUpdate:      r.UpgradeType == 2,
		MinSysVersion:    strings.TrimSpace(r.MinSysVersion),
		DeviceModels:     strings.TrimSpace(r.DeviceModels),
		Status:           st,
		StatusText:       firmwareStatusText(st),
		PublishStatus:    r.PublishStatus,
		DownloadCount:    r.DownloadCount,
		CreatedAt:        r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        r.UpdatedAt.Format(time.RFC3339),
		Creator:          strings.TrimSpace(r.Creator),
		IsLatest:         isLatest,
		ProductDeviceCnt: pc,
	}
	return item, nil
}
