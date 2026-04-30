package models

import (
	"time"

	"gorm.io/gorm"
)

// Device 设备表
type Device struct {
	Id              int            `gorm:"primaryKey;autoIncrement" json:"id"`
	Sn              string         `gorm:"column:sn;size:64;not null;uniqueIndex" json:"sn"`
	ProductKey      string         `gorm:"column:product_key;size:64" json:"productKey"`
	DeviceSecret    string         `gorm:"column:device_secret;size:128" json:"-"`
	Model           string         `gorm:"column:model;size:64" json:"model"`
	Mac             string         `gorm:"column:mac;size:64" json:"mac"`
	Ip              string         `gorm:"column:ip;size:64" json:"ip"`
	FirmwareVersion string         `gorm:"column:firmware_version;size:64" json:"firmwareVersion"`
	HardwareVersion string         `gorm:"column:hardware_version;size:64" json:"hardwareVersion"`
	OnlineStatus    int16          `gorm:"column:online_status;default:0" json:"onlineStatus"`
	Status          int16          `gorm:"column:status;default:1" json:"status"`
	AdminDisplayName string        `gorm:"column:admin_display_name;size:128;default:''" json:"adminDisplayName"`
	AdminRemark     string         `gorm:"column:admin_remark;type:text" json:"adminRemark"`
	AdminLocation   string         `gorm:"column:admin_location;size:256;default:''" json:"adminLocation"`
	AdminGroupID    string         `gorm:"column:admin_group_id;size:64;default:''" json:"adminGroupId"`
	AdminTags       string         `gorm:"column:admin_tags;type:jsonb" json:"-"`
	AdminConfig     string         `gorm:"column:admin_config;type:jsonb" json:"-"`
	LastActiveAt    *time.Time     `gorm:"column:last_active_at" json:"lastActiveAt"`
	CreateBy        int            `gorm:"column:create_by" json:"createBy"`
	UpdateBy        int            `gorm:"column:update_by" json:"updateBy"`
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Device) TableName() string {
	return "device"
}

// UserDeviceBind 用户设备绑定表
type UserDeviceBind struct {
	Id          int            `gorm:"primaryKey;autoIncrement" json:"id"`
	UserId      int            `gorm:"column:user_id;type:bigint;not null;index:idx_user_device" json:"userId"`
	DeviceId    int            `gorm:"column:device_id;type:bigint;not null;index:idx_user_device" json:"deviceId"`
	DeviceSn    string         `gorm:"column:device_sn;type:varchar(64);not null" json:"deviceSn"`
	DeviceName  string         `gorm:"column:device_name;type:varchar(64);default:''" json:"deviceName"`
	DeviceModel string         `gorm:"column:device_model;type:varchar(64);default:''" json:"deviceModel"`
	Status      int16          `gorm:"column:status;type:smallint;not null;default:1" json:"status"`
	BoundAt     *time.Time     `gorm:"column:bound_at;type:timestamp" json:"boundAt"`
	UnboundAt   *time.Time     `gorm:"column:unbound_at;type:timestamp" json:"unboundAt"`
	CreateBy    int            `gorm:"column:create_by;type:bigint" json:"createBy"`
	UpdateBy    int            `gorm:"column:update_by;type:bigint" json:"updateBy"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;type:timestamp;index" json:"-"`
}

func (UserDeviceBind) TableName() string {
	return "user_device_bind"
}

