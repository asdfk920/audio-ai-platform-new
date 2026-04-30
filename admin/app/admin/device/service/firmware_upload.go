package service

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MaxFirmwareUploadBytes 单文件默认上限（128MB），与 multipart 解析上限一致
const MaxFirmwareUploadBytes int64 = 128 << 20

// FirmwareUploadIn 固件入库入参（文件已保存且已计算哈希）
type FirmwareUploadIn struct {
	ProductKey    string
	Version       string
	VersionCode   *int
	DeviceModels  string
	ForceUpdate   bool
	MinSysVersion string
	Description   string
	FwStatus      int16 // 1 启用 2 禁用
	FilePath      string // 相对路径 static/uploadfile/...（无首尾歧义时由上层写入 DB 前加 /）
	FileSize      int64
	MD5Hex        string
	SHA256Hex     string
	Operator      string
}

// FirmwareUploadOut 上传返回
type FirmwareUploadOut struct {
	FirmwareID int64     `json:"firmware_id"`
	Version    string    `json:"version"`
	FileURL    string    `json:"file_url"`
	FileSize   int64     `json:"file_size"`
	FileMd5    string    `json:"file_md5"`
	Status     int16     `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	Message    string    `json:"message"`
}

var (
	ErrFirmwareDuplicateVersion = errors.New("该版本已存在，请勿重复上传")
	ErrFirmwareVersionFormat    = errors.New("版本号格式不规范，请使用如 v1.2.3")
	ErrFirmwareFileType         = errors.New("仅支持 bin 或 zip 格式的固件包")
	ErrFirmwareFileTooLarge     = errors.New("文件大小超出限制")
	ErrFirmwareProductKey       = errors.New("产品线无效：请先在设备侧使用该 product_key 或已有固件记录")
	ErrIotProductDisabled       = errors.New("产品线已禁用，无法上传固件")
	ErrFirmwareChecksum         = errors.New("文件校验失败，请重新上传")
	ErrFirmwareStorage          = errors.New("上传失败，请重试")
)

var firmwareVersionRegexp = regexp.MustCompile(`(?i)^v?\d+\.\d+\.\d+([.\-+_][0-9A-Za-z-]+)*$`)

func validateFirmwareVersion(ver string) bool {
	s := strings.TrimSpace(ver)
	if len(s) == 0 || len(s) > 32 {
		return false
	}
	return firmwareVersionRegexp.MatchString(s)
}

// ValidateFirmwareVersionFormat 校验版本号格式（如 v1.2.3）
func ValidateFirmwareVersionFormat(ver string) error {
	if !validateFirmwareVersion(ver) {
		return ErrFirmwareVersionFormat
	}
	return nil
}

func firmwareExtOK(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".bin" || ext == ".zip"
}

func safeProductKeyDir(pk string) string {
	pk = strings.TrimSpace(pk)
	if pk == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range pk {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		case r == '_' || r == '-' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	s := b.String()
	if s == "" {
		return "unknown"
	}
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

// ValidateProductKeyForFirmware 产品线：device 或已有 ota_firmware 中存在该 product_key
func (e *PlatformDeviceService) ValidateProductKeyForFirmware(productKey string) error {
	if e.Orm == nil {
		return fmt.Errorf("orm nil")
	}
	pk := strings.TrimSpace(productKey)
	if pk == "" {
		return ErrPlatformDeviceInvalid
	}
	var n int64
	if err := e.Orm.Table("device").Where("product_key = ?", pk).Limit(1).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	if err := e.Orm.Table("ota_firmware").Where("product_key = ? AND deleted_at IS NULL", pk).Limit(1).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	// iot_product 已登记：允许冷启动首包（无需先有设备或固件行）
	var iotSt string
	if err := e.Orm.Table("iot_product").
		Select("status").
		Where("deleted_at IS NULL AND LOWER(TRIM(product_key)) = LOWER(TRIM(?))", pk).
		Limit(1).
		Scan(&iotSt).Error; err != nil {
		return err
	}
	if iotSt != "" {
		if strings.EqualFold(iotSt, "disabled") {
			return ErrIotProductDisabled
		}
		return nil
	}
	return ErrFirmwareProductKey
}

// normalizeFirmwareFileURL 存库：以 / 开头的静态路径
func normalizeFirmwareFileURL(relPath string) string {
	s := strings.TrimSpace(relPath)
	s = filepath.ToSlash(s)
	s = strings.TrimPrefix(s, "/")
	if strings.HasPrefix(s, "static/") {
		return "/" + s
	}
	return "/" + strings.TrimPrefix(s, "/")
}

// UploadFirmware 固件入库（文件已保存且哈希已计算）
func (e *PlatformDeviceService) UploadFirmware(in *FirmwareUploadIn) (*FirmwareUploadOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || strings.TrimSpace(in.ProductKey) == "" || strings.TrimSpace(in.Version) == "" {
		return nil, ErrPlatformDeviceInvalid
	}
	if strings.TrimSpace(in.FilePath) == "" || in.FileSize <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}
	md5h := strings.ToLower(strings.TrimSpace(in.MD5Hex))
	if md5h == "" {
		return nil, ErrFirmwareChecksum
	}

	ver := strings.TrimSpace(in.Version)
	if !validateFirmwareVersion(ver) {
		return nil, ErrFirmwareVersionFormat
	}

	if err := e.ValidateProductKeyForFirmware(in.ProductKey); err != nil {
		return nil, err
	}

	vc := 0
	if in.VersionCode != nil && *in.VersionCode > 0 {
		vc = *in.VersionCode
	} else {
		vc = guessVersionCode(ver)
	}

	ut := int16(1)
	if in.ForceUpdate {
		ut = 2
	}
	fwSt := in.FwStatus
	if fwSt != 1 && fwSt != 2 {
		fwSt = 1
	}
	pub := int16(2) // 已发布

	desc := strings.TrimSpace(in.Description)
	models := strings.TrimSpace(in.DeviceModels)
	minSys := strings.TrimSpace(in.MinSysVersion)
	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "admin"
	}

	fileURL := normalizeFirmwareFileURL(in.FilePath)
	if len(fileURL) > 255 {
		return nil, fmt.Errorf("文件路径过长")
	}

	sha := strings.ToLower(strings.TrimSpace(in.SHA256Hex))

	var releaseNote interface{}
	if desc != "" {
		releaseNote = desc
	}

	pk := strings.TrimSpace(in.ProductKey)

	var out struct {
		ID        int64     `gorm:"column:id"`
		CreatedAt time.Time `gorm:"column:created_at"`
	}

	err := e.Orm.Transaction(func(tx *gorm.DB) error {
		var exists int64
		if err := tx.Table("ota_firmware").Where("product_key = ? AND version = ? AND deleted_at IS NULL", pk, ver).Count(&exists).Error; err != nil {
			return err
		}
		if exists > 0 {
			return ErrFirmwareDuplicateVersion
		}
		return tx.Raw(`
INSERT INTO ota_firmware (
  product_key, version, file_url, file_size, md5, sha256, upgrade_type, publish_status, release_note,
  version_code, min_sys_version, device_models, fw_status, creator
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
RETURNING id, created_at`,
			pk,
			ver,
			fileURL,
			in.FileSize,
			md5h,
			sha,
			ut,
			pub,
			releaseNote,
			vc,
			minSys,
			models,
			fwSt,
			op,
		).Scan(&out).Error
	})
	if err != nil {
		if errors.Is(err, ErrFirmwareDuplicateVersion) {
			return nil, err
		}
		le := strings.ToLower(err.Error())
		if strings.Contains(le, "unique") || strings.Contains(le, "duplicate key") {
			return nil, ErrFirmwareDuplicateVersion
		}
		return nil, err
	}

	return &FirmwareUploadOut{
		FirmwareID: out.ID,
		Version:    ver,
		FileURL:    fileURL,
		FileSize:   in.FileSize,
		FileMd5:    md5h,
		Status:     fwSt,
		CreatedAt:  out.CreatedAt,
		Message:    "固件上传成功",
	}, nil
}

// SaveFirmwareUploadedFileWithPK 将上传流保存到 static/uploadfile/firmware/{pk}/{uuid}{ext}，并计算 MD5/SHA256
func SaveFirmwareUploadedFileWithPK(productKey string, originalName string, src io.Reader, maxBytes int64) (relPath string, size int64, md5hex, sha256hex string, err error) {
	if maxBytes <= 0 {
		maxBytes = MaxFirmwareUploadBytes
	}
	if !firmwareExtOK(originalName) {
		return "", 0, "", "", ErrFirmwareFileType
	}
	ext := strings.ToLower(filepath.Ext(originalName))
	sub := safeProductKeyDir(productKey)
	dir := filepath.Join("static", "uploadfile", "firmware", sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, "", "", fmt.Errorf("%w: %v", ErrFirmwareStorage, err)
	}
	name := uuid.New().String() + ext
	full := filepath.Join(dir, name)
	f, err := os.Create(full)
	if err != nil {
		return "", 0, "", "", fmt.Errorf("%w: %v", ErrFirmwareStorage, err)
	}
	defer f.Close()

	h1 := md5.New()
	h2 := sha256.New()
	lr := io.LimitReader(src, maxBytes+1)
	written, err := io.Copy(f, io.TeeReader(lr, io.MultiWriter(h1, h2)))
	if err != nil {
		_ = os.Remove(full)
		return "", 0, "", "", fmt.Errorf("%w: %v", ErrFirmwareStorage, err)
	}
	if written > maxBytes {
		_ = os.Remove(full)
		return "", 0, "", "", ErrFirmwareFileTooLarge
	}
	return filepath.ToSlash(filepath.Join("static", "uploadfile", "firmware", sub, name)), written, hex.EncodeToString(h1.Sum(nil)), hex.EncodeToString(h2.Sum(nil)), nil
}
