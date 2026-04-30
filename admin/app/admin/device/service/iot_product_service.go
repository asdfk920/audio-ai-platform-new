package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go-admin/app/admin/device/models"

	"gorm.io/gorm"
)

var (
	ErrIotProductNotFound    = errors.New("产品不存在")
	ErrIotProductDuplicate   = errors.New("产品标识已存在")
	ErrIotProductInvalidKey  = errors.New("产品标识格式无效：2～64 位，字母数字及 ._-")
	ErrIotProductInvalidStatus = errors.New("状态无效")
)

var productKeyPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]{1,62}$`)

// IotProductListIn 列表参数
type IotProductListIn struct {
	Page     int
	PageSize int
	Keyword  string
	Status   string // draft|published|disabled 或空
}

// IotProductListItem 列表行
type IotProductListItem struct {
	ID                 int64  `json:"id"`
	ProductKey         string `json:"productKey"`
	Name               string `json:"name"`
	Category           string `json:"category"`
	Status             string `json:"status"`
	FirmwareCount      int64  `json:"firmwareCount"`
	DeviceCount        int64  `json:"deviceCount"`
	RegistrationReady  bool   `json:"registrationReady"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

// IotProductDetailOut 详情
type IotProductDetailOut struct {
	IotProductListItem
	Description   string          `json:"description"`
	Communication json.RawMessage `json:"communication"`
	DeviceType    string          `json:"deviceType"`
}

// IotProductCreateIn 创建
type IotProductCreateIn struct {
	ProductKey    string
	Name          string
	Category      string
	Description   string
	Communication json.RawMessage
	DeviceType    string
	Status        string
	CreateBy      int64
}

// IotProductUpdateIn 更新
type IotProductUpdateIn struct {
	Name          string
	Category      string
	Description   string
	Communication json.RawMessage
	DeviceType    string
	UpdateBy      int64
}

func normalizeProductKey(pk string) string {
	return strings.TrimSpace(pk)
}

func validateProductKeyFormat(pk string) error {
	s := normalizeProductKey(pk)
	if len(s) < 2 || len(s) > 64 {
		return ErrIotProductInvalidKey
	}
	if !productKeyPattern.MatchString(s) {
		return ErrIotProductInvalidKey
	}
	return nil
}

func validIotProductStatus(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "draft", "published", "disabled":
		return true
	default:
		return false
	}
}

func (e *PlatformDeviceService) countFirmwareForProductKey(tx *gorm.DB, productKey string) (int64, error) {
	var n int64
	err := tx.Table("ota_firmware").
		Where("product_key = ? AND deleted_at IS NULL", productKey).
		Count(&n).Error
	return n, err
}

func (e *PlatformDeviceService) countDevicesForProductKey(tx *gorm.DB, productKey string) (int64, error) {
	var n int64
	err := tx.Table("device").
		Where("product_key = ? AND deleted_at IS NULL", productKey).
		Count(&n).Error
	return n, err
}

// RegistrationReadyForProductKey 与 ValidateProductKeyPublishedActive 相同条件（含 iot_product 已发布）
func (e *PlatformDeviceService) RegistrationReadyForProductKey(tx *gorm.DB, productKey string) (bool, error) {
	return e.productKeyAllowsDeviceProvisionWithDB(tx, productKey)
}

