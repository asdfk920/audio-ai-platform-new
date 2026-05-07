package model

import (
	"database/sql"
	"fmt"
)

type OtaFirmwareRepo struct {
	db *sql.DB
}

func NewOtaFirmwareRepo(db *sql.DB) *OtaFirmwareRepo {
	return &OtaFirmwareRepo{db: db}
}

func (r *OtaFirmwareRepo) FindLatestVersion(model, deviceType string) (*OtaFirmware, error) {
	query := `SELECT id, model, version, device_type, upgrade_type, changelog, 
			  download_url, file_size, file_md5, published, gray_scale, gray_percent, 
			  created_at, updated_at, deleted_at 
			  FROM ota_firmware 
			  WHERE model = $1 AND device_type = $2 AND published = 1 AND deleted_at IS NULL 
			  ORDER BY created_at DESC LIMIT 1`

	var fw OtaFirmware
	err := r.db.QueryRow(query, model, deviceType).Scan(
		&fw.ID, &fw.Model, &fw.Version, &fw.DeviceType, &fw.UpgradeType,
		&fw.Changelog, &fw.DownloadURL, &fw.FileSize, &fw.FileMD5,
		&fw.Published, &fw.GrayScale, &fw.GrayPercent,
		&fw.CreatedAt, &fw.UpdatedAt, &fw.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query ota firmware version failed: %w, model=%s, deviceType=%s", err, model, deviceType)
	}

	return &fw, nil
}
