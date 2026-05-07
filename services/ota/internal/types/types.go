package types

type DeviceVersionInfo struct {
	DeviceSn       string `json:"device_sn"`
	Model          string `json:"model"`
	DeviceType     string `json:"device_type,omitempty"`
	FirmwareVer    string `json:"firmware_version"`
}

type BatchCheckVersionReq struct {
	Devices []DeviceVersionInfo `json:"devices" validate:"required,dive"`
}

type DeviceCheckResult struct {
	DeviceSn      string `json:"device_sn"`
	Model         string `json:"model"`
	CurrentVer    string `json:"current_version"`
	NeedUpgrade   bool   `json:"need_upgrade"`
	LatestVer     string `json:"latest_version,omitempty"`
	UpgradeType   string `json:"upgrade_type,omitempty"`
	Changelog     string `json:"changelog,omitempty"`
	DownloadURL   string `json:"download_url,omitempty"`
	FileSize      int64  `json:"file_size,omitempty"`
	FileMD5       string `json:"file_md5,omitempty"`
	UpgradeTips   string `json:"upgrade_tips,omitempty"`
	ForceUpgrade  bool   `json:"force_upgrade,omitempty"`
	GrayScale     bool   `json:"gray_scale,omitempty"`
	GrayPercent   int    `json:"gray_percent,omitempty"`
	CheckTime     int64  `json:"check_time"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

type BatchCheckVersionResp struct {
	Total   int                 `json:"total"`
	Results []DeviceCheckResult `json:"results"`
	CheckTime int64             `json:"check_time"`
}
