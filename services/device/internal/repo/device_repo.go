// Package repo 数据访问层：PostgreSQL 查询封装，logic 只依赖本包函数与行模型。
package repo

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// DeviceRow 设备表行模型
type DeviceRow struct {
	ID              int64
	SN              string
	ProductKey      string
	Mac             string
	FirmwareVersion string
	HardwareVersion string
	IP              string
	Status          int16
	OnlineStatus    int16
	Secret          string // 设备密钥（加密存储）
	LastActiveAt    *time.Time
}

// BoundDeviceRow 当前用户已绑定设备（影子上报 / 状态查询共用），含设备状态与活跃时间。
type BoundDeviceRow struct {
	DeviceID        int64
	SN              string
	ProductKey      string
	Mac             string
	FirmwareVersion string
	IP              string
	Status          int16
	OnlineStatus    int16
	LastActiveAt    *time.Time
	Model           string // 设备型号
}

// GetDeviceBySN 根据 SN 查询设备（含 Secret）
func GetDeviceBySN(ctx context.Context, db *sql.DB, sn string) (*DeviceRow, error) {
	sn = strings.ToUpper(strings.TrimSpace(sn))
	if sn == "" {
		return nil, sql.ErrNoRows
	}

	var r DeviceRow
	var lastActiveAt sql.NullTime

	err := db.QueryRowContext(ctx, `
		SELECT id, sn, product_key, mac, firmware_version, hardware_version, 
		       ip, status, online_status, secret, last_active_at
		FROM device
		WHERE sn = $1
		LIMIT 1`, sn).Scan(
		&r.ID, &r.SN, &r.ProductKey, &r.Mac, &r.FirmwareVersion,
		&r.HardwareVersion, &r.IP, &r.Status, &r.OnlineStatus,
		&r.Secret, &lastActiveAt,
	)

	if err != nil {
		return nil, err
	}

	if lastActiveAt.Valid {
		t := lastActiveAt.Time
		r.LastActiveAt = &t
	}

	return &r, nil
}

// GetBoundDeviceByUserAndSN 校验 user 与 SN 的活跃绑定并返回设备行（不筛 status，由 logic 层拒绝非「正常」态）。
func GetBoundDeviceByUserAndSN(ctx context.Context, db *sql.DB, userID int64, snNorm string) (*BoundDeviceRow, error) {
	snNorm = strings.ToUpper(strings.TrimSpace(snNorm))
	if userID <= 0 || snNorm == "" {
		return nil, sql.ErrNoRows
	}
	var r BoundDeviceRow
	var last sql.NullTime
	err := db.QueryRowContext(ctx, `
SELECT d.id, d.sn, d.product_key, d.mac, d.firmware_version, d.ip, d.status, d.online_status, d.last_active_at, d.model
FROM device d
INNER JOIN user_device_bind udb ON udb.device_id = d.id AND udb.status = 1
WHERE udb.user_id = $1 AND d.sn = $2
LIMIT 1`, userID, snNorm).Scan(
		&r.DeviceID, &r.SN, &r.ProductKey, &r.Mac, &r.FirmwareVersion, &r.IP,
		&r.Status, &r.OnlineStatus, &last, &r.Model,
	)
	if err != nil {
		return nil, err
	}
	if last.Valid {
		t := last.Time
		r.LastActiveAt = &t
	}
	return &r, nil
}

// GetBoundDeviceByUserAndID 校验 user 与设备 ID 的活跃绑定并返回设备行。
func GetBoundDeviceByUserAndID(ctx context.Context, db *sql.DB, userID int64, deviceID int64) (*BoundDeviceRow, error) {
	if userID <= 0 || deviceID <= 0 {
		return nil, sql.ErrNoRows
	}
	var r BoundDeviceRow
	var last sql.NullTime
	err := db.QueryRowContext(ctx, `
SELECT d.id, d.sn, d.product_key, d.mac, d.firmware_version, d.ip, d.status, d.online_status, d.last_active_at, d.model
FROM device d
INNER JOIN user_device_bind udb ON udb.device_id = d.id AND udb.status = 1
WHERE udb.user_id = $1 AND d.id = $2
LIMIT 1`, userID, deviceID).Scan(
		&r.DeviceID, &r.SN, &r.ProductKey, &r.Mac, &r.FirmwareVersion, &r.IP,
		&r.Status, &r.OnlineStatus, &last, &r.Model,
	)
	if err != nil {
		return nil, err
	}
	if last.Valid {
		t := last.Time
		r.LastActiveAt = &t
	}
	return &r, nil
}

// GetDeviceByID 根据设备 ID 查询设备（不含 Secret）
func GetDeviceByID(ctx context.Context, db *sql.DB, deviceID int64) (*DeviceRow, error) {
	if deviceID <= 0 {
		return nil, sql.ErrNoRows
	}

	var r DeviceRow
	var lastActiveAt sql.NullTime

	err := db.QueryRowContext(ctx, `
		SELECT id, sn, product_key, mac, firmware_version, hardware_version, 
		       ip, status, online_status, last_active_at
		FROM device
		WHERE id = $1
		LIMIT 1`, deviceID).Scan(
		&r.ID, &r.SN, &r.ProductKey, &r.Mac, &r.FirmwareVersion,
		&r.HardwareVersion, &r.IP, &r.Status, &r.OnlineStatus, &lastActiveAt,
	)

	if err != nil {
		return nil, err
	}

	if lastActiveAt.Valid {
		t := lastActiveAt.Time
		r.LastActiveAt = &t
	}

	return &r, nil
}
