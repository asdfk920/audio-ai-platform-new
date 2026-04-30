package repo

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// InsertDeviceStatusRow 写入状态快照行（异步落库）。
func InsertDeviceStatusRow(ctx context.Context, db *sql.DB, deviceID int64, sn, runState string, battery int32, firmwareVersion string, onlineStatus int16, lastActive time.Time, source string) error {
	if db == nil || deviceID <= 0 {
		return nil
	}
	sn = strings.ToUpper(strings.TrimSpace(sn))
	if sn == "" {
		return nil
	}
	runState = strings.TrimSpace(runState)
	if len(runState) > 32 {
		runState = runState[:32]
	}
	fw := strings.TrimSpace(firmwareVersion)
	if len(fw) > 64 {
		fw = fw[:64]
	}
	src := strings.TrimSpace(source)
	if src == "" {
		src = "mqtt"
	}
	if len(src) > 16 {
		src = src[:16]
	}
	_, err := db.ExecContext(ctx, `
INSERT INTO device_status (device_id, sn, run_state, battery, firmware_version, online_status, last_active_at, source)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		deviceID, sn, runState, battery, fw, onlineStatus, lastActive, src)
	return err
}

// UpdateDeviceOnlineMeta 同步 device 主表在线态与活跃时间（异步，不阻塞消费）。
func UpdateDeviceOnlineMeta(ctx context.Context, db *sql.DB, deviceID int64, onlineStatus int16, lastActive time.Time, firmwareVersion, ip string) error {
	if db == nil || deviceID <= 0 {
		return nil
	}
	fw := strings.TrimSpace(firmwareVersion)
	ip = strings.TrimSpace(ip)
	if len(ip) > 45 {
		ip = ip[:45]
	}
	if fw != "" && ip != "" {
		_, err := db.ExecContext(ctx, `
UPDATE device SET online_status = $1, last_active_at = $2,
  firmware_version = CASE WHEN $3 <> '' THEN $3 ELSE firmware_version END,
  ip = CASE WHEN $4 <> '' THEN $4 ELSE ip END
WHERE id = $5`, onlineStatus, lastActive, fw, ip, deviceID)
		return err
	}
	if fw != "" {
		_, err := db.ExecContext(ctx, `
UPDATE device SET online_status = $1, last_active_at = $2, firmware_version = $3 WHERE id = $4`,
			onlineStatus, lastActive, fw, deviceID)
		return err
	}
	if ip != "" {
		_, err := db.ExecContext(ctx, `
UPDATE device SET online_status = $1, last_active_at = $2, ip = $3 WHERE id = $4`,
			onlineStatus, lastActive, ip, deviceID)
		return err
	}
	_, err := db.ExecContext(ctx, `
UPDATE device SET online_status = $1, last_active_at = $2 WHERE id = $3`, onlineStatus, lastActive, deviceID)
	return err
}
