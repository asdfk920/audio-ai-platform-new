package models

import "time"

// DeviceImportJob 后台批量导入设备任务
type DeviceImportJob struct {
	ID                 int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Status             string     `gorm:"column:status;size:16" json:"status"`
	Total              int        `gorm:"column:total" json:"total"`
	Processed          int        `gorm:"column:processed" json:"processed"`
	SuccessCount       int        `gorm:"column:success_count" json:"success_count"`
	FailCount          int        `gorm:"column:fail_count" json:"fail_count"`
	ErrorMessage       string     `gorm:"column:error_message;type:text" json:"error_message"`
	FailureDetailJSON  string     `gorm:"column:failure_detail_json;type:jsonb" json:"-"`
	ResultFilePath     string     `gorm:"column:result_file_path" json:"-"`
	TempSourcePath     string     `gorm:"column:temp_source_path" json:"-"`
	CreatedBy          int64      `gorm:"column:created_by" json:"created_by"`
	CreatedAt          time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at" json:"updated_at"`
	FinishedAt         *time.Time `gorm:"column:finished_at" json:"finished_at"`
}

func (DeviceImportJob) TableName() string {
	return "device_import_job"
}
