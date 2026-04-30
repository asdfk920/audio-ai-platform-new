package repo

import (
	"context"
	"database/sql"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
)

// DeviceAuthRow MQTT 鉴权用设备行（含 bcrypt 的 device_secret）。
type DeviceAuthRow struct {
	ID              int64
	SN              string
	DeviceSecret    string
	Status          int16
	ProductKey      string
	Mac             string
	FirmwareVersion string
	IP              string
}

// GetDeviceForMQTTAuth 按规范化 SN 查设备（不校验用户绑定，供 MQTT 上报鉴权）。
func GetDeviceForMQTTAuth(ctx context.Context, db *sql.DB, snNorm string) (*DeviceAuthRow, error) {
	snNorm = strings.ToUpper(strings.TrimSpace(snNorm))
	if snNorm == "" || db == nil {
		return nil, sql.ErrNoRows
	}
	var r DeviceAuthRow
	err := db.QueryRowContext(ctx, `
SELECT id, sn, device_secret, status, product_key, mac, COALESCE(firmware_version,''), COALESCE(ip,'')
FROM device
WHERE deleted_at IS NULL AND UPPER(TRIM(sn)) = $1
LIMIT 1`, snNorm).Scan(
		&r.ID, &r.SN, &r.DeviceSecret, &r.Status, &r.ProductKey, &r.Mac, &r.FirmwareVersion, &r.IP,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ErrIfNotQueryable 与 HTTP 影子上报一致：仅 status=1 可上报。
func ErrIfNotQueryable(status int16) error {
	switch status {
	case 1:
		return nil
	case 2:
		return errorx.NewDefaultError(errorx.CodeDeviceDisabled)
	case 3:
		return errorx.NewDefaultError(errorx.CodeDeviceInactive)
	case 4:
		return errorx.NewDefaultError(errorx.CodeDeviceScrapped)
	default:
		return errorx.NewCodeError(errorx.CodeInvalidParam, "设备状态异常")
	}
}
