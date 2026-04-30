package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go-admin/app/admin/device/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	snProvisionPattern = regexp.MustCompile(`^[A-Za-z0-9]{16,32}$`)
	macPattern         = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)
)

// ProvisionIn 后台预注册设备入参
type ProvisionIn struct {
	Sn               string
	ProductKey       string
	Model            string
	Mac              string
	AdminDisplayName string
	AdminRemark      string
	PlainPresetSecret string
	RequireMAC       bool
	CreateBy         int
}

// ProvisionOut 创建成功返回（device_secret 仅此时明文返回）
type ProvisionOut struct {
	DeviceID           int64  `json:"device_id"`
	Sn                 string `json:"sn"`
	ProductKey         string `json:"product_key"`
	DeviceSecret       string `json:"device_secret"`
	Model              string `json:"model"`
	Mac                string `json:"mac"`
	Status             int16  `json:"status"`
	AdminDisplayName   string `json:"admin_display_name"`
	AdminRemark        string `json:"admin_remark"`
	CreatedAt          string `json:"created_at"`
	// Bootstrap* 为建档时生成的示例参数与签名，与 POST .../activate-cloud 规则一致；激活时须使用新的 nonce（不可重复），时间戳须在 ±5 分钟内。
	BootstrapTimestamp int64  `json:"bootstrap_timestamp"`
	BootstrapNonce       string `json:"bootstrap_nonce"`
	BootstrapSignature   string `json:"bootstrap_signature"`
}

// productKeyAllowsDeviceProvisionWithDB 允许预注册：① ota_firmware 有已发布且启用的固件；或 ② iot_product 为 published（可无首包固件先登记设备）
func (e *PlatformDeviceService) productKeyAllowsDeviceProvisionWithDB(db *gorm.DB, productKey string) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("orm nil")
	}
	pk := strings.TrimSpace(productKey)
	if pk == "" {
		return false, nil
	}
	var n int64
	err := db.Table("ota_firmware").
		Where("product_key = ? AND deleted_at IS NULL AND publish_status = ? AND fw_status = ?", pk, 2, 1).
		Limit(1).
		Count(&n).Error
	if err != nil {
		return false, err
	}
	otaOK := n > 0
	var iotPub int64
	if !otaOK {
		err = db.Table("iot_product").
			Where("deleted_at IS NULL AND LOWER(TRIM(product_key)) = LOWER(TRIM(?)) AND status = ?", pk, "published").
			Limit(1).
			Count(&iotPub).Error
		if err != nil {
			return false, err
		}
	}
	iotOK := iotPub > 0
	ok := otaOK || iotOK
	return ok, nil
}

// ValidateProductKeyPublishedActive 产品线合法且允许预注册（见 productKeyAllowsDeviceProvisionWithDB）
func (e *PlatformDeviceService) ValidateProductKeyPublishedActive(productKey string) error {
	if e.Orm == nil {
		return fmt.Errorf("orm nil")
	}
	pk := strings.TrimSpace(productKey)
	if pk == "" {
		return ErrPlatformDeviceInvalid
	}
	ok, err := e.productKeyAllowsDeviceProvisionWithDB(e.Orm, pk)
	if err != nil {
		return err
	}
	if !ok {
		return ErrProductKeyNotActive
	}
	return nil
}

func normalizeMACInput(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("empty")
	}
	s = strings.ReplaceAll(s, "-", ":")
	s = strings.ReplaceAll(s, " ", "")
	if !strings.Contains(s, ":") {
		if len(s) != 12 {
			return "", ErrPlatformDeviceMACFormat
		}
		for _, c := range s {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return "", ErrPlatformDeviceMACFormat
			}
		}
		var b strings.Builder
		for i := 0; i < 12; i += 2 {
			if i > 0 {
				b.WriteByte(':')
			}
			b.WriteString(strings.ToUpper(s[i : i+2]))
		}
		s = b.String()
	} else {
		parts := strings.Split(s, ":")
		if len(parts) != 6 {
			return "", ErrPlatformDeviceMACFormat
		}
		var b strings.Builder
		for i, p := range parts {
			if len(p) != 2 {
				return "", ErrPlatformDeviceMACFormat
			}
			if i > 0 {
				b.WriteByte(':')
			}
			b.WriteString(strings.ToUpper(p))
		}
		s = b.String()
	}
	if !macPattern.MatchString(s) {
		return "", ErrPlatformDeviceMACFormat
	}
	return s, nil
}

