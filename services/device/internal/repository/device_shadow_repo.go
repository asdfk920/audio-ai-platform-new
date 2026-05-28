package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
)

type DeviceShadowRepo struct {
	db *sql.DB
}

func NewDeviceShadowRepo(db *sql.DB) *DeviceShadowRepo {
	return &DeviceShadowRepo{db: db}
}

// ExistsByDeviceID 是否已有影子持久化行
func (r *DeviceShadowRepo) ExistsByDeviceID(ctx context.Context, deviceID int64) (bool, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM public.device_shadow WHERE device_id = $1`, deviceID).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("查询设备影子失败: %w", err)
	}
	return n > 0, nil
}

// InsertInitialIfAbsent 首次注册插入影子行（device_id 冲突则跳过）
func (r *DeviceShadowRepo) InsertInitialIfAbsent(ctx context.Context, deviceID int64, sn string, reported, desired, metadata json.RawMessage) (bool, error) {
	if len(reported) == 0 {
		reported = json.RawMessage(`{}`)
	}
	if len(desired) == 0 {
		desired = json.RawMessage(`{}`)
	}
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO public.device_shadow (device_id, sn, reported, desired, metadata, version, created_at, updated_at)
		VALUES ($1, $2, $3::jsonb, $4::jsonb, $5::jsonb, 0, NOW(), NOW())
		ON CONFLICT (device_id) DO NOTHING
	`, deviceID, sn, string(reported), string(desired), string(metadata))
	if err != nil {
		return false, fmt.Errorf("插入设备影子失败: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (r *DeviceShadowRepo) FindBySn(ctx context.Context, sn string) (*model.DeviceShadow, error) {
	query := `
		SELECT 
			id, sn, online_status, firmware_version, battery_level,
			last_report_time, last_command_time, reported, desired,
			metadata, version, disconnect_type, disconnect_at,
			created_at, updated_at
		FROM device_shadow 
		WHERE sn = $1
	`

	var shadow model.DeviceShadow
	var firmwareVersion sql.NullString
	var batteryLevel sql.NullInt64
	var lastReportTime sql.NullTime
	var lastCommandTime sql.NullTime
	var metadata json.RawMessage
	var version sql.NullInt64
	var disconnectType sql.NullString
	var disconnectAt sql.NullTime
	var updatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, sn).Scan(
		&shadow.ID, &shadow.Sn, &shadow.OnlineStatus, &firmwareVersion,
		&batteryLevel, &lastReportTime, &lastCommandTime,
		&shadow.Reported, &shadow.Desired, &metadata,
		&version, &disconnectType, &disconnectAt,
		&shadow.CreatedAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询设备影子失败: %v", err)
	}

	if firmwareVersion.Valid {
		shadow.FirmwareVersion = firmwareVersion.String
	}
	if batteryLevel.Valid {
		val := int(batteryLevel.Int64)
		shadow.BatteryLevel = &val
	}
	if lastReportTime.Valid {
		shadow.LastReportTime = &lastReportTime.Time
	}
	if lastCommandTime.Valid {
		shadow.LastCommandTime = &lastCommandTime.Time
	}
	if metadata != nil {
		shadow.Metadata = metadata
	}
	if version.Valid {
		shadow.Version = version.Int64
	}
	if disconnectType.Valid {
		shadow.DisconnectType = disconnectType.String
	}
	if disconnectAt.Valid {
		shadow.DisconnectAt = &disconnectAt.Time
	}
	if updatedAt.Valid {
		shadow.UpdatedAt = updatedAt.Time
	}

	return &shadow, nil
}

func (r *DeviceShadowRepo) CreateOrUpdate(ctx context.Context, shadow *model.DeviceShadow) error {
	query := `
		INSERT INTO device_shadow (
			sn, online_status, firmware_version, battery_level,
			reported, desired, metadata, version, disconnect_type, disconnect_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		ON CONFLICT (sn) DO UPDATE SET
			online_status = EXCLUDED.online_status,
			firmware_version = EXCLUDED.firmware_version,
			battery_level = EXCLUDED.battery_level,
			reported = EXCLUDED.reported,
			desired = EXCLUDED.desired,
			metadata = EXCLUDED.metadata,
			version = device_shadow.version + 1,
			disconnect_type = EXCLUDED.disconnect_type,
			disconnect_at = EXCLUDED.disconnect_at,
			updated_at = NOW()
		RETURNING id, version, updated_at
	`

	return r.db.QueryRowContext(ctx, query,
		shadow.Sn, shadow.OnlineStatus, shadow.FirmwareVersion, shadow.BatteryLevel,
		shadow.Reported, shadow.Desired, shadow.Metadata, 0,
		shadow.DisconnectType, shadow.DisconnectAt,
	).Scan(&shadow.ID, &shadow.Version, &shadow.UpdatedAt)
}

