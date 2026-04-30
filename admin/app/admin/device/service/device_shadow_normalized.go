package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"go-admin/app/admin/device/models"
)

// NormalizedShadowOut 规范化影子 + 可选合并现有 JSON/Redis 影子热字段
type NormalizedShadowOut struct {
	DeviceID int64                 `json:"device_id"`
	Sn       string                `json:"sn"`
	Profile  NormalizedProfileOut  `json:"profile"`
	Battery  NormalizedBatteryOut  `json:"battery"`
	Location NormalizedLocationOut `json:"location"`
	Configs  []NormalizedConfigOut `json:"configs"`
	Legacy   *DeviceShadowView     `json:"legacy_shadow,omitempty"`
}

type NormalizedProfileOut struct {
	FirmwareVersion string     `json:"firmware_version"`
	HardwareVersion string     `json:"hardware_version"`
	OnlineStatus    int16      `json:"online_status"`
	OfflineAt       *time.Time `json:"offline_at,omitempty"`
	LastActiveAt    *time.Time `json:"last_active_at,omitempty"`
	FwUpgradedAt    *time.Time `json:"fw_upgraded_at,omitempty"`
	NetworkType     string     `json:"network_type"`
	Rssi            *int       `json:"rssi,omitempty"`
	ProductKey      string     `json:"product_key"`
	FirstOnlineAt   *time.Time `json:"first_online_at,omitempty"`
	OfflineReason   string     `json:"offline_reason"`
	ReportedAt      *time.Time `json:"reported_at,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

type NormalizedBatteryOut struct {
	MainPercent     *int16     `json:"main_percent,omitempty"`
	SpeakerPercent  *int16     `json:"speaker_percent,omitempty"`
	Charging        int16      `json:"charging"`
	EstRemainingSec *int64     `json:"est_remaining_sec,omitempty"`
	LowThreshold    *int16     `json:"low_threshold,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

type NormalizedLocationOut struct {
	Latitude       *float64   `json:"latitude,omitempty"`
	Longitude      *float64   `json:"longitude,omitempty"`
	LocationMode   string     `json:"location_mode"`
	AccuracyM      *float64   `json:"accuracy_m,omitempty"`
	GeofenceStatus string     `json:"geofence_status"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

type NormalizedConfigOut struct {
	ConfigType string     `json:"config_type"`
	Desired    string     `json:"desired"`
	Reported   string     `json:"reported"`
	SyncStatus int16      `json:"sync_status"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

// NormalizedShadowPutIn 管理端更新规范化影子（各区块可省略；指针字段 nil 表示不修改该列）
type NormalizedShadowPutIn struct {
	Profile  *NormalizedProfilePut  `json:"profile"`
	Battery  *NormalizedBatteryPut  `json:"battery"`
	Location *NormalizedLocationPut `json:"location"`
	Configs  []NormalizedConfigPut  `json:"configs"`
	Operator string                 `json:"-"`
}

type NormalizedProfilePut struct {
	FirmwareVersion *string    `json:"firmware_version"`
	HardwareVersion *string    `json:"hardware_version"`
	OnlineStatus    *int16     `json:"online_status"`
	OfflineAt       *time.Time `json:"offline_at"`
	LastActiveAt    *time.Time `json:"last_active_at"`
	FwUpgradedAt    *time.Time `json:"fw_upgraded_at"`
	NetworkType     *string    `json:"network_type"`
	Rssi            *int       `json:"rssi"`
	ProductKey      *string    `json:"product_key"`
	FirstOnlineAt   *time.Time `json:"first_online_at"`
	OfflineReason   *string    `json:"offline_reason"`
	ReportedAt      *time.Time `json:"reported_at"`
}

type NormalizedBatteryPut struct {
	MainPercent     *int16 `json:"main_percent"`
	SpeakerPercent  *int16 `json:"speaker_percent"`
	Charging        *int16 `json:"charging"`
	EstRemainingSec *int64 `json:"est_remaining_sec"`
	LowThreshold    *int16 `json:"low_threshold"`
}

type NormalizedLocationPut struct {
	Latitude       *float64 `json:"latitude"`
	Longitude      *float64 `json:"longitude"`
	LocationMode   *string  `json:"location_mode"`
	AccuracyM      *float64 `json:"accuracy_m"`
	GeofenceStatus *string  `json:"geofence_status"`
}

type NormalizedConfigPut struct {
	ConfigType string  `json:"config_type"`
	Desired    *string `json:"desired"`
	Reported   *string `json:"reported"`
	SyncStatus *int16  `json:"sync_status"`
}

// GetNormalizedShadow 读取规范化四表，并与 GetDeviceShadow 做热字段合并（规范化为底，Redis/JSON 覆盖）
func (e *PlatformDeviceService) GetNormalizedShadow(sn string) (*NormalizedShadowOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	sn = strings.TrimSpace(sn)
	if sn == "" {
		return nil, ErrPlatformDeviceInvalid
	}

	var dev struct {
		ID         int64  `gorm:"column:id"`
		Sn         string `gorm:"column:sn"`
		ProductKey string `gorm:"column:product_key"`
	}
	if err := e.Orm.Table("device").Select("id, sn, product_key").Where("sn = ? AND deleted_at IS NULL", sn).Take(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}
	did := dev.ID

	var prof models.DeviceShadowProfile
	hasProf := e.Orm.Where("device_id = ?", did).Take(&prof).Error == nil

	var bat models.DeviceShadowBattery
	hasBat := e.Orm.Where("device_id = ?", did).Take(&bat).Error == nil

	var loc models.DeviceShadowLocation
	hasLoc := e.Orm.Where("device_id = ?", did).Take(&loc).Error == nil

	var cfgs []models.DeviceShadowConfig
	_ = e.Orm.Where("device_id = ?", did).Order("config_type").Find(&cfgs).Error

	out := &NormalizedShadowOut{
		DeviceID: did,
		Sn:       dev.Sn,
		Profile:  defaultProfileOut(did, dev.ProductKey),
		Battery:  NormalizedBatteryOut{Charging: 0},
		Location: NormalizedLocationOut{},
		Configs:  nil,
	}
	if hasProf {
		out.Profile = profileModelToOut(&prof)
	} else {
		out.Profile.ProductKey = dev.ProductKey
	}
	if hasBat {
		out.Battery = batteryModelToOut(&bat)
	}
	if hasLoc {
		out.Location = locationModelToOut(&loc)
	}
	if len(cfgs) > 0 {
		out.Configs = make([]NormalizedConfigOut, 0, len(cfgs))
		for i := range cfgs {
			out.Configs = append(out.Configs, configModelToOut(&cfgs[i]))
		}
	}

	legacy, err := e.GetDeviceShadow(sn)
	if err == nil && legacy != nil {
		out.Legacy = legacy
		mergeLegacyIntoNormalized(out, legacy)
	}
	return out, nil
}

func defaultProfileOut(deviceID int64, productKey string) NormalizedProfileOut {
	return NormalizedProfileOut{
		FirmwareVersion: "",
		HardwareVersion: "",
		OnlineStatus:    0,
		ProductKey:      strings.TrimSpace(productKey),
		OfflineReason:   "",
	}
}

func profileModelToOut(p *models.DeviceShadowProfile) NormalizedProfileOut {
	o := NormalizedProfileOut{
		FirmwareVersion: p.FirmwareVersion,
		HardwareVersion: p.HardwareVersion,
		OnlineStatus:    p.OnlineStatus,
		OfflineAt:       p.OfflineAt,
		LastActiveAt:    p.LastActiveAt,
		FwUpgradedAt:    p.FwUpgradedAt,
		NetworkType:     p.NetworkType,
		Rssi:            p.Rssi,
		ProductKey:      p.ProductKey,
		FirstOnlineAt:   p.FirstOnlineAt,
		OfflineReason:   p.OfflineReason,
		ReportedAt:      p.ReportedAt,
	}
	if !p.CreatedAt.IsZero() {
		t := p.CreatedAt
		o.CreatedAt = &t
	}
	if !p.UpdatedAt.IsZero() {
		t := p.UpdatedAt
		o.UpdatedAt = &t
	}
	return o
}

func batteryModelToOut(b *models.DeviceShadowBattery) NormalizedBatteryOut {
	o := NormalizedBatteryOut{
		MainPercent:     b.MainPercent,
		SpeakerPercent:  b.SpeakerPercent,
		Charging:        b.Charging,
		EstRemainingSec: b.EstRemainingSec,
		LowThreshold:    b.LowThreshold,
	}
	if !b.UpdatedAt.IsZero() {
		t := b.UpdatedAt
		o.UpdatedAt = &t
	}
	return o
}

func locationModelToOut(l *models.DeviceShadowLocation) NormalizedLocationOut {
	o := NormalizedLocationOut{
		Latitude:       l.Latitude,
		Longitude:      l.Longitude,
		LocationMode:   l.LocationMode,
		AccuracyM:      l.AccuracyM,
		GeofenceStatus: l.GeofenceStatus,
	}
	if !l.UpdatedAt.IsZero() {
		t := l.UpdatedAt
		o.UpdatedAt = &t
	}
	return o
}

func configModelToOut(c *models.DeviceShadowConfig) NormalizedConfigOut {
	o := NormalizedConfigOut{
		ConfigType: c.ConfigType,
		Desired:    c.Desired,
		Reported:   c.Reported,
		SyncStatus: c.SyncStatus,
	}
	if !c.UpdatedAt.IsZero() {
		t := c.UpdatedAt
		o.UpdatedAt = &t
	}
	return o
}

func mergeLegacyIntoNormalized(out *NormalizedShadowOut, leg *DeviceShadowView) {
	if strings.TrimSpace(leg.FirmwareVersion) != "" {
		out.Profile.FirmwareVersion = leg.FirmwareVersion
	}
	// 与 GetDeviceShadow 一致聚合视图：用热路径覆盖规范化列
	if leg.Online {
		out.Profile.OnlineStatus = 1
	} else {
		out.Profile.OnlineStatus = 0
	}
	if leg.LastOnlineTime != nil {
		t := *leg.LastOnlineTime
		out.Profile.LastActiveAt = &t
	}
	if leg.Battery != nil {
		v := int16(*leg.Battery)
		if v > 100 {
			v = 100
		}
		if v < 0 {
			v = 0
		}
		out.Battery.MainPercent = &v
	}
}

// PutNormalizedShadow 事务内 upsert 规范化表；不写 device_shadow JSONB
func (e *PlatformDeviceService) PutNormalizedShadow(sn string, in *NormalizedShadowPutIn) error {
	if e.Orm == nil {
		return fmt.Errorf("orm nil")
	}
	if in == nil {
		return ErrPlatformDeviceInvalid
	}
	sn = strings.TrimSpace(sn)
	if sn == "" {
		return ErrPlatformDeviceInvalid
	}
	hasPayload := in.Profile != nil || in.Battery != nil || in.Location != nil || len(in.Configs) > 0
	if !hasPayload {
		return ErrPlatformDeviceInvalid
	}

	var dev struct {
		ID     int64  `gorm:"column:id"`
		Sn     string `gorm:"column:sn"`
		Status int16  `gorm:"column:status"`
	}
	if err := e.Orm.Table("device").Select("id, sn, status").Where("sn = ? AND deleted_at IS NULL", sn).Take(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPlatformDeviceNotFound
		}
		return err
	}
	if dev.Status != 1 {
		return fmt.Errorf("设备未处于可用状态，无法写入规范化影子")
	}

	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "admin"
	}

	return e.Orm.Transaction(func(tx *gorm.DB) error {
		did := dev.ID

		if in.Profile != nil {
			if err := upsertProfile(tx, did, in.Profile); err != nil {
				return err
			}
		}
		if in.Battery != nil {
			if err := upsertBattery(tx, did, in.Battery); err != nil {
				return err
			}
		}
		if in.Location != nil {
			if err := upsertLocation(tx, did, in.Location); err != nil {
				return err
			}
		}
		for i := range in.Configs {
			if err := upsertConfig(tx, did, &in.Configs[i]); err != nil {
				return err
			}
		}

		sum, _ := json.Marshal(in)
		return tx.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator) VALUES (?,?,?,?,?)`,
			did, dev.Sn, "admin_normalized_shadow", truncateEvent(string(sum)), op).Error
	})
}

func upsertProfile(tx *gorm.DB, deviceID int64, p *NormalizedProfilePut) error {
	row := models.DeviceShadowProfile{DeviceId: deviceID}
	if err := tx.Where("device_id = ?", deviceID).Take(&row).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row = models.DeviceShadowProfile{DeviceId: deviceID}
	}

	if p.FirmwareVersion != nil {
		row.FirmwareVersion = strings.TrimSpace(*p.FirmwareVersion)
	}
	if p.HardwareVersion != nil {
		row.HardwareVersion = strings.TrimSpace(*p.HardwareVersion)
	}
	if p.OnlineStatus != nil {
		row.OnlineStatus = *p.OnlineStatus
	}
	if p.OfflineAt != nil {
		row.OfflineAt = p.OfflineAt
	}
	if p.LastActiveAt != nil {
		row.LastActiveAt = p.LastActiveAt
	}
	if p.FwUpgradedAt != nil {
		row.FwUpgradedAt = p.FwUpgradedAt
	}
	if p.NetworkType != nil {
		row.NetworkType = strings.TrimSpace(*p.NetworkType)
	}
	if p.Rssi != nil {
		row.Rssi = p.Rssi
	}
	if p.ProductKey != nil {
		row.ProductKey = strings.TrimSpace(*p.ProductKey)
	}
	if p.FirstOnlineAt != nil {
		row.FirstOnlineAt = p.FirstOnlineAt
	}
	if p.OfflineReason != nil {
		row.OfflineReason = strings.TrimSpace(*p.OfflineReason)
	}
	if p.ReportedAt != nil {
		row.ReportedAt = p.ReportedAt
	}

	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"firmware_version", "hardware_version", "online_status", "offline_at", "last_active_at", "fw_upgraded_at", "network_type", "rssi", "product_key", "first_online_at", "offline_reason", "reported_at", "updated_at"}),
	}).Create(&row).Error
}

func upsertBattery(tx *gorm.DB, deviceID int64, b *NormalizedBatteryPut) error {
	row := models.DeviceShadowBattery{DeviceId: deviceID}
	if err := tx.Where("device_id = ?", deviceID).Take(&row).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row = models.DeviceShadowBattery{DeviceId: deviceID}
	}

	if b.MainPercent != nil {
		row.MainPercent = b.MainPercent
	}
	if b.SpeakerPercent != nil {
		row.SpeakerPercent = b.SpeakerPercent
	}
	if b.Charging != nil {
		row.Charging = *b.Charging
	}
	if b.EstRemainingSec != nil {
		row.EstRemainingSec = b.EstRemainingSec
	}
	if b.LowThreshold != nil {
		row.LowThreshold = b.LowThreshold
	}

	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"main_percent", "speaker_percent", "charging", "est_remaining_sec", "low_threshold", "updated_at"}),
	}).Create(&row).Error
}

func upsertLocation(tx *gorm.DB, deviceID int64, l *NormalizedLocationPut) error {
	row := models.DeviceShadowLocation{DeviceId: deviceID}
	if err := tx.Where("device_id = ?", deviceID).Take(&row).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row = models.DeviceShadowLocation{DeviceId: deviceID}
	}

	if l.Latitude != nil {
		row.Latitude = l.Latitude
	}
	if l.Longitude != nil {
		row.Longitude = l.Longitude
	}
	if l.LocationMode != nil {
		row.LocationMode = strings.TrimSpace(*l.LocationMode)
	}
	if l.AccuracyM != nil {
		row.AccuracyM = l.AccuracyM
	}
	if l.GeofenceStatus != nil {
		row.GeofenceStatus = strings.TrimSpace(*l.GeofenceStatus)
	}

	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"latitude", "longitude", "location_mode", "accuracy_m", "geofence_status", "updated_at"}),
	}).Create(&row).Error
}

func upsertConfig(tx *gorm.DB, deviceID int64, c *NormalizedConfigPut) error {
	ct := strings.TrimSpace(c.ConfigType)
	if ct == "" {
		return fmt.Errorf("config_type 不能为空")
	}
	row := models.DeviceShadowConfig{DeviceId: deviceID, ConfigType: ct}
	if err := tx.Where("device_id = ? AND config_type = ?", deviceID, ct).Take(&row).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row = models.DeviceShadowConfig{DeviceId: deviceID, ConfigType: ct}
	}

	if c.Desired != nil {
		row.Desired = *c.Desired
	}
	if c.Reported != nil {
		row.Reported = *c.Reported
	}
	if c.SyncStatus != nil {
		row.SyncStatus = *c.SyncStatus
	}

	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_id"}, {Name: "config_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"desired", "reported", "sync_status", "updated_at"}),
	}).Create(&row).Error
}