// RegisterDeviceWithOptions 出厂预注册（status=3），device_secret 存 bcrypt 哈希
func (e *PlatformDeviceService) RegisterDeviceWithOptions(in *ProvisionIn) (*ProvisionOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	sn := strings.ToUpper(strings.TrimSpace(in.Sn))
	pk := strings.TrimSpace(in.ProductKey)
	model := strings.TrimSpace(in.Model)
	macIn := strings.TrimSpace(in.Mac)
	name := strings.TrimSpace(in.AdminDisplayName)
	remark := strings.TrimSpace(in.AdminRemark)

	if sn == "" || pk == "" {
		return nil, ErrPlatformDeviceInvalid
	}
	if !snProvisionPattern.MatchString(sn) {
		return nil, ErrPlatformDeviceSNFormat
	}

	var macNorm string
	if in.RequireMAC {
		if macIn == "" {
			return nil, ErrPlatformDeviceInvalid
		}
		var err error
		macNorm, err = normalizeMACInput(macIn)
		if err != nil {
			return nil, err
		}
	} else if macIn != "" {
		var err error
		macNorm, err = normalizeMACInput(macIn)
		if err != nil {
			return nil, err
		}
	}

	if err := e.ValidateProductKeyPublishedActive(pk); err != nil {
		return nil, err
	}

	var cnt int64
	if err := e.Orm.Model(&models.Device{}).Where("sn = ?", sn).Count(&cnt).Error; err != nil {
		return nil, err
	}
	if cnt > 0 {
		return nil, ErrPlatformDeviceDuplicate
	}

	if macNorm != "" {
		if err := e.Orm.Model(&models.Device{}).Where("mac = ? AND mac <> ''", macNorm).Count(&cnt).Error; err != nil {
			return nil, err
		}
		if cnt > 0 {
			return nil, ErrPlatformDeviceMacDuplicate
		}
	}

	plain := strings.TrimSpace(in.PlainPresetSecret)
	if plain != "" {
		if len(plain) < 8 || len(plain) > 128 {
			return nil, ErrPlatformDeviceInvalid
		}
	} else {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		plain = hex.EncodeToString(b)
	}

	hashB, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	dev := models.Device{
		Sn:               sn,
		ProductKey:       pk,
		DeviceSecret:     string(hashB),
		FirmwareVersion:  "",
		HardwareVersion:  "",
		Model:            model,
		Mac:              macNorm,
		Ip:               "",
		OnlineStatus:     0,
		Status:           3,
		AdminDisplayName: name,
		AdminRemark:      remark,
	}
	if in.CreateBy > 0 {
		dev.CreateBy = in.CreateBy
		dev.UpdateBy = in.CreateBy
	}

	if err := e.Orm.Create(&dev).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") ||
			strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, ErrPlatformDeviceDuplicate
		}
		return nil, err
	}

	bootstrapTS := time.Now().Unix()
	nonceBuf := make([]byte, 16)
	if _, err := rand.Read(nonceBuf); err != nil {
		return nil, err
	}
	bootstrapNonce := hex.EncodeToString(nonceBuf)
	bootstrapSig := ComputeCloudActivationSignature(plain, sn, pk, bootstrapTS, bootstrapNonce, "", "")

	return &ProvisionOut{
		DeviceID:         int64(dev.Id),
		Sn:               sn,
		ProductKey:       pk,
		DeviceSecret:     plain,
		Model:            model,
		Mac:              macNorm,
		Status:           3,
		AdminDisplayName: name,
		AdminRemark:      remark,
		CreatedAt:        dev.CreatedAt.Format("2006-01-02 15:04:05"),

		BootstrapTimestamp: bootstrapTS,
		BootstrapNonce:     bootstrapNonce,
		BootstrapSignature: bootstrapSig,
	}, nil
}

// RegisterDevice 兼容旧调用：仅 sn/product_key/model，不要求 MAC（不推荐）
func (e *PlatformDeviceService) RegisterDevice(sn, productKey, model string) error {
	_, err := e.RegisterDeviceWithOptions(&ProvisionIn{
		Sn:         strings.TrimSpace(sn),
		ProductKey: strings.TrimSpace(productKey),
		Model:      strings.TrimSpace(model),
		RequireMAC: false,
	})
	return err
}