// DeviceConfig 设备配置表
type DeviceConfig struct {
	Id          int            `gorm:"primaryKey;autoIncrement" json:"id"`
	DeviceId    int64          `gorm:"column:device_id;type:bigint;not null;index:idx_device" json:"device_id"`
	SN          string         `gorm:"column:sn;size:64;not null" json:"sn"`
	ConfigKey   string         `gorm:"column:config_key;size:64;not null" json:"config_key"`
	ConfigValue string         `gorm:"column:config_value;type:text" json:"config_value"`
	ConfigType  string         `gorm:"column:config_type;size:32;default:text" json:"config_type"` // text/json/number/boolean
	Status      int16          `gorm:"column:status;default:1" json:"status"`                      // 0-禁用 1-启用
	ApplyStatus int16          `gorm:"column:apply_status;default:0" json:"apply_status"`          // 0-待下发 1-已下发 2-已生效 3-生效失败
	ApplyTime   *time.Time     `gorm:"column:apply_time" json:"apply_time"`
	CreateBy    int            `gorm:"column:create_by;type:bigint" json:"create_by"`
	UpdateBy    int            `gorm:"column:update_by;type:bigint" json:"update_by"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (DeviceConfig) TableName() string {
	return "device_config"
}

// DeviceLog 设备日志表
type DeviceLog struct {
	Id           int            `gorm:"primaryKey;autoIncrement" json:"id"`
	BatchId      *int64         `gorm:"column:batch_id;type:bigint" json:"batch_id"`
	DeviceId     int64          `gorm:"column:device_id;type:bigint;not null;index:idx_device" json:"device_id"`
	SN           string         `gorm:"column:sn;size:64;not null;index:idx_sn" json:"sn"`
	LogType      string         `gorm:"column:log_type;size:32;not null;index:idx_type" json:"log_type"`        // system/error/operation/status
	LogLevel     string         `gorm:"column:log_level;size:16;default:info;index:idx_level" json:"log_level"` // debug/info/warn/error/fatal
	Module       string         `gorm:"column:module;size:64" json:"module"`
	Content      string         `gorm:"column:content;type:text" json:"content"`
	ErrorCode    *int           `gorm:"column:error_code" json:"error_code"`
	Extra        string         `gorm:"column:extra;type:json" json:"extra"` // 额外信息（JSON 格式）
	ReportTime   time.Time      `gorm:"column:report_time;not null;index:idx_report_time" json:"report_time"`
	ReportSource string         `gorm:"column:report_source;size:32;default:device" json:"report_source"` // device/cloud
	IpAddress    string         `gorm:"column:ip_address;size:64" json:"ip_address"`
	Processed    int16          `gorm:"column:processed;default:0" json:"processed"`   // 0-未处理 1-已处理 2-已忽略
	AlertSent    int16          `gorm:"column:alert_sent;default:0" json:"alert_sent"` // 0-未发送告警 1-已发送
	CreateBy     int            `gorm:"column:create_by;type:bigint" json:"create_by"`
	UpdateBy     int            `gorm:"column:update_by;type:bigint" json:"update_by"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (DeviceLog) TableName() string {
	return "device_log"
}

// DeviceDiagnosis 设备诊断记录表
type DeviceDiagnosis struct {
	Id             int            `gorm:"primaryKey;autoIncrement" json:"id"`
	DeviceId       int64          `gorm:"column:device_id;type:bigint;not null;index:idx_device" json:"device_id"`
	SN             string         `gorm:"column:sn;size:64;not null;index:idx_sn" json:"sn"`
	InstructionId  *int64         `gorm:"column:instruction_id;type:bigint" json:"instruction_id"`
	DiagType       string         `gorm:"column:diag_type;size:32;default:full" json:"diag_type"` // full/network/hardware/storage/audio/firmware/connection
	Status         int16          `gorm:"column:status;default:0" json:"status"`                  // 0-诊断中 1-已完成 2-诊断失败
	Params         string         `gorm:"column:params;type:json" json:"params"`
	ReportTime     *time.Time     `gorm:"column:report_time" json:"report_time"`
	Result         string         `gorm:"column:result;type:json" json:"result"`   // 诊断结果（JSON 格式）
	Summary        string         `gorm:"column:summary;type:text" json:"summary"` // 诊断摘要
	FailureReason  string         `gorm:"column:failure_reason;type:text" json:"failure_reason"`
	TotalItems     int            `gorm:"column:total_items" json:"total_items"`       // 总检查项数
	NormalItems    int            `gorm:"column:normal_items" json:"normal_items"`     // 正常项数
	AbnormalItems  int            `gorm:"column:abnormal_items" json:"abnormal_items"` // 异常项数
	HealthScore    int16          `gorm:"column:health_score" json:"health_score"`     // 健康评分（0-100）
	TimeoutSeconds int            `gorm:"column:timeout_seconds" json:"timeout_seconds"`
	Operator       string         `gorm:"column:operator;size:64" json:"operator"` // 操作人
	IpAddress      string         `gorm:"column:ip_address;size:64" json:"ip_address"`
	CompletedAt    *time.Time     `gorm:"column:completed_at" json:"completed_at"`
	CreateBy       int            `gorm:"column:create_by;type:bigint" json:"create_by"`
	UpdateBy       int            `gorm:"column:update_by;type:bigint" json:"update_by"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (DeviceDiagnosis) TableName() string {
	return "device_diagnosis"
}

// DeviceGroup 设备分组表
type DeviceGroup struct {
	Id          int            `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"column:name;size:64;not null;index:idx_name" json:"name"` // 分组名称
	Description string         `gorm:"column:description;size:256" json:"description"`          // 分组描述
	ParentId    *int           `gorm:"column:parent_id;index:idx_parent" json:"parent_id"`      // 父分组 ID（支持多级分组）
	Level       int16          `gorm:"column:level;default:1" json:"level"`                     // 分组层级（1-一级分组，2-二级分组...）
	Sort        int            `gorm:"column:sort;default:0" json:"sort"`                       // 排序权重
	DeviceCount int            `gorm:"column:device_count;default:0" json:"device_count"`       // 设备数量
	Status      int16          `gorm:"column:status;default:1" json:"status"`                   // 状态：0-禁用 1-启用
	CreateBy    int            `gorm:"column:create_by;type:bigint" json:"create_by"`
	UpdateBy    int            `gorm:"column:update_by;type:bigint" json:"update_by"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (DeviceGroup) TableName() string {
	return "device_group"
}

// DeviceGroupRelation 设备分组关联表
type DeviceGroupRelation struct {
	Id        int            `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupId   int            `gorm:"column:group_id;type:int;not null;index:idx_group_device" json:"group_id"`
	DeviceId  int64          `gorm:"column:device_id;type:bigint;not null;index:idx_group_device" json:"device_id"`
	DeviceSn  string         `gorm:"column:device_sn;size:64;not null" json:"device_sn"`
	CreateBy  int            `gorm:"column:create_by;type:bigint" json:"create_by"`
	UpdateBy  int            `gorm:"column:update_by;type:bigint" json:"update_by"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (DeviceGroupRelation) TableName() string {
	return "device_group_relation"
}

// DeviceScene 设备场景联动规则表
type DeviceScene struct {
	Id          int            `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"column:name;size:64;not null;index:idx_name" json:"name"`  // 场景名称
	Description string         `gorm:"column:description;size:256" json:"description"`           // 场景描述
	TriggerType string         `gorm:"column:trigger_type;size:32;not null" json:"trigger_type"` // 触发类型：device_status/timer/manual
	TriggerCond string         `gorm:"column:trigger_cond;type:json" json:"trigger_cond"`        // 触发条件（JSON 格式）
	ActionType  string         `gorm:"column:action_type;size:32;not null" json:"action_type"`   // 执行动作类型：device_control/mqtt_command/scene_trigger
	ActionExec  string         `gorm:"column:action_exec;type:json" json:"action_exec"`          // 执行动作（JSON 格式）
	DeviceIds   string         `gorm:"column:device_ids;type:text" json:"device_ids"`            // 关联设备 ID 列表（逗号分隔）
	Status      int16          `gorm:"column:status;default:1" json:"status"`                    // 状态：0-禁用 1-启用
	ExecCount   int            `gorm:"column:exec_count;default:0" json:"exec_count"`            // 执行次数
	LastExecAt  *time.Time     `gorm:"column:last_exec_at" json:"last_exec_at"`                  // 最后执行时间
	CreateBy    int            `gorm:"column:create_by;type:bigint" json:"create_by"`
	UpdateBy    int            `gorm:"column:update_by;type:bigint" json:"update_by"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (DeviceScene) TableName() string {
	return "device_scene"
}

// DeviceSceneExecLog 设备场景执行日志表
type DeviceSceneExecLog struct {
	Id          int            `gorm:"primaryKey;autoIncrement" json:"id"`
	SceneId     int            `gorm:"column:scene_id;type:int;not null;index:idx_scene" json:"scene_id"`
	SceneName   string         `gorm:"column:scene_name;size:64" json:"scene_name"`
	TriggerSrc  string         `gorm:"column:trigger_src;size:32" json:"trigger_src"`     // 触发来源：device/timer/manual/api
	TriggerData string         `gorm:"column:trigger_data;type:json" json:"trigger_data"` // 触发数据（JSON 格式）
	ExecStatus  int16          `gorm:"column:exec_status;default:0" json:"exec_status"`   // 执行状态：0-执行中 1-执行成功 2-执行失败
	ExecResult  string         `gorm:"column:exec_result;type:text" json:"exec_result"`   // 执行结果
	Devices     string         `gorm:"column:devices;type:text" json:"devices"`           // 涉及设备列表（逗号分隔）
	ErrMsg      string         `gorm:"column:err_msg;type:text" json:"err_msg"`           // 错误信息
	Duration    int            `gorm:"column:duration;default:0" json:"duration"`         // 执行耗时（毫秒）
	CreateBy    int            `gorm:"column:create_by;type:bigint" json:"create_by"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (DeviceSceneExecLog) TableName() string {
	return "device_scene_exec_log"
}

// DeviceShare 设备共享记录表
type DeviceShare struct {
	Id           int            `gorm:"primaryKey;autoIncrement" json:"id"`
	DeviceId     int            `gorm:"column:device_id;type:bigint;not null;index:idx_device" json:"deviceId"`               // 设备 ID
	DeviceSn     string         `gorm:"column:device_sn;size:64;not null" json:"deviceSn"`                                    // 设备 SN
	DeviceName   string         `gorm:"column:device_name;size:64" json:"deviceName"`                                         // 设备名称
	OwnerId      int            `gorm:"column:owner_id;type:bigint;not null;index:idx_owner" json:"ownerId"`                  // 设备主人 ID
	OwnerName    string         `gorm:"column:owner_name;size:64" json:"ownerName"`                                           // 设备主人名称
	SharedUserId int            `gorm:"column:shared_user_id;type:bigint;not null;index:idx_shared_user" json:"sharedUserId"` // 被共享人 ID
	SharedUser   string         `gorm:"column:shared_user;size:64;not null;index:idx_shared" json:"sharedUser"`               // 被共享人账号
	ShareType    int16          `gorm:"column:share_type;default:1" json:"shareType"`                                         // 共享类型：1-临时共享 2-永久共享
	Permissions  string         `gorm:"column:permissions;type:json" json:"permissions"`                                      // 共享权限（JSON 格式）
	Status       int16          `gorm:"column:status;default:0" json:"status"`                                                // 状态：0-待确认 1-已生效 2-已取消 3-已过期
	ExpireTime   *time.Time     `gorm:"column:expire_time" json:"expireTime"`                                                 // 过期时间
	ConfirmTime  *time.Time     `gorm:"column:confirm_time" json:"confirmTime"`                                               // 确认时间
	Remark       string         `gorm:"column:remark;size:256" json:"remark"`                                                 // 备注
	CreateBy     int            `gorm:"column:create_by;type:bigint" json:"create_by"`
	UpdateBy     int            `gorm:"column:update_by;type:bigint" json:"update_by"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (DeviceShare) TableName() string {
	return "device_share"
}

// DeviceShareLog 设备共享操作日志表
type DeviceShareLog struct {
	Id         int            `gorm:"primaryKey;autoIncrement" json:"id"`
	ShareId    int            `gorm:"column:share_id;type:bigint;not null;index:idx_share" json:"shareId"`
	DeviceId   int            `gorm:"column:device_id;type:bigint;not null" json:"deviceId"`
	DeviceName string         `gorm:"column:device_name;size:64" json:"deviceName"`
	OpType     string         `gorm:"column:op_type;size:32;not null" json:"opType"` // 操作类型：create/confirm/cancel/expire
	OpContent  string         `gorm:"column:op_content;type:text" json:"opContent"`  // 操作内容
	Operator   string         `gorm:"column:operator;size:64" json:"operator"`       // 操作人
	CreateBy   int            `gorm:"column:create_by;type:bigint" json:"create_by"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (DeviceShareLog) TableName() string {
	return "device_share_log"
}

// DeviceTask 设备定时任务表
type DeviceTask struct {
	Id          int            `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskName    string         `gorm:"column:task_name;size:64;not null" json:"taskName"`             // 任务名称
	TaskType    string         `gorm:"column:task_type;size:32;not null" json:"taskType"`             // 任务类型：single/cron
	DeviceIds   string         `gorm:"column:device_ids;type:text;not null" json:"device_ids"`        // 目标设备 ID 列表（逗号分隔）
	DeviceNames string         `gorm:"column:device_names;type:text" json:"device_names"`             // 目标设备名称列表（逗号分隔）
	ActionType  string         `gorm:"column:action_type;size:32;not null" json:"action_type"`        // 执行动作类型：device_control/mqtt_command
	ActionExec  string         `gorm:"column:action_exec;type:json" json:"action_exec"`               // 执行动作（JSON 格式）
	ExecTime    *time.Time     `gorm:"column:exec_time" json:"exec_time"`                             // 执行时间（单次任务）
	CronExpr    string         `gorm:"column:cron_expr;size:64" json:"cron_expr"`                     // Cron 表达式（重复任务）
	Timezone    string         `gorm:"column:timezone;size:32;default:Asia/Shanghai" json:"timezone"` // 时区
	Status      int16          `gorm:"column:status;default:1" json:"status"`                         // 状态：0-禁用 1-启用
	LastExecAt  *time.Time     `gorm:"column:last_exec_at" json:"last_exec_at"`                       // 最后执行时间
	NextExecAt  *time.Time     `gorm:"column:next_exec_at" json:"next_exec_at"`                       // 下次执行时间
	ExecCount   int            `gorm:"column:exec_count;default:0" json:"exec_count"`                 // 执行次数
	Remark      string         `gorm:"column:remark;size:256" json:"remark"`                          // 备注
	CreateBy    int            `gorm:"column:create_by;type:bigint" json:"create_by"`
	UpdateBy    int            `gorm:"column:update_by;type:bigint" json:"update_by"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (DeviceTask) TableName() string {
	return "device_task"
}

// DeviceTaskExecLog 设备定时任务执行日志表
type DeviceTaskExecLog struct {
	Id         int            `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskId     int            `gorm:"column:task_id;type:bigint;not null;index:idx_task" json:"taskId"`
	TaskName   string         `gorm:"column:task_name;size:64" json:"taskName"`
	DeviceIds  string         `gorm:"column:device_ids;type:text" json:"device_ids"`   // 涉及设备 ID 列表
	ExecStatus int16          `gorm:"column:exec_status;default:0" json:"exec_status"` // 执行状态：0-执行中 1-执行成功 2-执行失败
	ExecResult string         `gorm:"column:exec_result;type:text" json:"exec_result"` // 执行结果
	ErrMsg     string         `gorm:"column:err_msg;type:text" json:"err_msg"`         // 错误信息
	Duration   int            `gorm:"column:duration;default:0" json:"duration"`       // 执行耗时（毫秒）
	CreateBy   int            `gorm:"column:create_by;type:bigint" json:"create_by"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (DeviceTaskExecLog) TableName() string {
	return "device_task_exec_log"
}
