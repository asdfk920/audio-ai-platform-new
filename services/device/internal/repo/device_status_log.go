package repo

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// InsertDeviceStatusLog 写入设备定时状态上报日志。reportType：auto / manual / sync。
func InsertDeviceStatusLog(ctx context.Context, db *sql.DB, deviceID int64, sn string,
	batteryLevel int, storageUsed, storageTotal int64, speakerCount int,
	uwbX, uwbY, uwbZ *float64, acousticCalibrated int16, acousticOffset *float64,
	reportedAt time.Time, reportType string,
) error {
	if db == nil || deviceID <= 0 {
		return nil
	}
	sn = strings.ToUpper(strings.TrimSpace(sn))
	if sn == "" {
		return nil
	}
	if reportType == "" {
		reportType = "auto"
	}
	_, err := db.ExecContext(ctx, `
INSERT INTO public.device_status_logs (
  device_id, sn, battery_level, storage_used, storage_total, speaker_count,
  uwb_x, uwb_y, uwb_z, acoustic_calibrated, acoustic_offset, reported_at, report_type
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		deviceID, sn, batteryLevel, storageUsed, storageTotal, speakerCount,
		uwbX, uwbY, uwbZ, acousticCalibrated, acousticOffset, reportedAt, reportType,
	)
	return err
}