func (r *DeviceShadowRepo) UpdateOnlineStatus(ctx context.Context, sn string, onlineStatus int16, disconnectType string) (int64, error) {
	query := `
		UPDATE device_shadow 
		SET online_status = $1, 
		    disconnect_type = $2,
		    disconnect_at = CASE WHEN $1 = 0 THEN NOW() ELSE NULL END,
		    version = version + 1,
		    updated_at = NOW()
		WHERE sn = $3
		RETURNING version
	`

	var newVersion int64
	err := r.db.QueryRowContext(ctx, query, onlineStatus, disconnectType, sn).Scan(&newVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("设备影子不存在: %s", sn)
		}
		return 0, fmt.Errorf("更新影子在线状态失败: %v", err)
	}

	return newVersion, nil
}

func (r *DeviceShadowRepo) UpdateReported(ctx context.Context, sn string, reported json.RawMessage) (*model.DeviceShadow, error) {
	now := time.Now()
	query := `
		UPDATE device_shadow 
		SET reported = $1,
		    last_report_time = $2,
		    version = version + 1,
		    updated_at = NOW()
		WHERE sn = $3
		RETURNING 
			id, sn, online_status, firmware_version, battery_level,
			last_report_time, last_command_time, reported, desired,
			metadata, version, disconnect_type, disconnect_at,
			created_at, updated_at
	`

	var shadow model.DeviceShadow
	var firmwareVersion sql.NullString
	var batteryLevel sql.NullInt64
	var lastReportTime sql.NullTime
	var lastCommandTime sql.NullTime
	var metadata json.RawMessage
	var version sql.NullInt64
	var disconnectType sql.NullString
	var disconnectAt sql.NullTime
	var updatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, reported, now, sn).Scan(
		&shadow.ID, &shadow.Sn, &shadow.OnlineStatus, &firmwareVersion,
		&batteryLevel, &lastReportTime, &lastCommandTime,
		&shadow.Reported, &shadow.Desired, &metadata,
		&version, &disconnectType, &disconnectAt,
		&shadow.CreatedAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("设备影子不存在: %s", sn)
		}
		return nil, fmt.Errorf("更新影子reported属性失败: %v", err)
	}

	if firmwareVersion.Valid {
		shadow.FirmwareVersion = firmwareVersion.String
	}
	if batteryLevel.Valid {
		val := int(batteryLevel.Int64)
		shadow.BatteryLevel = &val
	}
	if lastReportTime.Valid {
		shadow.LastReportTime = &lastReportTime.Time
	}
	if lastCommandTime.Valid {
		shadow.LastCommandTime = &lastCommandTime.Time
	}
	if metadata != nil {
		shadow.Metadata = metadata
	}
	if version.Valid {
		shadow.Version = version.Int64
	}
	if disconnectType.Valid {
		shadow.DisconnectType = disconnectType.String
	}
	if disconnectAt.Valid {
		shadow.DisconnectAt = &disconnectAt.Time
	}
	if updatedAt.Valid {
		shadow.UpdatedAt = updatedAt.Time
	}

	return &shadow, nil
}

func (r *DeviceShadowRepo) UpdateDesired(ctx context.Context, sn string, desired json.RawMessage) (*model.DeviceShadow, error) {
	now := time.Now()
	query := `
		UPDATE device_shadow 
		SET desired = $1,
		    last_command_time = $2,
		    version = version + 1,
		    updated_at = NOW()
		WHERE sn = $3
		RETURNING 
			id, sn, online_status, firmware_version, battery_level,
			last_report_time, last_command_time, reported, desired,
			metadata, version, disconnect_type, disconnect_at,
			created_at, updated_at
	`

	var shadow model.DeviceShadow
	var firmwareVersion sql.NullString
	var batteryLevel sql.NullInt64
	var lastReportTime sql.NullTime
	var lastCommandTime sql.NullTime
	var metadata json.RawMessage
	var version sql.NullInt64
	var disconnectType sql.NullString
	var disconnectAt sql.NullTime
	var updatedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, desired, now, sn).Scan(
		&shadow.ID, &shadow.Sn, &shadow.OnlineStatus, &firmwareVersion,
		&batteryLevel, &lastReportTime, &lastCommandTime,
		&shadow.Reported, &shadow.Desired, &metadata,
		&version, &disconnectType, &disconnectAt,
		&shadow.CreatedAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("设备影子不存在: %s", sn)
		}
		return nil, fmt.Errorf("更新影子desired属性失败: %v", err)
	}

	if firmwareVersion.Valid {
		shadow.FirmwareVersion = firmwareVersion.String
	}
	if batteryLevel.Valid {
		val := int(batteryLevel.Int64)
		shadow.BatteryLevel = &val
	}
	if lastReportTime.Valid {
		shadow.LastReportTime = &lastReportTime.Time
	}
	if lastCommandTime.Valid {
		shadow.LastCommandTime = &lastCommandTime.Time
	}
	if metadata != nil {
		shadow.Metadata = metadata
	}
	if version.Valid {
		shadow.Version = version.Int64
	}
	if disconnectType.Valid {
		shadow.DisconnectType = disconnectType.String
	}
	if disconnectAt.Valid {
		shadow.DisconnectAt = &disconnectAt.Time
	}
	if updatedAt.Valid {
		shadow.UpdatedAt = updatedAt.Time
	}

	return &shadow, nil
}
