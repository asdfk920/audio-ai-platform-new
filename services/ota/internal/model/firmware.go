package model

import "time"

type OtaFirmware struct {
	ID          int64      `db:"id"`
	Model       string     `db:"model"`
	Version     string     `db:"version"`
	DeviceType  string     `db:"device_type"`
	UpgradeType string     `db:"upgrade_type"`
	Changelog   string     `db:"changelog"`
	DownloadURL string     `db:"download_url"`
	FileSize    int64      `db:"file_size"`
	FileMD5     string     `db:"file_md5"`
	Published   int16      `db:"published"`
	GrayScale   int16      `db:"gray_scale"`
	GrayPercent int        `db:"gray_percent"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
}

type OtaUpgradeTask struct {
	ID           int64      `db:"id"`
	TaskID       string     `db:"task_id"`
	DeviceSn     string     `db:"device_sn"`
	DeviceModel  string     `db:"device_model"`
	DeviceType   string     `db:"device_type"`
	UserID       int64      `db:"user_id"`
	FromVersion  string     `db:"from_version"`
	ToVersion    string     `db:"to_version"`
	UpgradeType  string     `db:"upgrade_type"`
	Status       int16      `db:"status"`
	Progress     int        `db:"progress"`
	DownloadURL  string     `db:"download_url"`
	FileSize     int64      `db:"file_size"`
	FileMD5      string     `db:"file_md5"`
	ErrorMessage string     `db:"error_message"`
	StartedAt    *time.Time `db:"started_at"`
	CompletedAt  *time.Time `db:"completed_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

type OtaUpgradeHistory struct {
	ID           int64     `db:"id"`
	DeviceSn     string    `db:"device_sn"`
	DeviceModel  string    `db:"device_model"`
	UserID       int64     `db:"user_id"`
	FromVersion  string    `db:"from_version"`
	ToVersion    string    `db:"to_version"`
	UpgradeType  string    `db:"upgrade_type"`
	Status       int16     `db:"status"`
	Duration     int       `db:"duration"`
	ErrorMessage string    `db:"error_message"`
	UpgradeTime  time.Time `db:"upgrade_time"`
	CreatedAt    time.Time `db:"created_at"`
}

const (
	UpgradeTypeForce    = "force"
	UpgradeTypeOptional = "optional"
)

const (
	PublishedNo  = 0
	PublishedYes = 1
)

const (
	GrayScaleNo  = 0
	GrayScaleYes = 1
)

const (
	TaskStatusPending     int16 = 0
	TaskStatusDownloading int16 = 1
	TaskStatusUpgrading   int16 = 2
	TaskStatusSuccess     int16 = 3
	TaskStatusFailed      int16 = 4
	TaskStatusCancelled   int16 = 5
)

const (
	HistoryStatusSuccess int16 = 3
	HistoryStatusFailed  int16 = 4
)