// ListIotProducts 分页列表
func (e *PlatformDeviceService) ListIotProducts(in IotProductListIn) ([]IotProductListItem, int64, error) {
	if e.Orm == nil {
		return nil, 0, fmt.Errorf("orm nil")
	}
	page := in.Page
	if page < 1 {
		page = 1
	}
	ps := in.PageSize
	if ps < 1 {
		ps = 20
	}
	if ps > 200 {
		ps = 200
	}

	q := e.Orm.Model(&models.IotProduct{})
	kw := strings.TrimSpace(in.Keyword)
	if kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name ILIKE ? OR product_key ILIKE ?", like, like)
	}
	if s := strings.TrimSpace(in.Status); s != "" {
		if !validIotProductStatus(s) {
			return nil, 0, ErrIotProductInvalidStatus
		}
		q = q.Where("status = ?", strings.ToLower(s))
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []models.IotProduct
	offset := (page - 1) * ps
	if err := q.Order("id DESC").Offset(offset).Limit(ps).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	out := make([]IotProductListItem, 0, len(rows))
	for _, r := range rows {
		pk := strings.TrimSpace(r.ProductKey)
		fc, _ := e.countFirmwareForProductKey(e.Orm, pk)
		dc, _ := e.countDevicesForProductKey(e.Orm, pk)
		ready, _ := e.RegistrationReadyForProductKey(e.Orm, pk)
		out = append(out, IotProductListItem{
			ID:                r.Id,
			ProductKey:        pk,
			Name:              r.Name,
			Category:          r.Category,
			Status:            r.Status,
			FirmwareCount:     fc,
			DeviceCount:       dc,
			RegistrationReady: ready,
			CreatedAt:         r.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:         r.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return out, total, nil
}

// GetIotProduct 详情
func (e *PlatformDeviceService) GetIotProduct(id int64) (*IotProductDetailOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	var r models.IotProduct
	if err := e.Orm.Where("id = ?", id).First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrIotProductNotFound
		}
		return nil, err
	}
	pk := strings.TrimSpace(r.ProductKey)
	fc, _ := e.countFirmwareForProductKey(e.Orm, pk)
	dc, _ := e.countDevicesForProductKey(e.Orm, pk)
	ready, _ := e.RegistrationReadyForProductKey(e.Orm, pk)
	comm := r.Communication
	if comm == nil {
		comm = json.RawMessage(`[]`)
	}
	return &IotProductDetailOut{
		IotProductListItem: IotProductListItem{
			ID:                r.Id,
			ProductKey:        pk,
			Name:              r.Name,
			Category:          r.Category,
			Status:            r.Status,
			FirmwareCount:     fc,
			DeviceCount:       dc,
			RegistrationReady: ready,
			CreatedAt:         r.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:         r.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
		Description:   r.Description,
		Communication: comm,
		DeviceType:    r.DeviceType,
	}, nil
}

// CreateIotProduct 创建
func (e *PlatformDeviceService) CreateIotProduct(in IotProductCreateIn) (*models.IotProduct, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	pk := normalizeProductKey(in.ProductKey)
	if err := validateProductKeyFormat(pk); err != nil {
		return nil, err
	}
	st := strings.ToLower(strings.TrimSpace(in.Status))
	if st == "" {
		st = "draft"
	}
	if !validIotProductStatus(st) {
		return nil, ErrIotProductInvalidStatus
	}
	var exists int64
	if err := e.Orm.Model(&models.IotProduct{}).
		Where("LOWER(TRIM(product_key)) = LOWER(TRIM(?))", pk).
		Count(&exists).Error; err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, ErrIotProductDuplicate
	}

	comm := in.Communication
	if len(comm) == 0 || string(comm) == "null" {
		comm = json.RawMessage(`[]`)
	} else if !json.Valid(comm) {
		return nil, fmt.Errorf("communication 须为合法 JSON")
	}

	row := models.IotProduct{
		ProductKey:    pk,
		Name:          strings.TrimSpace(in.Name),
		Category:      strings.TrimSpace(in.Category),
		Description:   strings.TrimSpace(in.Description),
		Communication: comm,
		DeviceType:    strings.TrimSpace(in.DeviceType),
		Status:        st,
		CreateBy:      in.CreateBy,
		UpdateBy:      in.CreateBy,
	}
	if row.Name == "" {
		row.Name = pk
	}
	if err := e.Orm.Create(&row).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return nil, ErrIotProductDuplicate
		}
		return nil, err
	}
	return &row, nil
}

// UpdateIotProduct 更新展示字段
func (e *PlatformDeviceService) UpdateIotProduct(id int64, in IotProductUpdateIn) error {
	if e.Orm == nil {
		return fmt.Errorf("orm nil")
	}
	var cnt int64
	if err := e.Orm.Model(&models.IotProduct{}).Where("id = ?", id).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt == 0 {
		return ErrIotProductNotFound
	}
	updates := map[string]interface{}{
		"updated_by": in.UpdateBy,
	}
	if n := strings.TrimSpace(in.Name); n != "" {
		updates["name"] = n
	}
	updates["category"] = strings.TrimSpace(in.Category)
	updates["description"] = strings.TrimSpace(in.Description)
	updates["device_type"] = strings.TrimSpace(in.DeviceType)
	if in.Communication != nil && len(in.Communication) > 0 {
		if !json.Valid(in.Communication) {
			return fmt.Errorf("communication 须为合法 JSON")
		}
		updates["communication"] = in.Communication
	}
	return e.Orm.Model(&models.IotProduct{}).Where("id = ?", id).Updates(updates).Error
}

// PublishIotProduct 发布
func (e *PlatformDeviceService) PublishIotProduct(id int64, updateBy int64) error {
	return e.setIotProductStatus(id, "published", updateBy)
}

// DisableIotProduct 禁用
func (e *PlatformDeviceService) DisableIotProduct(id int64, updateBy int64) error {
	return e.setIotProductStatus(id, "disabled", updateBy)
}

func (e *PlatformDeviceService) setIotProductStatus(id int64, status string, updateBy int64) error {
	if e.Orm == nil {
		return fmt.Errorf("orm nil")
	}
	if !validIotProductStatus(status) {
		return ErrIotProductInvalidStatus
	}
	res := e.Orm.Model(&models.IotProduct{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_by": updateBy,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrIotProductNotFound
	}
	return nil
}
