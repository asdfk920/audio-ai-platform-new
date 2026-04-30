package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go-admin/app/admin/device/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 云端认证激活（后台）：与设备侧 HMAC 规则对齐（参见原设备服务 canonicalPayload + HMAC-SHA256(secret)）
var (
	ErrDeviceCloudTimestampInvalid   = errors.New("时间戳无效或超出允许范围（±5分钟）")
	ErrDeviceCloudNonceReplay        = errors.New("随机数重复，拒绝重放请求")
	ErrDeviceCloudSignatureInvalid   = errors.New("签名验证失败")
	ErrDeviceCloudSecretInvalid      = errors.New("设备密钥不正确")
	ErrDeviceCloudNotInactive        = errors.New("设备不是未激活状态，无法执行云端认证激活")
	ErrDeviceCloudProductKeyMismatch = errors.New("产品密钥与设备记录不一致")
	ErrProductDisabledOrDraft        = errors.New("产品已禁用或未发布，无法激活")
)

const cloudActivateTimestampWindow = 300 * time.Second // ±5 分钟
const cloudActivateNonceMaxLen     = 256
const cloudActivateNonceMinLen     = 4

// CloudActivateIn 设备注册/激活风格请求体（与设备端 JSON 字段一致）
type CloudActivateIn struct {
	ProductKey      string `json:"product_key"`
	Sn              string `json:"sn"`
	Secret          string `json:"secret"`
	Timestamp       int64  `json:"timestamp"`
	Nonce           string `json:"nonce"`
	Signature       string `json:"signature"`
	FirmwareVersion string `json:"firmware_version"`
	Ip              string `json:"ip"`
}

// CloudActivateOut 激活成功结果
type CloudActivateOut struct {
	DeviceID int64  `json:"device_id"`
	Sn       string `json:"sn"`
	Status   int16  `json:"status"`
}

func validateCloudTimestamp(ts int64) error {
	if ts <= 0 {
		return ErrDeviceCloudTimestampInvalid
	}
	var target time.Time
	if ts > 1_000_000_000_000 {
		target = time.UnixMilli(ts)
	} else {
		target = time.Unix(ts, 0)
	}
	now := time.Now()
	if target.Before(now.Add(-cloudActivateTimestampWindow)) || target.After(now.Add(cloudActivateTimestampWindow)) {
		return ErrDeviceCloudTimestampInvalid
	}
	return nil
}

func canonicalSignPayload(payload map[string]string, nonce string, timestamp int64) string {
	keys := make([]string, 0, len(payload)+2)
	clean := make(map[string]string, len(payload)+2)
	for k, v := range payload {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		clean[k] = strings.TrimSpace(v)
		keys = append(keys, k)
	}
	clean["nonce"] = strings.TrimSpace(nonce)
	clean["timestamp"] = strconv.FormatInt(timestamp, 10)
	keys = append(keys, "nonce", "timestamp")
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		parts = append(parts, key+"="+clean[key])
	}
	return strings.Join(parts, "&")
}

func verifyCloudSignature(plainSecret string, payload map[string]string, nonce string, timestamp int64, signature string) bool {
	signature = strings.ToLower(strings.TrimSpace(signature))
	if signature == "" {
		return false
	}
	expected := ComputeCloudActivationSignatureFromPayload(plainSecret, payload, nonce, timestamp)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ComputeCloudActivationSignatureFromPayload 十六进制小写 HMAC-SHA256，与设备端 / 云端认证校验一致。
func ComputeCloudActivationSignatureFromPayload(plainSecret string, signPayload map[string]string, nonce string, timestamp int64) string {
	body := canonicalSignPayload(signPayload, nonce, timestamp)
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(plainSecret)))
	_, _ = mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

// ComputeCloudActivationSignature 按与 ActivateDeviceCloudAuth 相同的参与字段计算签名（sn 大写、product_key trim）。
func ComputeCloudActivationSignature(plainSecret, snUpper, productKey string, timestamp int64, nonce, firmwareVersion, ip string) string {
	signMap := map[string]string{
		"sn":          strings.TrimSpace(snUpper),
		"product_key": strings.TrimSpace(productKey),
	}
	if v := strings.TrimSpace(firmwareVersion); v != "" {
		signMap["firmware_version"] = v
	}
	if v := strings.TrimSpace(ip); v != "" {
		signMap["ip"] = v
	}
	return ComputeCloudActivationSignatureFromPayload(plainSecret, signMap, nonce, timestamp)
}

func isPGUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate key") || strings.Contains(s, "unique constraint")
}

// ActivateDeviceCloudAuth 四步：时间戳、nonce 防重放、产品与设备状态、密钥与签名，成功则置 status=1。
func (e *PlatformDeviceService) ActivateDeviceCloudAuth(in *CloudActivateIn, operator string) (*CloudActivateOut, error) {
	if e.Orm == nil || in == nil {
		return nil, fmt.Errorf("orm nil")
	}
	pk := strings.TrimSpace(in.ProductKey)
	sn := strings.ToUpper(strings.TrimSpace(in.Sn))
	secret := strings.TrimSpace(in.Secret)
	nonce := strings.TrimSpace(in.Nonce)
	sig := strings.TrimSpace(in.Signature)
	if pk == "" || sn == "" || secret == "" || nonce == "" || sig == "" {
		return nil, ErrPlatformDeviceInvalid
	}
	nr := utf8.RuneCountInString(nonce)
	if nr < cloudActivateNonceMinLen || nr > cloudActivateNonceMaxLen {
		return nil, ErrPlatformDeviceInvalid
	}

	if err := validateCloudTimestamp(in.Timestamp); err != nil {
		return nil, err
	}

	if err := e.ValidateProductKeyPublishedActive(pk); err != nil {
		return nil, err
	}

	var iotRow models.IotProduct
	if err := e.Orm.Where("deleted_at IS NULL AND LOWER(TRIM(product_key)) = LOWER(TRIM(?))", pk).First(&iotRow).Error; err == nil {
		if strings.ToLower(strings.TrimSpace(iotRow.Status)) == "disabled" {
			return nil, ErrProductDisabledOrDraft
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var dev models.Device
	if err := e.Orm.Where("sn = ? AND deleted_at IS NULL", sn).First(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(dev.ProductKey), pk) {
		return nil, ErrDeviceCloudProductKeyMismatch
	}
	if dev.Status != 3 {
		return nil, ErrDeviceCloudNotInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dev.DeviceSecret), []byte(secret)); err != nil {
		return nil, ErrDeviceCloudSecretInvalid
	}

	signMap := map[string]string{
		"sn":          sn,
		"product_key": strings.TrimSpace(dev.ProductKey),
	}
	if v := strings.TrimSpace(in.FirmwareVersion); v != "" {
		signMap["firmware_version"] = v
	}
	if v := strings.TrimSpace(in.Ip); v != "" {
		signMap["ip"] = v
	}
	if !verifyCloudSignature(secret, signMap, nonce, in.Timestamp, sig) {
		return nil, ErrDeviceCloudSignatureInvalid
	}

	op := strings.TrimSpace(operator)
	if op == "" {
		op = "admin"
	}

	out := &CloudActivateOut{Sn: sn, Status: 1, DeviceID: int64(dev.Id)}

	err := e.Orm.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`INSERT INTO device_activate_nonce (sn, nonce) VALUES (?, ?)`, sn, nonce).Error; err != nil {
			if isPGUniqueViolation(err) {
				return ErrDeviceCloudNonceReplay
			}
			return err
		}

		res := tx.Exec(`UPDATE device SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL AND status = ?`,
			1, dev.Id, 3)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrDeviceCloudNotInactive
		}
		msg := "云端认证激活为正常（HMAC 校验通过）"
		return tx.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator) VALUES (?,?,?,?,?)`,
			dev.Id, sn, "cloud_activate", truncateEvent(msg), op).Error
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

// AdminTrustedCloudActivate 已登录管理员在后台将「未激活」设备置为正常。
// 不接收出厂密钥与 HMAC：云端仅存密钥哈希无法还原明文；本路径依赖 JWT + 管理端 RBAC，业务校验与 ActivateDeviceCloudAuth 中的产品与状态检查一致（不含密钥与签名）。
func (e *PlatformDeviceService) AdminTrustedCloudActivate(sn, productKey, operator string) (*CloudActivateOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	pk := strings.TrimSpace(productKey)
	sn = strings.ToUpper(strings.TrimSpace(sn))
	if pk == "" || sn == "" {
		return nil, ErrPlatformDeviceInvalid
	}

	if err := e.ValidateProductKeyPublishedActive(pk); err != nil {
		return nil, err
	}

	var iotRow models.IotProduct
	if err := e.Orm.Where("deleted_at IS NULL AND LOWER(TRIM(product_key)) = LOWER(TRIM(?))", pk).First(&iotRow).Error; err == nil {
		if strings.ToLower(strings.TrimSpace(iotRow.Status)) == "disabled" {
			return nil, ErrProductDisabledOrDraft
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var dev models.Device
	if err := e.Orm.Where("sn = ? AND deleted_at IS NULL", sn).First(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(dev.ProductKey), pk) {
		return nil, ErrDeviceCloudProductKeyMismatch
	}
	if dev.Status != 3 {
		return nil, ErrDeviceCloudNotInactive
	}

	op := strings.TrimSpace(operator)
	if op == "" {
		op = "admin"
	}

	out := &CloudActivateOut{Sn: sn, Status: 1, DeviceID: int64(dev.Id)}

	err := e.Orm.Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(`UPDATE device SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL AND status = ?`,
			1, dev.Id, 3)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrDeviceCloudNotInactive
		}
		msg := "云端认证激活为正常（管理员后台可信操作，无设备端 HMAC）"
		return tx.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator) VALUES (?,?,?,?,?)`,
			dev.Id, sn, "cloud_activate_admin", truncateEvent(msg), op).Error
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
