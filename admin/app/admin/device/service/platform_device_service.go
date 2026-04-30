package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-admin-team/go-admin-core/sdk/service"
	"gorm.io/gorm"

	"go-admin/app/admin/device/models"
	"go-admin/common/mqttadmin"
)

// user_device_bind.status：0=已解绑 1=绑定中（035）
const (
	deviceInstructionPending = int16(1)
	deviceInstructionSent    = int16(2)
)

var (
	ErrPlatformDeviceNotFound   = errors.New("设备不存在")
	ErrPlatformDeviceInvalid    = errors.New("参数无效")
	ErrPlatformDeviceDuplicate  = errors.New("设备 SN 已存在")
	ErrFirmwareVersionProtected = errors.New("版本号不可修改")
	ErrFirmwareTaskConflict     = errors.New("固件关联进行中的升级任务")
	ErrFirmwareUpdatePermission = errors.New("没有权限修改该固件")
	ErrOTATaskNotFound          = errors.New("OTA 任务不存在")
	ErrOTATaskAlreadyCompleted  = errors.New("任务已完成")
	ErrOTATaskAlreadyCancelled  = errors.New("任务已取消")
	ErrOTATaskCancelFailed      = errors.New("取消操作失败")
	ErrProductKeyNotActive      = errors.New("产品密钥不存在或未激活（需存在已发布且启用的固件）")
	ErrPlatformDeviceMacDuplicate = errors.New("MAC 已被其他设备使用")
	ErrPlatformDeviceSNFormat   = errors.New("序列号须为 16-32 位字母或数字")
	ErrPlatformDeviceMACFormat  = errors.New("MAC 地址格式不正确")
	ErrImportJobNotFound        = errors.New("导入任务不存在")
	ErrImportJobAccessDenied    = errors.New("无权访问该导入任务")
)

// PlatformDeviceService 平台设备管理
type PlatformDeviceService struct {
	service.Service
}

// PlatformDeviceListFilter 列表筛选
type PlatformDeviceListFilter struct {
	Sn           string
	SnExact      bool
	UserID       int64
	UserQuery    string
	Status       *int16
	OnlineStatus *int16
	ProductKey   string
	FirmwareVer  string
	BindStatus   *int16 // 0=未绑定 1=已绑定
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
	SortBy       string // id, created_at, last_active_at
	SortOrder    string // asc, desc
}

// PlatformDeviceListItem 列表行
type PlatformDeviceListItem struct {
	ID              int64      `json:"id"`
	Sn              string     `json:"sn"`
	Model           string     `json:"model"`
	ProductKey      string     `json:"product_key"`
	FirmwareVersion string     `json:"firmware_version"`
	HardwareVersion string     `json:"hardware_version"`
	Mac             string     `json:"mac"`
	Ip              string     `json:"ip"`
	OnlineStatus    int16      `json:"online_status"`
	DisplayOnline   int16      `json:"display_online"`
	Status          int16      `json:"status"`
	UserID          int64      `json:"user_id"`
	UserNickname    string     `json:"user_nickname"`
	UserMobile      string     `json:"user_mobile"`
	BindTime        *time.Time `json:"bind_time"`
	LastActiveAt    *time.Time `json:"last_active_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func maskDeviceSecret(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r := []rune(s)
	if len(r) <= 4 {
		return "****"
	}
	return string(r[:4]) + "****"
}

func displayOnlineFromLastActive(t *time.Time, dbOnline int16) int16 {
	if t != nil && time.Since(*t) <= 5*time.Minute {
		return 1
	}
	if dbOnline == 1 && t == nil {
		return 1
	}
	return 0
}

func maskUserMobile(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r := []rune(s)
	if len(r) <= 4 {
		return "****"
	}
	return "****" + string(r[len(r)-4:])
}

func listOrderSQL(f PlatformDeviceListFilter) string {
	col := "d.id"
	switch strings.ToLower(strings.TrimSpace(f.SortBy)) {
	case "created_at":
		col = "d.created_at"
	case "last_active_at":
		col = "d.last_active_at"
	case "bind_time":
		col = "udb.bound_at"
	case "id":
		col = "d.id"
	}
	ord := "DESC"
	if strings.EqualFold(strings.TrimSpace(f.SortOrder), "asc") {
		ord = "ASC"
	}
	return col + " " + ord
}

func (e *PlatformDeviceService) deviceListQuery(f PlatformDeviceListFilter) *gorm.DB {
	q := e.Orm.Table("device AS d").
		Joins("LEFT JOIN user_device_bind udb ON udb.device_id = d.id AND udb.status = 1").
		Joins("LEFT JOIN users u ON u.id = udb.user_id")
	if s := strings.TrimSpace(f.Sn); s != "" {
		if f.SnExact {
			q = q.Where("d.sn = ?", s)
		} else {
			q = q.Where("d.sn ILIKE ?", "%"+s+"%")
		}
	}
	if f.UserID > 0 {
		q = q.Where("udb.user_id = ?", f.UserID)
	}
	if kw := strings.TrimSpace(f.UserQuery); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(u.nickname ILIKE ? OR u.mobile ILIKE ? OR CAST(u.id AS TEXT) = ?)", like, like, kw)
	}
	if f.Status != nil {
		q = q.Where("d.status = ?", *f.Status)
	}
	if f.OnlineStatus != nil {
		q = q.Where("d.online_status = ?", *f.OnlineStatus)
	}
	if pk := strings.TrimSpace(f.ProductKey); pk != "" {
		q = q.Where("d.product_key = ?", pk)
	}
	if fw := strings.TrimSpace(f.FirmwareVer); fw != "" {
		q = q.Where("d.firmware_version ILIKE ?", "%"+fw+"%")
	}
	if f.BindStatus != nil {
		if *f.BindStatus == 1 {
			q = q.Where("udb.user_id IS NOT NULL")
		} else {
			q = q.Where("udb.user_id IS NULL")
		}
	}
	if f.CreatedFrom != nil {
		q = q.Where("d.created_at >= ?", *f.CreatedFrom)
	}
	if f.CreatedTo != nil {
		q = q.Where("d.created_at < ?", f.CreatedTo.Add(24*time.Hour))
	}
	return q
}

// ListDevices 分页列表
func (e *PlatformDeviceService) ListDevices(page, pageSize int, f PlatformDeviceListFilter) ([]PlatformDeviceListItem, int64, error) {
	if e.Orm == nil {
		return nil, 0, fmt.Errorf("orm nil")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var total int64
	sub := e.deviceListQuery(f).Select("d.id").Distinct()
	if err := e.Orm.Table("(?) AS _cnt", sub).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	type row struct {
		ID              int64      `gorm:"column:id"`
		Sn              string     `gorm:"column:sn"`
		Model           string     `gorm:"column:model"`
		ProductKey      string     `gorm:"column:product_key"`
		FirmwareVersion string     `gorm:"column:firmware_version"`
		HardwareVersion string     `gorm:"column:hardware_version"`
		Mac             string     `gorm:"column:mac"`
		Ip              string     `gorm:"column:ip"`
		OnlineStatus    int16      `gorm:"column:online_status"`
		Status          int16      `gorm:"column:status"`
		CreatedAt       time.Time  `gorm:"column:created_at"`
		UpdatedAt       time.Time  `gorm:"column:updated_at"`
		LastActiveAt    *time.Time `gorm:"column:last_active_at"`
		BindUserID      *int64     `gorm:"column:bind_user_id"`
		UserNickname    *string    `gorm:"column:user_nickname"`
		UserMobileRaw   *string    `gorm:"column:user_mobile_raw"`
		BindTime        *time.Time `gorm:"column:bind_time"`
	}
	var rows []row
	err := e.deviceListQuery(f).
		Select(`d.id, d.sn, d.model, d.product_key, d.firmware_version, d.hardware_version,
			d.mac, d.ip, d.online_status, d.status, d.created_at, d.updated_at, d.last_active_at,
			udb.user_id AS bind_user_id, u.nickname AS user_nickname, u.mobile AS user_mobile_raw, udb.bound_at AS bind_time`).
		Order(listOrderSQL(f)).
		Limit(pageSize).
		Offset(offset).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	out := make([]PlatformDeviceListItem, 0, len(rows))
	for _, r := range rows {
		uid := int64(0)
		if r.BindUserID != nil {
			uid = *r.BindUserID
		}
		nick := ""
		if r.UserNickname != nil {
			nick = *r.UserNickname
		}
		mob := ""
		if r.UserMobileRaw != nil {
			mob = maskUserMobile(*r.UserMobileRaw)
		}
		la := r.LastActiveAt
		do := displayOnlineFromLastActive(la, r.OnlineStatus)
		out = append(out, PlatformDeviceListItem{
			ID: r.ID, Sn: r.Sn, Model: r.Model, ProductKey: r.ProductKey,
			FirmwareVersion: r.FirmwareVersion, HardwareVersion: r.HardwareVersion,
			Mac: r.Mac, Ip: r.Ip, OnlineStatus: r.OnlineStatus, DisplayOnline: do,
			Status: r.Status, UserID: uid, UserNickname: nick, UserMobile: mob, BindTime: r.BindTime,
			LastActiveAt: la, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		})
	}
	return out, total, nil
}

// DeviceInstructionSummary 指令摘要
type DeviceInstructionSummary struct {
	ID        int64           `json:"id"`
	Cmd       string          `json:"cmd"`
	Status    int16           `json:"status"`
	Params    json.RawMessage `json:"params"`
	CreatedAt time.Time       `json:"created_at"`
}

// OtaTaskSummary OTA 任务摘要
type OtaTaskSummary struct {
	ID        int64     `json:"id"`
	FromVer   string    `json:"from_version"`
	ToVer     string    `json:"to_version"`
	Status    int16     `json:"status"`
	Progress  int       `json:"progress"`
	ErrorMsg  string    `json:"error_msg"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeviceEventSummary 事件摘要
type DeviceEventSummary struct {
	ID        int64     `json:"id"`
	EventType string    `json:"event_type"`
	Content   string    `json:"content"`
	Operator  string    `json:"operator"`
	CreatedAt time.Time `json:"created_at"`
}

// PlatformDeviceDetailOut 详情聚合
type PlatformDeviceDetailOut struct {
	Device       PlatformDeviceDetailCore   `json:"device"`
	Instructions []DeviceInstructionSummary `json:"instructions"`
	OtaTasks     []OtaTaskSummary           `json:"ota_tasks"`
	Events       []DeviceEventSummary       `json:"events"`
}

// PlatformDeviceDetailCore 设备核心信息
type PlatformDeviceDetailCore struct {
	ID              int64           `json:"id"`
	Sn              string          `json:"sn"`
	ProductKey      string          `json:"product_key"`
	DeviceSecret    string          `json:"device_secret_masked"`
	FirmwareVersion string          `json:"firmware_version"`
	HardwareVersion string          `json:"hardware_version"`
	Model           string          `json:"model"`
	Mac             string          `json:"mac"`
	Ip              string          `json:"ip"`
	OnlineStatus    int16           `json:"online_status"`
	DisplayOnline   int16           `json:"display_online"`
	Status          int16           `json:"status"`
	UserID          int64           `json:"user_id"`
	UserNickname    string          `json:"user_nickname"`
	BindTime        *time.Time      `json:"bind_time"`
	LastActiveAt    *time.Time      `json:"last_active_at"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeviceName      string          `json:"device_name"`
	Remark          string          `json:"remark"`
	Location        string          `json:"location"`
	GroupID         string          `json:"group_id"`
	Tags            json.RawMessage `json:"tags"`
	Config          json.RawMessage `json:"config"`
}

const (
	deviceDetailSelectFull = `d.id, d.sn, d.product_key, d.device_secret, d.firmware_version, d.hardware_version,
			d.model, d.mac, d.ip, d.online_status, d.status, d.created_at, d.updated_at, d.last_active_at,
			d.admin_display_name, d.admin_remark, d.admin_location, d.admin_group_id, d.admin_tags::text, d.admin_config::text,
			udb.user_id AS bind_user_id, u.nickname AS user_nickname, udb.bound_at AS bind_time`
	deviceDetailSelectLegacy079 = `d.id, d.sn, d.product_key, d.device_secret, d.firmware_version, d.hardware_version,
			d.model, d.mac, d.ip, d.online_status, d.status, d.created_at, d.updated_at, d.last_active_at,
			d.admin_display_name, d.admin_remark,
			udb.user_id AS bind_user_id, u.nickname AS user_nickname, udb.bound_at AS bind_time`
	deviceDetailSelectMinimal = `d.id, d.sn, d.product_key, d.device_secret, d.firmware_version, d.hardware_version,
			d.model, d.mac, d.ip, d.online_status, d.status, d.created_at, d.updated_at, d.last_active_at,
			udb.user_id AS bind_user_id, u.nickname AS user_nickname, udb.bound_at AS bind_time`
)

type deviceDetailRow struct {
	ID               int64      `gorm:"column:id"`
	Sn               string     `gorm:"column:sn"`
	ProductKey       string     `gorm:"column:product_key"`
	DeviceSecret     string     `gorm:"column:device_secret"`
	FirmwareVersion  string     `gorm:"column:firmware_version"`
	HardwareVersion  string     `gorm:"column:hardware_version"`
	Model            string     `gorm:"column:model"`
	Mac              string     `gorm:"column:mac"`
	Ip               string     `gorm:"column:ip"`
	OnlineStatus     int16      `gorm:"column:online_status"`
	Status           int16      `gorm:"column:status"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
	LastActiveAt     *time.Time `gorm:"column:last_active_at"`
	BindUserID       *int64     `gorm:"column:bind_user_id"`
	UserNickname     *string    `gorm:"column:user_nickname"`
	BindTime         *time.Time `gorm:"column:bind_time"`
	AdminDisplayName string     `gorm:"column:admin_display_name"`
	AdminRemark      string     `gorm:"column:admin_remark"`
	AdminLocation    string     `gorm:"column:admin_location"`
	AdminGroupID     string     `gorm:"column:admin_group_id"`
	AdminTags        string     `gorm:"column:admin_tags"`
	AdminConfig      string     `gorm:"column:admin_config"`
}

func (e *PlatformDeviceService) takeDeviceDetailByID(deviceID int64, selectSQL string, dr *deviceDetailRow) error {
	return e.Orm.Session(&gorm.Session{NewDB: true}).
		Table("device AS d").
		Select(selectSQL).
		Joins("LEFT JOIN user_device_bind udb ON udb.device_id = d.id AND udb.status = 1").
		Joins("LEFT JOIN users u ON u.id = udb.user_id").
		Where("d.id = ?", deviceID).
		Take(dr).Error
}

// resolveDevicePK 仅用 device 表解析主键：device_id 优先；若主键不存在且提供了 sn 则按 SN 回退（容错列表行 id 异常）。
func (e *PlatformDeviceService) resolveDevicePK(sn string, deviceID int64) (int64, error) {
	sn = strings.TrimSpace(sn)
	if deviceID > 0 {
		var row struct {
			ID int64 `gorm:"column:id"`
		}
		err := e.Orm.Session(&gorm.Session{NewDB: true}).Table("device").Select("id").Where("id = ?", deviceID).Take(&row).Error
		if err == nil {
			return row.ID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if sn == "" {
		return 0, ErrPlatformDeviceNotFound
	}
	snNorm := strings.ToUpper(sn)
	var row struct {
		ID int64 `gorm:"column:id"`
	}
	err := e.Orm.Session(&gorm.Session{NewDB: true}).Table("device").Select("id").Where("UPPER(TRIM(sn)) = ?", snNorm).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, ErrPlatformDeviceNotFound
	}
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}

// GetDeviceDetail 详情（含指令/OTA/事件摘要）。先解析 device.id 再带 JOIN 拉详情，与列表数据源一致。
func (e *PlatformDeviceService) GetDeviceDetail(sn string, deviceID int64) (*PlatformDeviceDetailOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	sn = strings.TrimSpace(sn)
	if deviceID <= 0 && sn == "" {
		return nil, ErrPlatformDeviceNotFound
	}
	pk, err := e.resolveDevicePK(sn, deviceID)
	if err != nil {
		return nil, err
	}

	var dr deviceDetailRow
	err = e.takeDeviceDetailByID(pk, deviceDetailSelectFull, &dr)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPlatformDeviceNotFound
	}
	if err != nil {
		dr = deviceDetailRow{}
		err = e.takeDeviceDetailByID(pk, deviceDetailSelectLegacy079, &dr)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
	}
	if err != nil {
		dr = deviceDetailRow{}
		err = e.takeDeviceDetailByID(pk, deviceDetailSelectMinimal, &dr)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
	}
	if err != nil {
		return nil, err
	}

	uid := int64(0)
	if dr.BindUserID != nil {
		uid = *dr.BindUserID
	}
	nick := ""
	if dr.UserNickname != nil {
		nick = *dr.UserNickname
	}

	tagsJSON := strings.TrimSpace(dr.AdminTags)
	if tagsJSON == "" {
		tagsJSON = "[]"
	}
	cfgJSON := strings.TrimSpace(dr.AdminConfig)
	if cfgJSON == "" {
		cfgJSON = "{}"
	}

	core := PlatformDeviceDetailCore{
		ID: dr.ID, Sn: dr.Sn, ProductKey: dr.ProductKey,
		DeviceSecret:    maskDeviceSecret(dr.DeviceSecret),
		FirmwareVersion: dr.FirmwareVersion, HardwareVersion: dr.HardwareVersion,
		Model: dr.Model, Mac: dr.Mac, Ip: dr.Ip, OnlineStatus: dr.OnlineStatus,
		DisplayOnline: displayOnlineFromLastActive(dr.LastActiveAt, dr.OnlineStatus),
		Status:        dr.Status, UserID: uid, UserNickname: nick, BindTime: dr.BindTime,
		LastActiveAt: dr.LastActiveAt, CreatedAt: dr.CreatedAt, UpdatedAt: dr.UpdatedAt,
		DeviceName:   dr.AdminDisplayName,
		Remark:       dr.AdminRemark,
		Location:     dr.AdminLocation,
		GroupID:      dr.AdminGroupID,
		Tags:         json.RawMessage(tagsJSON),
		Config:       json.RawMessage(cfgJSON),
	}

	var instr []DeviceInstructionSummary
	_ = e.Orm.Table("device_instruction").
		Select("id, cmd, status, params, created_at").
		Where("device_id = ?", dr.ID).
		Order("id DESC").
		Limit(50).
		Scan(&instr).Error

	var ota []OtaTaskSummary
	_ = e.Orm.Table("ota_upgrade_task").
		Select("id, from_version, to_version, status, progress, error_msg, created_at, updated_at").
		Where("device_id = ?", dr.ID).
		Order("id DESC").
		Limit(20).
		Scan(&ota).Error

	var ev []DeviceEventSummary
	_ = e.Orm.Table("device_event_log").
		Select("id, event_type, content, operator, created_at").
		Where("device_id = ?", dr.ID).
		Order("id DESC").
		Limit(50).
		Scan(&ev).Error

	return &PlatformDeviceDetailOut{
		Device:       core,
		Instructions: instr,
		OtaTasks:     ota,
		Events:       ev,
	}, nil
}

// DeviceDiagnosisType 诊断类型常量
const (
	DiagTypeFull       = "full"       // 全面诊断
	DiagTypeNetwork    = "network"    // 网络诊断
	DiagTypeHardware   = "hardware"   // 硬件诊断
	DiagTypeStorage    = "storage"    // 存储诊断
	DiagTypeAudio      = "audio"      // 音频诊断
	DiagTypeFirmware   = "firmware"   // 固件诊断
	DiagTypeConnection = "connection" // 连接诊断
)

// DeviceDiagnosisItem 诊断项
type DeviceDiagnosisItem struct {
	Item      string `json:"item"`       // 检查项名称
	Status    string `json:"status"`     // normal/abnormal/error
	Message   string `json:"message"`    // 检查结果描述
	ErrorCode *int   `json:"error_code"` // 错误码（可选）
	Detail    string `json:"detail"`     // 详细信息
}

// StartDiagnosisIn 开始诊断输入
type StartDiagnosisIn struct {
	DeviceID  int64  `json:"device_id"`
	Sn        string `json:"sn"`
	DiagType  string `json:"diag_type"` // full/network/hardware/storage/audio/firmware/connection
	Operator  string `json:"operator"`
	IpAddress string `json:"ip_address"`
}

// StartDiagnosisOut 开始诊断输出
type StartDiagnosisOut struct {
	DiagnosisId   int    `json:"diagnosis_id"` // 诊断记录 ID
	InstructionId int64  `json:"instruction_id"`
	DeviceID      int64  `json:"device_id"`
	Sn            string `json:"sn"`
	DiagType      string `json:"diag_type"`
	Status        string `json:"status"` // diagnosing（诊断中）
	Message       string `json:"message"`
}

// StartDiagnosis 开始设备诊断
// 1. 校验设备存在且在线
// 2. 校验用户诊断权限
// 3. 创建诊断记录
// 4. 构造诊断指令
// 5. 通过 MQTT 下发诊断指令
// 6. 返回诊断记录 ID
func (e *PlatformDeviceService) StartDiagnosis(in *StartDiagnosisIn) (*StartDiagnosisOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	if in.DeviceID <= 0 && strings.TrimSpace(in.Sn) == "" {
		return nil, ErrPlatformDeviceInvalid
	}

	// 1. 查询设备
	var dev struct {
		ID           int64      `gorm:"column:id"`
		Sn           string     `gorm:"column:sn"`
		Status       int16      `gorm:"column:status"`
		OnlineStatus int16      `gorm:"column:online_status"`
		LastActiveAt *time.Time `gorm:"column:last_active_at"`
	}
	q := e.Orm.Table("device").Select("id, sn, status, online_status, last_active_at")
	if in.DeviceID > 0 {
		q = q.Where("id = ?", in.DeviceID)
	} else {
		q = q.Where("sn = ?", strings.TrimSpace(in.Sn))
	}
	if err := q.Take(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}
	if dev.Status != 1 {
		return nil, fmt.Errorf("设备已禁用，无法诊断")
	}
	if dev.OnlineStatus != 1 {
		return nil, fmt.Errorf("设备不在线，无法执行诊断")
	}

	// 2. 校验诊断类型
	if in.DiagType == "" {
		in.DiagType = DiagTypeFull
	}
	validDiagTypes := map[string]bool{
		DiagTypeFull:       true,
		DiagTypeNetwork:    true,
		DiagTypeHardware:   true,
		DiagTypeStorage:    true,
		DiagTypeAudio:      true,
		DiagTypeFirmware:   true,
		DiagTypeConnection: true,
	}
	if !validDiagTypes[in.DiagType] {
		return nil, fmt.Errorf("无效的诊断类型：%s", in.DiagType)
	}

	now := time.Now()
	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "admin"
	}

	// 3. 创建诊断记录
	diagnosis := models.DeviceDiagnosis{
		DeviceId:       dev.ID,
		SN:             dev.Sn,
		DiagType:       in.DiagType,
		Status:         0, // 诊断中
		TimeoutSeconds: 300,
		ReportTime:     nil,
		Result:         "{}",
		Summary:        fmt.Sprintf("开始 %s 诊断", in.DiagType),
		TotalItems:     0,
		NormalItems:    0,
		AbnormalItems:  0,
		HealthScore:    0,
		Operator:       op,
		IpAddress:      in.IpAddress,
		CreateBy:       0,
		UpdateBy:       0,
	}

	if err := e.Orm.Create(&diagnosis).Error; err != nil {
		return nil, err
	}

	diagParams := map[string]interface{}{
		"diagnosis_id": diagnosis.Id,
		"diag_type":    in.DiagType,
		"timestamp":    now.Unix(),
		"source":       "admin",
	}
	diagReason := "remote_diagnosis:" + in.DiagType
	adminCmdOut, err := e.AdminRemoteCommand(&AdminRemoteCommandIn{
		DeviceID:            dev.ID,
		Sn:                  dev.Sn,
		Command:             "diagnosis",
		Reason:              diagReason,
		Operator:            op,
		ConfirmFactoryReset: false,
		Params:              diagParams,
	})
	if err != nil {
		return nil, err
	}
	var instructionID int64
	if strings.HasPrefix(adminCmdOut.CommandID, "cmd_") {
		if parsed, parseErr := strconv.ParseInt(strings.TrimPrefix(adminCmdOut.CommandID, "cmd_"), 10, 64); parseErr == nil {
			instructionID = parsed
		}
	}
	if instructionID > 0 {
		if err := e.Orm.Model(&models.DeviceDiagnosis{}).Where("id = ?", diagnosis.Id).Updates(map[string]interface{}{
			"instruction_id": instructionID,
			"params":         mustJSONMap(diagParams),
			"updated_at":     time.Now(),
		}).Error; err != nil {
			return nil, err
		}
	}

	return &StartDiagnosisOut{
		DiagnosisId:   diagnosis.Id,
		InstructionId: instructionID,
		DeviceID:      dev.ID,
		Sn:            dev.Sn,
		DiagType:      in.DiagType,
		Status:        "diagnosing",
		Message:       "诊断指令已入队，设备将执行自检",
	}, nil
}

// ReportDiagnosisIn 上报诊断结果输入
type ReportDiagnosisIn struct {
	DiagnosisId int                   `json:"diagnosis_id"`
	DeviceID    int64                 `json:"device_id"`
	Sn          string                `json:"sn"`
	Items       []DeviceDiagnosisItem `json:"items"`   // 诊断项列表
	Summary     string                `json:"summary"` // 诊断摘要
	ReportTime  time.Time             `json:"report_time"`
}

// ReportDiagnosisOut 上报诊断结果输出
type ReportDiagnosisOut struct {
	DiagnosisId int    `json:"diagnosis_id"`
	DeviceID    int64  `json:"device_id"`
	Sn          string `json:"sn"`
	Status      string `json:"status"` // completed（已完成）
	Message     string `json:"message"`
}

// ReportDiagnosis 设备上报诊断结果
// 1. 校验诊断记录存在
// 2. 解析诊断结果
// 3. 统计正常/异常项
// 4. 计算健康评分
// 5. 更新诊断记录
// 6. 返回成功响应
func (e *PlatformDeviceService) ReportDiagnosis(in *ReportDiagnosisIn) (*ReportDiagnosisOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	if in.DiagnosisId <= 0 {
		return nil, fmt.Errorf("diagnosis_id 无效")
	}

	// 1. 查询诊断记录
	var diagnosis models.DeviceDiagnosis
	if err := e.Orm.Table("device_diagnosis").
		Where("id = ? AND status = 0", in.DiagnosisId).
		Take(&diagnosis).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("诊断记录不存在或已完成")
		}
		return nil, err
	}

	// 2. 统计诊断结果
	totalItems := len(in.Items)
	normalItems := 0
	abnormalItems := 0

	for _, item := range in.Items {
		if item.Status == "normal" {
			normalItems++
		} else {
			abnormalItems++
		}
	}

	// 3. 计算健康评分（0-100）
	healthScore := int16(0)
	if totalItems > 0 {
		healthScore = int16(float64(normalItems) / float64(totalItems) * 100)
	}

	// 4. 序列化诊断结果
	resultJSON, _ := json.Marshal(in.Items)

	// 5. 更新诊断记录
	now := time.Now()
	updateData := map[string]interface{}{
		"status":         1, // 已完成
		"report_time":    in.ReportTime,
		"result":         string(resultJSON),
		"summary":        in.Summary,
		"total_items":    totalItems,
		"normal_items":   normalItems,
		"abnormal_items": abnormalItems,
		"health_score":   healthScore,
		"updated_at":     now,
	}

	if err := e.Orm.Model(&models.DeviceDiagnosis{}).
		Where("id = ?", in.DiagnosisId).
		Updates(updateData).Error; err != nil {
		return nil, err
	}

	return &ReportDiagnosisOut{
		DiagnosisId: in.DiagnosisId,
		DeviceID:    in.DeviceID,
		Sn:          in.Sn,
		Status:      "completed",
		Message:     "诊断结果已保存",
	}, nil
}

// GetDiagnosisResultIn 获取诊断结果输入
type GetDiagnosisResultIn struct {
	DiagnosisId int `json:"diagnosis_id"`
}

// GetDiagnosisResultOut 获取诊断结果输出
type GetDiagnosisResultOut struct {
	DiagnosisId   int                   `json:"diagnosis_id"`
	DeviceID      int64                 `json:"device_id"`
	SN            string                `json:"sn"`
	DiagType      string                `json:"diag_type"`
	Status        int16                 `json:"status"` // 0-诊断中 1-已完成 2-诊断失败
	ReportTime    *time.Time            `json:"report_time"`
	Result        []DeviceDiagnosisItem `json:"result"`
	Summary       string                `json:"summary"`
	TotalItems    int                   `json:"total_items"`
	NormalItems   int                   `json:"normal_items"`
	AbnormalItems int                   `json:"abnormal_items"`
	HealthScore   int16                 `json:"health_score"`
	Operator      string                `json:"operator"`
	CreatedAt     time.Time             `json:"created_at"`
}

// GetDiagnosisResult 获取诊断结果
func (e *PlatformDeviceService) GetDiagnosisResult(in *GetDiagnosisResultIn) (*GetDiagnosisResultOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	if in.DiagnosisId <= 0 {
		return nil, fmt.Errorf("diagnosis_id 无效")
	}

	// 查询诊断记录
	var diagnosis models.DeviceDiagnosis
	if err := e.Orm.Table("device_diagnosis").
		Where("id = ?", in.DiagnosisId).
		Take(&diagnosis).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("诊断记录不存在")
		}
		return nil, err
	}

	// 解析诊断结果
	var result []DeviceDiagnosisItem
	if diagnosis.Result != "" {
		json.Unmarshal([]byte(diagnosis.Result), &result)
	}

	return &GetDiagnosisResultOut{
		DiagnosisId:   diagnosis.Id,
		DeviceID:      diagnosis.DeviceId,
		SN:            diagnosis.SN,
		DiagType:      diagnosis.DiagType,
		Status:        diagnosis.Status,
		ReportTime:    diagnosis.ReportTime,
		Result:        result,
		Summary:       diagnosis.Summary,
		TotalItems:    diagnosis.TotalItems,
		NormalItems:   diagnosis.NormalItems,
		AbnormalItems: diagnosis.AbnormalItems,
		HealthScore:   diagnosis.HealthScore,
		Operator:      diagnosis.Operator,
		CreatedAt:     diagnosis.CreatedAt,
	}, nil
}

// sendDiagnosisCommand 通过 MQTT 下发诊断指令
func (e *PlatformDeviceService) sendDiagnosisCommand(sn string, command map[string]interface{}) {
	if cli := mqttadmin.Client(); cli != nil {
		b, _ := json.Marshal(command)
		topic := fmt.Sprintf("device/%s/command", sn)
		_ = cli.Publish(topic, 1, false, b)
	}
}

// GetDiagnosisHistoryIn 获取诊断历史输入
type GetDiagnosisHistoryIn struct {
	DeviceID  int64     `json:"device_id"`
	Sn        string    `json:"sn"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Page      int       `json:"page"`
	PageSize  int       `json:"page_size"`
}

// GetDiagnosisHistoryOut 获取诊断历史输出
type GetDiagnosisHistoryOut struct {
	List  []models.DeviceDiagnosis `json:"list"`
	Total int64                    `json:"total"`
	Page  int                      `json:"page"`
	Size  int                      `json:"size"`
}

// GetDiagnosisHistory 获取设备诊断历史
func (e *PlatformDeviceService) GetDiagnosisHistory(in *GetDiagnosisHistoryIn) (*GetDiagnosisHistoryOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	if in.DeviceID <= 0 && strings.TrimSpace(in.Sn) == "" {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询设备
	var dev struct {
		ID int64 `gorm:"column:id"`
	}
	q := e.Orm.Table("device").Select("id")
	if in.DeviceID > 0 {
		q = q.Where("id = ?", in.DeviceID)
	} else {
		q = q.Where("sn = ?", strings.TrimSpace(in.Sn))
	}
	if err := q.Take(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}

	// 构建查询条件
	query := e.Orm.Table("device_diagnosis").Where("device_id = ?", dev.ID)

	if !in.StartTime.IsZero() {
		query = query.Where("created_at >= ?", in.StartTime)
	}
	if !in.EndTime.IsZero() {
		query = query.Where("created_at <= ?", in.EndTime)
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	page := in.Page
	if page <= 0 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 查询列表
	var list []models.DeviceDiagnosis
	if err := query.Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, err
	}

	return &GetDiagnosisHistoryOut{
		List:  list,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// SetDeviceStatus 更新设备状态（正常/禁用/报废）
func (e *PlatformDeviceService) SetDeviceStatus(sn string, status int16, operator string) error {
	if e.Orm == nil {
		return fmt.Errorf("orm nil")
	}
	sn = strings.TrimSpace(sn)
	if sn == "" {
		return ErrPlatformDeviceNotFound
	}
	if status != 1 && status != 2 && status != 4 {
		return fmt.Errorf("status 仅支持 1/2/4")
	}
	op := strings.TrimSpace(operator)
	if op == "" {
		op = "admin"
	}
	return e.Orm.Transaction(func(tx *gorm.DB) error {
		var rec struct {
			ID int64 `gorm:"column:id"`
		}
		if err := tx.Table("device").Select("id").Where("sn = ?", sn).Take(&rec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPlatformDeviceNotFound
			}
			return err
		}
		id := rec.ID
		if err := tx.Exec(`UPDATE device SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id).Error; err != nil {
			return err
		}
		msg := fmt.Sprintf("状态变更为 %d", status)
		return tx.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator) VALUES (?,?,?,?,?)`,
			id, sn, "status_change", truncateEvent(msg), op).Error
	})
}

func truncateEvent(s string) string {
	if utf8.RuneCountInString(s) <= 250 {
		return s
	}
	r := []rune(s)
	return string(r[:250])
}

// AdminUnbind 后台强制解绑
func (e *PlatformDeviceService) AdminUnbind(sn string, operator string) error {
	if e.Orm == nil {
		return fmt.Errorf("orm nil")
	}
	sn = strings.TrimSpace(sn)
	if sn == "" {
		return ErrPlatformDeviceNotFound
	}
	op := strings.TrimSpace(operator)
	if op == "" {
		op = "admin"
	}
	return e.Orm.Transaction(func(tx *gorm.DB) error {
		var rec struct {
			ID int64 `gorm:"column:id"`
		}
		if err := tx.Table("device").Select("id").Where("sn = ?", sn).Take(&rec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPlatformDeviceNotFound
			}
			return err
		}
		id := rec.ID
		r := tx.Exec(`UPDATE user_device_bind SET status = 0, unbound_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE device_id = ? AND status = 1`, id)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return fmt.Errorf("当前无有效绑定")
		}
		return tx.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator) VALUES (?,?,?,?,?)`,
			id, sn, "admin_unbind", "管理员强制解绑", op).Error
	})
}

// AdminForceUnbindIn 后台强制解绑（支持 device_id / sn、原因、审计）
type AdminForceUnbindIn struct {
	DeviceID int64
	Sn       string
	Reason   string
	Confirm  bool
	Operator string
}

// AdminForceUnbindOut 与前端约定返回
type AdminForceUnbindOut struct {
	DeviceSn   string `json:"device_sn"`
	OldUserID  int64  `json:"old_user_id"`
	UnbindType int    `json:"unbind_type"` // 固定 2=后台强制
	UnbindTime string `json:"unbind_time"` // 本地时间 YYYY-MM-DD HH:MM:SS
}

const unbindTypeAdminForce = 2

// AdminForceUnbind 解除 user_device_bind 活跃绑定；无绑定则幂等成功。设备须为「正常」态（status=1）。
func (e *PlatformDeviceService) AdminForceUnbind(in *AdminForceUnbindIn) (*AdminForceUnbindOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	if !in.Confirm {
		return nil, fmt.Errorf("请确认强制解绑：传 confirm=true")
	}

	var dev struct {
		ID     int64  `gorm:"column:id"`
		Sn     string `gorm:"column:sn"`
		Status int16  `gorm:"column:status"`
	}
	q := e.Orm.Table("device").Select("id, sn, status")
	if in.DeviceID > 0 {
		q = q.Where("id = ?", in.DeviceID)
	} else if strings.TrimSpace(in.Sn) != "" {
		q = q.Where("sn = ?", strings.TrimSpace(in.Sn))
	} else {
		return nil, fmt.Errorf("请提供 device_id 或 sn")
	}
	if err := q.Take(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}
	if dev.Status != 1 {
		return nil, fmt.Errorf("设备已禁用、未激活或已报废，无法解绑")
	}

	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "admin"
	}
	reason := strings.TrimSpace(in.Reason)
	now := time.Now()
	unbindTimeStr := now.Format("2006-01-02 15:04:05")

	var oldUID int64
	var hadBind bool

	err := e.Orm.Transaction(func(tx *gorm.DB) error {
		type bindRow struct {
			UserID int64 `gorm:"column:user_id"`
		}
		var br bindRow
		err := tx.Table("user_device_bind").Select("user_id").Where("device_id = ? AND status = 1", dev.ID).Take(&br).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		oldUID = br.UserID
		hadBind = true
		r := tx.Exec(`UPDATE user_device_bind SET status = 0, unbound_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE device_id = ? AND status = 1`, dev.ID)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			hadBind = false
			oldUID = 0
			return nil
		}
		content := fmt.Sprintf("后台强制解绑 old_user_id=%d", oldUID)
		if reason != "" {
			content = content + " reason=" + reason
		}
		return tx.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator) VALUES (?,?,?,?,?)`,
			dev.ID, dev.Sn, "admin_force_unbind", truncateEvent(content), op).Error
	})
	if err != nil {
		return nil, err
	}

	out := &AdminForceUnbindOut{
		DeviceSn:   dev.Sn,
		OldUserID:  oldUID,
		UnbindType: unbindTypeAdminForce,
		UnbindTime: unbindTimeStr,
	}
	if !hadBind {
		out.OldUserID = 0
		return out, nil
	}

	// 通知设备端（可选）：与远程指令同主题，便于设备侧统一处理
	if cli := mqttadmin.Client(); cli != nil {
		payload := map[string]interface{}{
			"cmd":         "unbind",
			"source":      "admin",
			"device_sn":   dev.Sn,
			"old_user_id": oldUID,
			"timestamp":   now.Unix(),
			"unbind_type": unbindTypeAdminForce,
		}
		if reason != "" {
			payload["reason"] = reason
		}
		b, _ := json.Marshal(payload)
		topic := fmt.Sprintf("device/%s/command", dev.Sn)
		_ = cli.Publish(topic, 0, false, b)
	}

	return out, nil
}

// EnqueueCommand 远程指令入队（device_instruction + 事件）
func (e *PlatformDeviceService) EnqueueCommand(sn, command string, params map[string]interface{}, operator string) error {
	if e.Orm == nil {
		return fmt.Errorf("orm nil")
	}
	sn = strings.TrimSpace(sn)
	cmd := strings.TrimSpace(command)
	if sn == "" || cmd == "" {
		return ErrPlatformDeviceInvalid
	}
	op := strings.TrimSpace(operator)
	if op == "" {
		op = "admin"
	}
	b, _ := json.Marshal(params)
	if len(b) == 0 {
		b = []byte("{}")
	}
	return e.Orm.Transaction(func(tx *gorm.DB) error {
		var rec struct {
			ID int64 `gorm:"column:id"`
		}
		if err := tx.Table("device").Select("id").Where("sn = ?", sn).Take(&rec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPlatformDeviceNotFound
			}
			return err
		}
		id := rec.ID
		if err := tx.Exec(`INSERT INTO device_instruction (device_id, sn, user_id, cmd, params, status, operator, reason)
			VALUES (?,?,0,?,?::jsonb,?,?,?)`, id, sn, cmd, string(b), deviceInstructionPending, op, "").Error; err != nil {
			return err
		}
		msg := fmt.Sprintf("指令 %s 已入队", cmd)
		return tx.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator) VALUES (?,?,?,?,?)`,
			id, sn, "command", truncateEvent(msg), op).Error
	})
}

// EnqueueOTA 创建 OTA 升级任务
func (e *PlatformDeviceService) EnqueueOTA(sn, version string, operator string) error {
	if e.Orm == nil {
		return fmt.Errorf("orm nil")
	}
	sn = strings.TrimSpace(sn)
	ver := strings.TrimSpace(version)
	if sn == "" || ver == "" {
		return ErrPlatformDeviceInvalid
	}
	op := strings.TrimSpace(operator)
	if op == "" {
		op = "admin"
	}
	return e.Orm.Transaction(func(tx *gorm.DB) error {
		type dev struct {
			ID         int64  `gorm:"column:id"`
			ProductKey string `gorm:"column:product_key"`
			FwVer      string `gorm:"column:firmware_version"`
		}
		var d dev
		if err := tx.Table("device").Select("id, product_key, firmware_version").Where("sn = ?", sn).Take(&d).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPlatformDeviceNotFound
			}
			return err
		}
		var fw struct {
			ID int64 `gorm:"column:id"`
		}
		if err := tx.Table("ota_firmware").Select("id").Where("product_key = ? AND version = ? AND deleted_at IS NULL", d.ProductKey, ver).Take(&fw).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("未找到对应固件包 product_key=%s version=%s", d.ProductKey, ver)
			}
			return err
		}
		if err := tx.Exec(`INSERT INTO ota_upgrade_task (device_id, sn, firmware_id, from_version, to_version, status, progress, error_msg)
			VALUES (?,?,?,?,?,1,0,'')`, d.ID, sn, fw.ID, d.FwVer, ver).Error; err != nil {
			return err
		}
		msg := fmt.Sprintf("OTA 任务已创建 -> %s", ver)
		return tx.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator) VALUES (?,?,?,?,?)`,
			d.ID, sn, "ota", truncateEvent(msg), op).Error
	})
}

// PlatformDeviceSummaryOut 设备看板统计
type PlatformDeviceSummaryOut struct {
	Total       int64 `json:"total"`
	Online      int64 `json:"online"`
	Offline     int64 `json:"offline"`
	Unbound     int64 `json:"unbound"`
	TodayNew    int64 `json:"today_new"`
	TodayActive int64 `json:"today_active"`
}

// GetSummary 看板统计（全库）
func (e *PlatformDeviceService) GetSummary() (*PlatformDeviceSummaryOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	var out PlatformDeviceSummaryOut
	if err := e.Orm.Table("device").Count(&out.Total).Error; err != nil {
		return nil, err
	}
	if err := e.Orm.Table("device").
		Where("online_status = 1 OR (last_active_at IS NOT NULL AND last_active_at > NOW() - INTERVAL '5 minutes')").
		Count(&out.Online).Error; err != nil {
		return nil, err
	}
	if out.Total >= out.Online {
		out.Offline = out.Total - out.Online
	}
	if err := e.Orm.Raw(`
SELECT COUNT(*) FROM device d
WHERE NOT EXISTS (SELECT 1 FROM user_device_bind udb WHERE udb.device_id = d.id AND udb.status = 1)`).Scan(&out.Unbound).Error; err != nil {
		return nil, err
	}
	_ = e.Orm.Table("device").Where("created_at::date = CURRENT_DATE").Count(&out.TodayNew).Error
	_ = e.Orm.Table("device").Where("last_active_at::date = CURRENT_DATE").Count(&out.TodayActive).Error
	return &out, nil
}

// ListProductKeys 列表中出现过的 product_key（下拉）
func (e *PlatformDeviceService) ListProductKeys() ([]string, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	rows, err := e.Orm.Raw(`SELECT DISTINCT product_key FROM device WHERE TRIM(product_key) <> '' ORDER BY product_key`).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// DeviceImportRow 批量导入一行（JSON 同步接口；建议 mac 必填）
type DeviceImportRow struct {
	Sn           string `json:"sn"`
	ProductKey   string `json:"product_key"`
	Model        string `json:"model"`
	Mac          string `json:"mac"`
	DeviceName   string `json:"device_name"`
	Remark       string `json:"remark"`
	PresetSecret string `json:"preset_secret"`
}

// ImportDevices 批量导入（与单条相同校验；mac 非空时校验格式与唯一性）
func (e *PlatformDeviceService) ImportDevices(rows []DeviceImportRow) (success int, failLines []string) {
	for i, it := range rows {
		line := i + 1
		req := &ProvisionIn{
			Sn:                strings.TrimSpace(it.Sn),
			ProductKey:        strings.TrimSpace(it.ProductKey),
			Model:             strings.TrimSpace(it.Model),
			Mac:               strings.TrimSpace(it.Mac),
			AdminDisplayName:  strings.TrimSpace(it.DeviceName),
			AdminRemark:       strings.TrimSpace(it.Remark),
			PlainPresetSecret: strings.TrimSpace(it.PresetSecret),
			RequireMAC:        true,
		}
		if _, err := e.RegisterDeviceWithOptions(req); err != nil {
			failLines = append(failLines, fmt.Sprintf("行%d %s: %v", line, strings.TrimSpace(it.Sn), err))
			continue
		}
		success++
	}
	return success, failLines
}

// BatchSetStatus 批量更新状态（1/2/4）
func (e *PlatformDeviceService) BatchSetStatus(sns []string, status int16, operator string) (int, error) {
	if status != 1 && status != 2 && status != 4 {
		return 0, fmt.Errorf("status 仅支持 1/2/4")
	}
	n := 0
	var lastErr error
	for _, sn := range sns {
		sn = strings.TrimSpace(sn)
		if sn == "" {
			continue
		}
		if err := e.SetDeviceStatus(sn, status, operator); err != nil {
			lastErr = err
			continue
		}
		n++
	}
	if n == 0 && lastErr != nil {
		return 0, lastErr
	}
	return n, nil
}

// AdminRemoteCommandIn 后台远程指令（重启 / 恢复出厂 / 诊断 / 立即上报状态）
type AdminRemoteCommandIn struct {
	DeviceID            int64
	Sn                  string
	Command             string
	Reason              string
	ConfirmFactoryReset bool
	Operator            string
	Params              map[string]interface{}
}

// AdminRemoteCommandOut 返回给前端的统一结构
type AdminRemoteCommandOut struct {
	CommandID  string `json:"command_id"`
	DeviceSn   string `json:"device_sn"`
	Command    string `json:"command"`
	Status     int    `json:"status"` // 0 待下发 1 已下发（与需求文档一致）
	StatusText string `json:"status_text"`
}

func randomHexBytes(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte("00000000"))
	}
	return hex.EncodeToString(b)
}

func mustJSONMap(v map[string]interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func dbInstructionToAPIStatus(db int16) (int, string) {
	switch db {
	case 1:
		return 0, "待下发"
	case 2:
		return 1, "执行中"
	case 3:
		return 2, "执行成功"
	case 4:
		return 3, "执行失败"
	case 5:
		return 4, "已超时"
	case 6:
		return 5, "已取消"
	default:
		return 0, "未知"
	}
}

// AdminRemoteCommand 管理员远程指令：校验设备 → 落库 device_instruction → MQTT → 更新状态 → 事件日志（含 report_status）
func (e *PlatformDeviceService) AdminRemoteCommand(in *AdminRemoteCommandIn) (*AdminRemoteCommandOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	cmd := strings.TrimSpace(strings.ToLower(in.Command))
	if cmd == "reboot" {
		cmd = "restart"
	}
	if cmd != "restart" && cmd != "factory_reset" && cmd != "diagnosis" && cmd != "report_status" {
		return nil, fmt.Errorf("非法指令，仅支持 restart、factory_reset、diagnosis、report_status")
	}
	if cmd == "factory_reset" && !in.ConfirmFactoryReset {
		return nil, fmt.Errorf("恢复出厂需二次确认：请传 confirm_factory_reset=true")
	}

	var dev struct {
		ID     int64  `gorm:"column:id"`
		Sn     string `gorm:"column:sn"`
		Status int16  `gorm:"column:status"`
	}
	q := e.Orm.Table("device").Select("id, sn, status")
	if in.DeviceID > 0 {
		q = q.Where("id = ?", in.DeviceID)
	} else if strings.TrimSpace(in.Sn) != "" {
		snNorm := strings.ToUpper(strings.TrimSpace(in.Sn))
		q = q.Where("UPPER(TRIM(sn)) = ? AND deleted_at IS NULL", snNorm)
	} else {
		return nil, fmt.Errorf("请提供 device_id 或 sn")
	}
	if err := q.Take(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}
	if dev.Status != 1 {
		return nil, fmt.Errorf("设备已禁用、未激活或已报废，无法下发指令")
	}

	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "admin"
	}
	nonce := randomHexBytes(8)
	now := time.Now().Unix()
	reason := strings.TrimSpace(in.Reason)

	pre := map[string]interface{}{
		"reason": reason,
		"source": "admin",
		"nonce":  nonce,
	}
	for k, v := range in.Params {
		pre[k] = v
	}
	preJSON, _ := json.Marshal(pre)

	var instrID int64
	err := e.Orm.Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(`INSERT INTO device_instruction (device_id, sn, user_id, cmd, params, status, operator, reason)
			VALUES (?,?,0,?,?::jsonb,?,?,?) RETURNING id`,
			dev.ID, dev.Sn, cmd, string(preJSON), deviceInstructionPending, op, reason).Scan(&instrID).Error; err != nil {
			return err
		}
		commandID := fmt.Sprintf("cmd_%d", instrID)
		full := map[string]interface{}{
			"command_id": commandID,
			"device_sn":  dev.Sn,
			"cmd":        cmd,
			"timestamp":  now,
			"nonce":      nonce,
			"sign":       "",
			"reason":     reason,
			"source":     "admin",
		}
		for k, v := range in.Params {
			full[k] = v
		}
		fullJSON, _ := json.Marshal(full)
		if err := tx.Exec(`UPDATE device_instruction SET params = ?::jsonb WHERE id = ?`, string(fullJSON), instrID).Error; err != nil {
			return err
		}
		evContent := fmt.Sprintf("远程指令 %s %s", cmd, commandID)
		if reason != "" {
			evContent = evContent + " " + reason
		}
		if err := tx.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator) VALUES (?,?,?,?,?)`,
			dev.ID, dev.Sn, "admin_remote_command", truncateEvent(evContent), op).Error; err != nil {
			return err
		}
		_ = InsertInstructionStateLog(tx, instrID, nil, InstrStatusPending, "指令已创建，等待下发", op)
		return nil
	})
	if err != nil {
		return nil, err
	}

	commandID := fmt.Sprintf("cmd_%d", instrID)
	mqttBody := map[string]interface{}{
		"command_id": commandID,
		"device_sn":  dev.Sn,
		"cmd":        cmd,
		"timestamp":  now,
		"nonce":      nonce,
		"sign":       "",
		"source":     "admin",
	}
	mqttJSON, _ := json.Marshal(mqttBody)
	topic := fmt.Sprintf("device/%s/command", dev.Sn)

	cli := mqttadmin.Client()
	st := deviceInstructionPending
	if cli != nil {
		token := cli.Publish(topic, 0, false, mqttJSON)
		if token.Wait() && token.Error() != nil {
			return nil, fmt.Errorf("MQTT 推送失败: %v", token.Error())
		}
		st = deviceInstructionSent
		if err := e.Orm.Exec(`UPDATE device_instruction SET status = ?, received_at = COALESCE(received_at, CURRENT_TIMESTAMP), updated_at = CURRENT_TIMESTAMP WHERE id = ?`, st, instrID).Error; err != nil {
			return nil, err
		}
		fp := InstrStatusPending
		_ = InsertInstructionStateLog(e.Orm, instrID, &fp, InstrStatusExecuting, "已下发至设备（MQTT）", op)
	}

	apiCode, apiText := dbInstructionToAPIStatus(st)
	return &AdminRemoteCommandOut{
		CommandID:  commandID,
		DeviceSn:   dev.Sn,
		Command:    cmd,
		Status:     apiCode,
		StatusText: apiText,
	}, nil
}

// DeviceConfigItem 配置项
type DeviceConfigItem struct {
	ConfigKey   string `json:"config_key"`
	ConfigValue string `json:"config_value"`
	ConfigType  string `json:"config_type"`
}

// GetDeviceConfigIn 获取设备配置输入
type GetDeviceConfigIn struct {
	DeviceID int64
	Sn       string
}

// GetDeviceConfigOut 获取设备配置输出
type GetDeviceConfigOut struct {
	DeviceID int64              `json:"device_id"`
	Sn       string             `json:"sn"`
	Configs  []DeviceConfigItem `json:"configs"`
}

// GetDeviceConfig 获取设备配置
func (e *PlatformDeviceService) GetDeviceConfig(in *GetDeviceConfigIn) (*GetDeviceConfigOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in.DeviceID <= 0 && strings.TrimSpace(in.Sn) == "" {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询设备
	var dev struct {
		ID int64  `gorm:"column:id"`
		Sn string `gorm:"column:sn"`
	}
	q := e.Orm.Table("device").Select("id, sn")
	if in.DeviceID > 0 {
		q = q.Where("id = ?", in.DeviceID)
	} else {
		q = q.Where("sn = ?", strings.TrimSpace(in.Sn))
	}
	if err := q.Take(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}

	// 查询设备配置
	var configs []models.DeviceConfig
	if err := e.Orm.Table("device_config").
		Where("device_id = ? AND status = 1", dev.ID).
		Order("config_key ASC").
		Find(&configs).Error; err != nil {
		return nil, err
	}

	items := make([]DeviceConfigItem, 0, len(configs))
	for _, c := range configs {
		items = append(items, DeviceConfigItem{
			ConfigKey:   c.ConfigKey,
			ConfigValue: c.ConfigValue,
			ConfigType:  c.ConfigType,
		})
	}

	return &GetDeviceConfigOut{
		DeviceID: dev.ID,
		Sn:       dev.Sn,
		Configs:  items,
	}, nil
}

// UpdateDeviceConfigIn 更新设备配置输入
type UpdateDeviceConfigIn struct {
	DeviceID int64              `json:"device_id"`
	Sn       string             `json:"sn"`
	Configs  []DeviceConfigItem `json:"configs"`
	Operator string             `json:"operator"`
}

// UpdateDeviceConfigOut 更新设备配置输出
type UpdateDeviceConfigOut struct {
	DeviceID int64  `json:"device_id"`
	Sn       string `json:"sn"`
	Success  int    `json:"success"`
	Message  string `json:"message"`
}

// UpdateDeviceConfig 更新设备配置
// 1. 校验设备存在且状态正常
// 2. 校验配置参数格式
// 3. 保存配置到数据库
// 4. 通过 MQTT 下发配置到设备
// 5. 更新设备影子
// 6. 记录操作日志
func (e *PlatformDeviceService) UpdateDeviceConfig(in *UpdateDeviceConfigIn) (*UpdateDeviceConfigOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	if in.DeviceID <= 0 && strings.TrimSpace(in.Sn) == "" {
		return nil, ErrPlatformDeviceInvalid
	}
	if len(in.Configs) == 0 {
		return nil, fmt.Errorf("配置参数不能为空")
	}

	// 1. 查询设备
	var dev struct {
		ID           int64      `gorm:"column:id"`
		Sn           string     `gorm:"column:sn"`
		Status       int16      `gorm:"column:status"`
		OnlineStatus int16      `gorm:"column:online_status"`
		LastActiveAt *time.Time `gorm:"column:last_active_at"`
	}
	q := e.Orm.Table("device").Select("id, sn, status, online_status, last_active_at")
	if in.DeviceID > 0 {
		q = q.Where("id = ?", in.DeviceID)
	} else {
		q = q.Where("sn = ?", strings.TrimSpace(in.Sn))
	}
	if err := q.Take(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}
	if dev.Status != 1 {
		return nil, fmt.Errorf("设备已禁用，无法更新配置")
	}

	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "admin"
	}
	now := time.Now()

	success := 0
	var lastErr error

	// 2. 遍历配置项，逐个处理
	for _, cfg := range in.Configs {
		// 校验配置参数
		if strings.TrimSpace(cfg.ConfigKey) == "" {
			lastErr = fmt.Errorf("config_key 不能为空")
			continue
		}
		if len(cfg.ConfigKey) > 64 {
			lastErr = fmt.Errorf("config_key 长度不能超过 64")
			continue
		}

		// 3. 保存配置到数据库（使用 UPSERT 逻辑）
		var existing models.DeviceConfig
		err := e.Orm.Table("device_config").
			Where("device_id = ? AND config_key = ?", dev.ID, cfg.ConfigKey).
			Take(&existing).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 新增配置
			newConfig := models.DeviceConfig{
				DeviceId:    dev.ID,
				SN:          dev.Sn,
				ConfigKey:   cfg.ConfigKey,
				ConfigValue: cfg.ConfigValue,
				ConfigType:  cfg.ConfigType,
				Status:      1,
				ApplyStatus: 0, // 待下发
				CreateBy:    0,
				UpdateBy:    0,
			}
			if err := e.Orm.Create(&newConfig).Error; err != nil {
				lastErr = err
				continue
			}
			success++
		} else if err != nil {
			lastErr = err
			continue
		} else {
			// 更新配置
			if err := e.Orm.Model(&models.DeviceConfig{}).
				Where("id = ?", existing.Id).
				Updates(map[string]interface{}{
					"config_value": cfg.ConfigValue,
					"config_type":  cfg.ConfigType,
					"apply_status": 0, // 重置为待下发
					"updated_at":   now,
				}).Error; err != nil {
				lastErr = err
				continue
			}
			success++
		}

		// 4. 通过 MQTT 下发配置到设备
		go e.sendConfigViaMQTT(dev.Sn, cfg.ConfigKey, cfg.ConfigValue, cfg.ConfigType)
	}

	// 5. 记录操作日志
	if success > 0 {
		go func() {
			logContent := fmt.Sprintf("更新设备配置 success=%d configs=%d", success, len(in.Configs))
			_ = e.Orm.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator, created_at) VALUES (?,?,?,?,?,?)`,
				dev.ID, dev.Sn, "config_update", truncateEvent(logContent), op, now)
		}()
	}

	if lastErr != nil && success == 0 {
		return nil, lastErr
	}

	return &UpdateDeviceConfigOut{
		DeviceID: dev.ID,
		Sn:       dev.Sn,
		Success:  success,
		Message:  fmt.Sprintf("成功更新 %d 项配置", success),
	}, nil
}

// sendConfigViaMQTT 通过 MQTT 下发配置
func (e *PlatformDeviceService) sendConfigViaMQTT(sn, configKey, configValue, configType string) {
	if cli := mqttadmin.Client(); cli != nil {
		configData := map[string]interface{}{
			"cmd":          "update_config",
			"config_key":   configKey,
			"config_value": configValue,
			"config_type":  configType,
			"timestamp":    time.Now().Unix(),
			"source":       "admin",
		}
		b, _ := json.Marshal(configData)
		topic := fmt.Sprintf("device/%s/command", sn)
		_ = cli.Publish(topic, 0, false, b)
	}
}

// AdminDeleteDeviceIn 后台删除设备输入参数
type AdminDeleteDeviceIn struct {
	DeviceID int64
	Sn       string
	Operator string
	Confirm  bool
}

// AdminDeleteDeviceOut 后台删除设备输出
type AdminDeleteDeviceOut struct {
	DeviceSn string `json:"device_sn"`
	DeviceID int64  `json:"device_id"`
	Success  bool   `json:"success"`
}

// AdminDeleteDevice 后台删除设备（软删除）
// 1. 检查设备是否存在
// 2. 检查设备状态（在线不能删除）
// 3. 执行软删除
// 4. 删除关联数据（绑定关系、影子、指令、OTA、事件等）
// 5. 清理缓存
// 6. 记录操作日志
func (e *PlatformDeviceService) AdminDeleteDevice(in *AdminDeleteDeviceIn) (*AdminDeleteDeviceOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	if !in.Confirm {
		return nil, fmt.Errorf("请确认删除设备：传 confirm=true")
	}

	// 查询设备信息
	var dev struct {
		ID           int64      `gorm:"column:id"`
		Sn           string     `gorm:"column:sn"`
		Status       int16      `gorm:"column:status"`
		OnlineStatus int16      `gorm:"column:online_status"`
		LastActiveAt *time.Time `gorm:"column:last_active_at"`
	}
	q := e.Orm.Table("device").Select("id, sn, status, online_status, last_active_at")
	if in.DeviceID > 0 {
		q = q.Where("id = ?", in.DeviceID)
	} else if strings.TrimSpace(in.Sn) != "" {
		q = q.Where("sn = ?", strings.TrimSpace(in.Sn))
	} else {
		return nil, fmt.Errorf("请提供 device_id 或 sn")
	}
	if err := q.Take(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}

	// 检查设备状态：已禁用或已报废的设备不能删除（避免重复操作）
	if dev.Status == 2 || dev.Status == 4 {
		return nil, fmt.Errorf("设备已禁用或已报废，无需重复删除")
	}

	// 检查设备是否在线（在线设备不允许删除，避免业务中断）
	if dev.OnlineStatus == 1 || (dev.LastActiveAt != nil && time.Since(*dev.LastActiveAt) <= 5*time.Minute) {
		return nil, fmt.Errorf("设备在线，无法删除。请先断开设备连接或等待 5 分钟后重试")
	}

	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "admin"
	}
	now := time.Now()

	err := e.Orm.Transaction(func(tx *gorm.DB) error {
		deviceID := dev.ID
		deviceSn := dev.Sn

		// 1. 执行软删除设备
		if err := tx.Exec(`UPDATE device SET status = 2, deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, deviceID).Error; err != nil {
			return err
		}

		// 2. 删除用户设备绑定关系
		if err := tx.Exec(`UPDATE user_device_bind SET status = 0, unbound_at = CURRENT_TIMESTAMP, deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE device_id = ?`, deviceID).Error; err != nil {
			return err
		}

		// 3. 删除设备影子（如果有独立的影子表）
		// 注：如果影子存储在 Redis，需要在下方清理缓存时处理
		_ = tx.Exec(`DELETE FROM device_shadow WHERE device_id = ?`, deviceID).Error

		// 4. 删除设备指令记录
		_ = tx.Exec(`DELETE FROM device_instruction WHERE device_id = ?`, deviceID).Error

		// 5. 删除 OTA 升级任务记录
		_ = tx.Exec(`DELETE FROM ota_upgrade_task WHERE device_id = ?`, deviceID).Error

		// 6. 删除设备事件日志（可选保留审计）
		// _ = tx.Exec(`DELETE FROM device_event_log WHERE device_id = ?`, deviceID).Error

		// 7. 记录删除事件日志（必须在删除前记录）
		logContent := fmt.Sprintf("管理员删除设备 old_status=%d old_online=%d", dev.Status, dev.OnlineStatus)
		if err := tx.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator, created_at) VALUES (?,?,?,?,?,?)`,
			deviceID, deviceSn, "admin_delete", truncateEvent(logContent), op, now).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 8. 清理缓存（Redis）
	// 如果使用了 Redis 缓存设备信息，需要清理
	// 示例：删除设备影子缓存
	// redisClient.Del(ctx, fmt.Sprintf("device:shadow:%d", dev.ID))
	// 删除用户设备列表缓存
	// redisClient.Del(ctx, fmt.Sprintf("user:devices:%d", userID))

	return &AdminDeleteDeviceOut{
		DeviceSn: dev.Sn,
		DeviceID: dev.ID,
		Success:  true,
	}, nil
}

// DeviceLogLevel 日志级别常量
const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
	LogLevelFatal = "fatal"
)

// DeviceLogType 日志类型常量
const (
	LogTypeSystem    = "system"    // 系统日志
	LogTypeError     = "error"     // 异常日志
	LogTypeOperation = "operation" // 操作日志
	LogTypeStatus    = "status"    // 状态日志
)

// DeviceLogItem 日志项
type DeviceLogItem struct {
	LogType      string                 `json:"log_type"`      // system/error/operation/status
	LogLevel     string                 `json:"log_level"`     // debug/info/warn/error/fatal
	Content      string                 `json:"content"`       // 日志内容
	ErrorCode    *int                   `json:"error_code"`    // 错误码（可选）
	Extra        map[string]interface{} `json:"extra"`         // 额外信息
	ReportTime   time.Time              `json:"report_time"`   // 上报时间
	ReportSource string                 `json:"report_source"` // device/cloud
	IpAddress    string                 `json:"ip_address"`    // IP 地址
}

// ReportDeviceLogIn 设备日志上报输入
type ReportDeviceLogIn struct {
	DeviceID int64           `json:"device_id"`
	Sn       string          `json:"sn"`
	Logs     []DeviceLogItem `json:"logs"`
}

// ReportDeviceLogOut 设备日志上报输出
type ReportDeviceLogOut struct {
	DeviceID int64  `json:"device_id"`
	Sn       string `json:"sn"`
	Success  int    `json:"success"`
	Message  string `json:"message"`
}

// ReportDeviceLog 设备日志上报
// 1. 校验设备存在且状态正常
// 2. 校验日志格式和必填字段
// 3. 清洗、过滤、分级日志
// 4. 批量保存日志到数据库
// 5. 更新设备最新日志记录
// 6. 对错误级别日志触发告警
// 7. 返回上报成功响应
func (e *PlatformDeviceService) ReportDeviceLog(in *ReportDeviceLogIn) (*ReportDeviceLogOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	if in.DeviceID <= 0 && strings.TrimSpace(in.Sn) == "" {
		return nil, ErrPlatformDeviceInvalid
	}
	if len(in.Logs) == 0 {
		return nil, fmt.Errorf("日志列表不能为空")
	}

	// 1. 查询设备
	var dev struct {
		ID     int64  `gorm:"column:id"`
		Sn     string `gorm:"column:sn"`
		Status int16  `gorm:"column:status"`
	}
	q := e.Orm.Table("device").Select("id, sn, status")
	if in.DeviceID > 0 {
		q = q.Where("id = ?", in.DeviceID)
	} else {
		q = q.Where("sn = ?", strings.TrimSpace(in.Sn))
	}
	if err := q.Take(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}
	if dev.Status != 1 {
		return nil, fmt.Errorf("设备已禁用，无法上报日志")
	}

	success := 0
	var lastErr error

	// 2. 遍历日志项，逐个处理
	for _, logItem := range in.Logs {
		// 校验日志格式
		if err := validateLogLevel(logItem.LogLevel); err != nil {
			lastErr = err
			continue
		}
		if err := validateLogType(logItem.LogType); err != nil {
			lastErr = err
			continue
		}
		if strings.TrimSpace(logItem.Content) == "" {
			lastErr = fmt.Errorf("日志内容不能为空")
			continue
		}

		// 3. 清洗日志内容（限制长度）
		cleanedContent := truncateLogContent(logItem.Content, 4096)

		// 4. 序列化额外信息
		var extraJSON string
		if len(logItem.Extra) > 0 {
			b, _ := json.Marshal(logItem.Extra)
			extraJSON = string(b)
		}

		// 5. 保存日志到数据库
		log := models.DeviceLog{
			DeviceId:     dev.ID,
			SN:           dev.Sn,
			LogType:      logItem.LogType,
			LogLevel:     logItem.LogLevel,
			Content:      cleanedContent,
			ErrorCode:    logItem.ErrorCode,
			Extra:        extraJSON,
			ReportTime:   logItem.ReportTime,
			ReportSource: logItem.ReportSource,
			IpAddress:    logItem.IpAddress,
			Processed:    0, // 未处理
			AlertSent:    0, // 未发送告警
			CreateBy:     0,
			UpdateBy:     0,
		}

		if err := e.Orm.Create(&log).Error; err != nil {
			lastErr = err
			continue
		}
		success++

		// 6. 对错误级别日志触发告警
		if logItem.LogLevel == LogLevelError || logItem.LogLevel == LogLevelFatal {
			go e.sendAlert(log, dev.Sn)
		}
	}

	if lastErr != nil && success == 0 {
		return nil, lastErr
	}

	return &ReportDeviceLogOut{
		DeviceID: dev.ID,
		Sn:       dev.Sn,
		Success:  success,
		Message:  fmt.Sprintf("成功上报 %d 条日志", success),
	}, nil
}

// validateLogLevel 校验日志级别
func validateLogLevel(level string) error {
	validLevels := map[string]bool{
		LogLevelDebug: true,
		LogLevelInfo:  true,
		LogLevelWarn:  true,
		LogLevelError: true,
		LogLevelFatal: true,
	}
	if !validLevels[level] {
		return fmt.Errorf("无效的日志级别：%s", level)
	}
	return nil
}

// FirmwareUpdateRequest 固件更新请求
type FirmwareUpdateRequest struct {
	FirmwareID    int64
	ProductKey    string
	Version       string
	VersionDesc   string
	DeviceModels  []string
	ForceUpdate   *bool
	MinSysVersion string
	Status        *int16
	Tags          []string
	Confirm       bool
	Operator      int64
}

// FirmwareUpdateResponse 固件更新响应
type FirmwareUpdateResponse struct {
	FirmwareID    int64    `json:"firmware_id"`
	UpdatedFields []string `json:"updated_fields"`
	UpdatedAt     string   `json:"updated_at"`
	Message       string   `json:"message"`
	Success       bool     `json:"success"`
}

// FirmwareUpdate 固件信息更新（不修改固件包文件）
// 处理流程：
// 1. 参数解析，定位目标记录
// 2. 权限校验
// 3. 版本号保护校验（禁止修改版本号）
// 4. 字段冲突校验
// 5. 任务关联检查
// 6. 数据更新
// 7. 缓存清理
// 8. 日志记录
func (e *PlatformDeviceService) FirmwareUpdate(in FirmwareUpdateRequest) (*FirmwareUpdateResponse, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}

	// 1. 参数解析，查询固件记录
	var fw struct {
		ID            int64     `gorm:"column:id"`
		ProductKey    string    `gorm:"column:product_key"`
		Version       string    `gorm:"column:version"`
		VersionCode   int       `gorm:"column:version_code"`
		VersionDesc   string    `gorm:"column:version_description"`
		DeviceModels  string    `gorm:"column:device_models"`
		ForceUpdate   bool      `gorm:"column:force_update"`
		MinSysVersion string    `gorm:"column:min_sys_version"`
		Status        int16     `gorm:"column:status"`
		Tags          string    `gorm:"column:tags"`
		FileSize      int64     `gorm:"column:file_size"`
		FileMd5       string    `gorm:"column:file_md5"`
		DownloadCount int64     `gorm:"column:download_count"`
		CreatedAt     time.Time `gorm:"column:created_at"`
		UpdatedAt     time.Time `gorm:"column:updated_at"`
		DeletedAt     gorm.DeletedAt
	}

	q := e.Orm.Table("ota_firmware").Select("id, product_key, version, version_code, version_description, device_models, force_update, min_sys_version, status, tags, file_size, file_md5, download_count, created_at, updated_at")
	if in.FirmwareID > 0 {
		q = q.Where("id = ?", in.FirmwareID)
	} else if in.ProductKey != "" && in.Version != "" {
		q = q.Where("product_key = ? AND version = ?", in.ProductKey, in.Version)
	} else {
		return nil, ErrFirmwareNotFound
	}

	if err := q.Take(&fw).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFirmwareNotFound
		}
		return nil, err
	}

	// 2. 权限校验（这里简化处理，实际应该检查操作人是否有 firmware:manage 权限）
	// 如果 operator <= 0 说明未登录，已在 API 层校验

	// 3. 版本号保护校验 - 禁止修改版本号和版本码
	if in.Version != "" && in.Version != fw.Version {
		return nil, ErrFirmwareVersionProtected
	}

	// 4. 字段冲突校验 - 检查是否有禁止修改的字段
	// 文件大小、MD5 等也不允许修改（这些字段不在更新参数中，所以不会被动到）

	// 5. 任务关联检查 - 检查是否有关联的进行中升级任务
	var taskCount int64
	if err := e.Orm.Table("ota_upgrade_task").
		Where("firmware_id = ? AND status IN (1, 2)", fw.ID). // 1=进行中 2=部分完成
		Count(&taskCount).Error; err != nil {
		return nil, err
	}

	// 如果有关联的进行中任务，检查修改的字段是否影响任务执行
	if taskCount > 0 {
		// 如果修改了 force_update 或 min_sys_version，可能影响任务执行
		if in.ForceUpdate != nil || in.MinSysVersion != "" {
			return nil, ErrFirmwareTaskConflict
		}
	}

	// 6. 数据更新 - 构建更新字段
	updates := make(map[string]interface{})
	updatedFields := make([]string, 0)

	// 可修改字段：version_description, device_models, force_update, min_sys_version, status, tags
	if in.VersionDesc != "" {
		updates["version_description"] = in.VersionDesc
		updatedFields = append(updatedFields, "version_description")
	}

	if in.DeviceModels != nil && len(in.DeviceModels) > 0 {
		modelsJSON, _ := json.Marshal(in.DeviceModels)
		updates["device_models"] = string(modelsJSON)
		updatedFields = append(updatedFields, "device_models")
	}

	if in.ForceUpdate != nil {
		updates["force_update"] = *in.ForceUpdate
		updatedFields = append(updatedFields, "force_update")
	}

	if in.MinSysVersion != "" {
		updates["min_sys_version"] = in.MinSysVersion
		updatedFields = append(updatedFields, "min_sys_version")
	}

	if in.Status != nil && (*in.Status == 1 || *in.Status == 2) {
		updates["status"] = *in.Status
		updatedFields = append(updatedFields, "status")
	}

	if in.Tags != nil {
		tagsJSON, _ := json.Marshal(in.Tags)
		updates["tags"] = string(tagsJSON)
		updatedFields = append(updatedFields, "tags")
	}

	if len(updatedFields) == 0 {
		return &FirmwareUpdateResponse{
			FirmwareID:    fw.ID,
			UpdatedFields: []string{},
			UpdatedAt:     fw.UpdatedAt.Format("2006-01-02 15:04:05"),
			Message:       "没有需要更新的字段",
			Success:       true,
		}, nil
	}

	// 更新 updated_at 和 operator
	updates["updated_at"] = time.Now()

	if err := e.Orm.Table("ota_firmware").Where("id = ?", fw.ID).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 7. 缓存清理 - 清除固件相关的缓存（如果有）
	// TODO: 清理 ota_firmware:{id} 缓存
	// TODO: 清理 ota_firmware_list:{product_key} 缓存

	// 8. 日志记录 - 记录修改操作
	// TODO: 记录到 ota_firmware_log 表或 system_operation_log 表
	// 包含：修改人、修改时间、修改前后字段值

	return &FirmwareUpdateResponse{
		FirmwareID:    fw.ID,
		UpdatedFields: updatedFields,
		UpdatedAt:     time.Now().Format("2006-01-02 15:04:05"),
		Message:       "固件信息更新成功",
		Success:       true,
	}, nil
}

// FirmwareHistoryRequest 版本历史查询请求
type FirmwareHistoryRequest struct {
	ProductKey    string
	DeviceModel   string
	DateFrom      string
	DateTo        string
	ReleaseType   string
	Status        string
	Page          int
	PageSize      int
	WithStats     bool
	WithChangeLog bool
}

// FirmwareHistoryItem 版本历史项
type FirmwareHistoryItem struct {
	Version          string  `json:"version"`
	VersionCode      int     `json:"version_code"`
	ReleaseDate      string  `json:"release_date"`
	ReleaseType      string  `json:"release_type"`
	ReleaseTypeText  string  `json:"release_type_text"`
	Status           string  `json:"status"`
	StatusText       string  `json:"status_text"`
	ForceUpdate      bool    `json:"force_update"`
	FileSize         int64   `json:"file_size"`
	FileSizeHuman    string  `json:"file_size_human"`
	DownloadCount    int64   `json:"download_count"`
	InstalledCount   int64   `json:"installed_count"`
	SuccessRate      float64 `json:"success_rate"`
	Description      string  `json:"description"`
	CreatedAt        string  `json:"created_at"`
	Creator          string  `json:"creator"`
	ChangeLog        string  `json:"change_log,omitempty"`
	IsCurrentVersion bool    `json:"is_current_version"`
	IsLatest         bool    `json:"is_latest"`
}

// FirmwareHistoryResponse 版本历史响应
type FirmwareHistoryResponse struct {
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
	List     []FirmwareHistoryItem `json:"list"`
	HasNext  bool                  `json:"has_next"`
	HasPrev  bool                  `json:"has_prev"`
}

// FirmwareHistory 版本历史查询
// 处理流程：
// 1. 参数解析，解析产品标识或设备型号作为主要筛选条件
// 2. 构建查询条件，附加时间范围筛选、版本状态筛选
// 3. 版本排序，按发布时间倒序排列
// 4. 版本标注，计算是否为最新版本、当前推荐版本等
// 5. 变更日志关联，生成变更日志内容
// 6. 统计数据补充，查询下载次数、安装设备数等
// 7. 返回格式化，时间戳转换、文件大小转换、枚举值转换
func (e *PlatformDeviceService) FirmwareHistory(in FirmwareHistoryRequest) (*FirmwareHistoryResponse, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}

	// 1. 参数解析，构建查询条件
	q := e.Orm.Table("ota_firmware AS f").Where("f.deleted_at IS NULL")

	// 按产品标识或设备型号筛选
	if pk := strings.TrimSpace(in.ProductKey); pk != "" {
		q = q.Where("f.product_key = ?", pk)
	}
	if dm := strings.TrimSpace(in.DeviceModel); dm != "" {
		// device_models 可能是 JSON 数组或逗号分隔字符串
		q = q.Where("f.device_models LIKE ?", "%"+dm+"%")
	}

	// 时间范围筛选
	if in.DateFrom != "" {
		if t, err := time.ParseInLocation("2006-01-02", in.DateFrom, time.Local); err == nil {
			q = q.Where("f.created_at >= ?", t)
		}
	}
	if in.DateTo != "" {
		if t, err := time.ParseInLocation("2006-01-02 23:59:59", in.DateTo+" 23:59:59", time.Local); err == nil {
			q = q.Where("f.created_at <= ?", t)
		}
	}

	// 版本状态筛选
	if in.Status != "" {
		switch in.Status {
		case "draft":
			q = q.Where("f.status = 0")
		case "testing":
			q = q.Where("f.status = 3")
		case "published":
			q = q.Where("f.status = 1")
		case "withdrawn":
			q = q.Where("f.status = 4")
		case "obsolete":
			q = q.Where("f.status = 5")
		}
	}

	// 发布类型筛选（如果有 release_type 字段）
	if in.ReleaseType != "" {
		// TODO: 如果数据库有 release_type 字段，添加筛选
		// q = q.Where("f.release_type = ?", in.ReleaseType)
	}

	// 2. 查询总数
	var total int64
	countQ := e.Orm.Table("ota_firmware AS f").Where("f.deleted_at IS NULL")
	if pk := strings.TrimSpace(in.ProductKey); pk != "" {
		countQ = countQ.Where("f.product_key = ?", pk)
	}
	if dm := strings.TrimSpace(in.DeviceModel); dm != "" {
		countQ = countQ.Where("f.device_models LIKE ?", "%"+dm+"%")
	}
	if in.DateFrom != "" {
		if t, err := time.ParseInLocation("2006-01-02", in.DateFrom, time.Local); err == nil {
			countQ = countQ.Where("f.created_at >= ?", t)
		}
	}
	if in.DateTo != "" {
		if t, err := time.ParseInLocation("2006-01-02 23:59:59", in.DateTo+" 23:59:59", time.Local); err == nil {
			countQ = countQ.Where("f.created_at <= ?", t)
		}
	}
	if in.Status != "" {
		switch in.Status {
		case "draft":
			countQ = countQ.Where("f.status = 0")
		case "testing":
			countQ = countQ.Where("f.status = 3")
		case "published":
			countQ = countQ.Where("f.status = 1")
		case "withdrawn":
			countQ = countQ.Where("f.status = 4")
		case "obsolete":
			countQ = countQ.Where("f.status = 5")
		}
	}
	if err := countQ.Count(&total).Error; err != nil {
		return nil, err
	}

	// 3. 分页和排序
	offset := (in.Page - 1) * in.PageSize
	q = q.Order("f.created_at DESC, f.version_code DESC")
	if in.PageSize > 0 {
		q = q.Limit(in.PageSize).Offset(offset)
	}

	// 4. 查询固件列表
	var firmwares []struct {
		ID            int64     `gorm:"column:id"`
		ProductKey    string    `gorm:"column:product_key"`
		Version       string    `gorm:"column:version"`
		VersionCode   int       `gorm:"column:version_code"`
		VersionDesc   string    `gorm:"column:version_description"`
		DeviceModels  string    `gorm:"column:device_models"`
		ForceUpdate   bool      `gorm:"column:force_update"`
		MinSysVersion string    `gorm:"column:min_sys_version"`
		Status        int16     `gorm:"column:status"`
		FileSize      int64     `gorm:"column:file_size"`
		FileMd5       string    `gorm:"column:file_md5"`
		DownloadCount int64     `gorm:"column:download_count"`
		CreatedAt     time.Time `gorm:"column:created_at"`
		UpdatedAt     time.Time `gorm:"column:updated_at"`
		CreateBy      int       `gorm:"column:create_by"`
	}
	if err := q.Select("id, product_key, version, version_code, version_description, device_models, force_update, min_sys_version, status, file_size, file_md5, download_count, created_at, updated_at, create_by").Find(&firmwares).Error; err != nil {
		return nil, err
	}

	// 5. 查询最新版本号（用于标注）
	var latestVersion struct {
		Version     string `gorm:"column:version"`
		VersionCode int    `gorm:"column:version_code"`
	}
	latestQ := e.Orm.Table("ota_firmware").Where("deleted_at IS NULL")
	if pk := strings.TrimSpace(in.ProductKey); pk != "" {
		latestQ = latestQ.Where("product_key = ?", pk)
	}
	if dm := strings.TrimSpace(in.DeviceModel); dm != "" {
		latestQ = latestQ.Where("device_models LIKE ?", "%"+dm+"%")
	}
	latestQ = latestQ.Order("version_code DESC").Limit(1)
	_ = latestQ.Select("version, version_code").Take(&latestVersion).Error

	// 6. 构建返回结果
	items := make([]FirmwareHistoryItem, 0, len(firmwares))
	for _, fw := range firmwares {
		item := FirmwareHistoryItem{
			Version:       fw.Version,
			VersionCode:   fw.VersionCode,
			ReleaseDate:   fw.CreatedAt.Format("2006-01-02"),
			ForceUpdate:   fw.ForceUpdate,
			FileSize:      fw.FileSize,
			FileSizeHuman: formatFileHuman(fw.FileSize),
			DownloadCount: fw.DownloadCount,
			Description:   fw.VersionDesc,
			CreatedAt:     fw.CreatedAt.Format("2006-01-02 15:04:05"),
			IsLatest:      fw.Version == latestVersion.Version,
		}

		// 版本类型标注
		item.ReleaseType = "formal"
		item.ReleaseTypeText = "正式版"
		if strings.Contains(strings.ToLower(fw.Version), "beta") || strings.Contains(strings.ToLower(fw.Version), "rc") {
			item.ReleaseType = "test"
			item.ReleaseTypeText = "测试版"
		}

		// 状态转换
		item.Status = firmwareStatus(fw.Status)
		item.StatusText = firmwareStatusText(fw.Status)

		// 6. 统计数据补充
		if in.WithStats {
			// 查询安装设备数
			var installedCount int64
			_ = e.Orm.Table("device").
				Where("firmware_version = ?", fw.Version).
				Count(&installedCount).Error
			item.InstalledCount = installedCount

			// 计算升级成功率（如果有升级任务记录）
			// TODO: 查询 ota_upgrade_task 计算成功率
			item.SuccessRate = 0
		}

		// 7. 变更日志
		if in.WithChangeLog {
			item.ChangeLog = fw.VersionDesc
			// TODO: 如果有独立的 change_log 表，关联查询
		}

		// 查询创建人
		if fw.CreateBy > 0 {
			var creator struct {
				Nickname string `gorm:"column:nickname"`
			}
			_ = e.Orm.Table("sys_user").
				Where("user_id = ?", fw.CreateBy).
				Select("nickname").
				Take(&creator).Error
			if creator.Nickname != "" {
				item.Creator = creator.Nickname
			}
		}

		items = append(items, item)
	}

	return &FirmwareHistoryResponse{
		Total:    total,
		Page:     in.Page,
		PageSize: in.PageSize,
		List:     items,
		HasNext:  in.Page*in.PageSize < int(total),
		HasPrev:  in.Page > 1,
	}, nil
}

func firmwareStatus(s int16) string {
	switch s {
	case 0:
		return "draft"
	case 1:
		return "published"
	case 2:
		return "disabled"
	case 3:
		return "testing"
	case 4:
		return "withdrawn"
	case 5:
		return "obsolete"
	default:
		return "unknown"
	}
}

// OTATaskDetailRequest OTA 任务详情查询请求
type OTATaskDetailRequest struct {
	TaskID      int64
	WithDevices bool
	WithLogs    bool
}

// OTATaskDeviceInfo OTA 任务设备信息
type OTATaskDeviceInfo struct {
	DeviceID    int64  `json:"device_id"`
	DeviceName  string `json:"device_name"`
	DeviceSn    string `json:"device_sn"`
	DeviceModel string `json:"device_model"`
	CurrentVer  string `json:"current_version"`
	TargetVer   string `json:"target_version"`
	Status      string `json:"status"`
	StatusText  string `json:"status_text"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Duration    int64  `json:"duration_seconds"`
	ErrorCode   string `json:"error_code,omitempty"`
	ErrorMsg    string `json:"error_msg,omitempty"`
	RetryCount  int    `json:"retry_count"`
	IsOnline    bool   `json:"is_online"`
}

// OTATaskLog OTA 任务操作日志
type OTATaskLog struct {
	LogID     int64  `json:"log_id"`
	TaskID    int64  `json:"task_id"`
	Operator  string `json:"operator"`
	Operation string `json:"operation"`
	Content   string `json:"content"`
	IpAddress string `json:"ip_address"`
	CreatedAt string `json:"created_at"`
}

// OTATaskDetailResponse OTA 任务详情响应
type OTATaskDetailResponse struct {
	// 基础信息
	TaskID        int64  `json:"task_id"`
	TaskName      string `json:"task_name"`
	TaskType      string `json:"task_type"`
	TaskTypeText  string `json:"task_type_text"`
	Status        string `json:"status"`
	StatusText    string `json:"status_text"`
	CreatedAt     string `json:"created_at"`
	Creator       string `json:"creator"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	TotalDuration int64  `json:"total_duration_seconds"`

	// 任务配置
	ProductKey     string `json:"product_key"`
	ProductName    string `json:"product_name"`
	TargetVersion  string `json:"target_version"`
	TargetCode     int    `json:"target_version_code"`
	FileURL        string `json:"file_url"`
	ForceUpdate    bool   `json:"force_update"`
	MaxRetry       int    `json:"max_retry"`
	TimeoutSeconds int    `json:"timeout_seconds"`

	// 目标范围
	TotalDevices   int64                  `json:"total_devices"`
	DeviceList     []OTATaskDeviceInfo    `json:"device_list,omitempty"`
	DeviceFilter   map[string]interface{} `json:"device_filter,omitempty"`
	ExcludeDevices []int64                `json:"exclude_devices,omitempty"`

	// 执行统计
	Pending     int64 `json:"pending"`
	Downloading int64 `json:"downloading"`
	Downloaded  int64 `json:"downloaded"`
	Upgrading   int64 `json:"upgrading"`
	Success     int64 `json:"success"`
	Failed      int64 `json:"failed"`
	Timeout     int64 `json:"timeout"`
	Cancelled   int64 `json:"cancelled"`

	// 进度信息
	Progress           float64 `json:"progress"`
	SuccessRate        float64 `json:"success_rate"`
	AvgDuration        float64 `json:"avg_duration_seconds"`
	EstimatedRemaining int64   `json:"estimated_remaining_seconds"`

	// 操作日志
	Logs []OTATaskLog `json:"logs,omitempty"`
}

// OTATaskDetail OTA 任务详情查询
// 处理流程：
// 1. 任务定位，根据任务 ID 从数据库查询任务主记录
// 2. 基础信息组装，读取任务创建时保存的配置信息
// 3. 执行数据汇总，实时统计各状态的设备数量
// 4. 目标设备列表获取，读取任务关联的设备列表或筛选条件
// 5. 结果详情获取，读取每台设备的升级结果记录
// 6. 日志信息关联，关联任务的创建日志、下发日志等操作记录
// 7. 汇总返回，组装完整的任务详情
func (e *PlatformDeviceService) OTATaskDetail(in OTATaskDetailRequest) (*OTATaskDetailResponse, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}

	// 1. 任务定位
	var task struct {
		ID             int64      `gorm:"column:id"`
		TaskName       string     `gorm:"column:task_name"`
		TaskType       string     `gorm:"column:task_type"`
		Status         int16      `gorm:"column:status"`
		ProductKey     string     `gorm:"column:product_key"`
		TargetVersion  string     `gorm:"column:target_version"`
		TargetCode     int        `gorm:"column:target_version_code"`
		FileURL        string     `gorm:"column:file_url"`
		ForceUpdate    bool       `gorm:"column:force_update"`
		MaxRetry       int        `gorm:"column:max_retry"`
		TimeoutSeconds int        `gorm:"column:timeout_seconds"`
		DeviceFilter   string     `gorm:"column:device_filter"`
		ExcludeDevices string     `gorm:"column:exclude_devices"`
		StartTime      *time.Time `gorm:"column:start_time"`
		EndTime        *time.Time `gorm:"column:end_time"`
		CreatedAt      time.Time  `gorm:"column:created_at"`
		CreateBy       int        `gorm:"column:create_by"`
	}

	if err := e.Orm.Table("ota_upgrade_task").
		Select("id, task_name, task_type, status, product_key, target_version, target_version_code, file_url, force_update, max_retry, timeout_seconds, device_filter, exclude_devices, start_time, end_time, created_at, create_by").
		Where("id = ?", in.TaskID).
		Take(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOTATaskNotFound
		}
		return nil, err
	}

	// 2. 基础信息组装
	response := &OTATaskDetailResponse{
		TaskID:         task.ID,
		TaskName:       task.TaskName,
		TaskType:       task.TaskType,
		TaskTypeText:   otaTaskTypeText(task.TaskType),
		Status:         otaTaskStatus(task.Status),
		StatusText:     otaTaskStatusText(task.Status),
		CreatedAt:      task.CreatedAt.Format("2006-01-02 15:04:05"),
		ProductKey:     task.ProductKey,
		TargetVersion:  task.TargetVersion,
		TargetCode:     task.TargetCode,
		FileURL:        task.FileURL,
		ForceUpdate:    task.ForceUpdate,
		MaxRetry:       task.MaxRetry,
		TimeoutSeconds: task.TimeoutSeconds,
	}

	// 计算总耗时
	if task.StartTime != nil {
		response.StartTime = task.StartTime.Format("2006-01-02 15:04:05")
		if task.EndTime != nil {
			response.EndTime = task.EndTime.Format("2006-01-02 15:04:05")
			response.TotalDuration = int64(task.EndTime.Sub(*task.StartTime).Seconds())
		} else {
			response.TotalDuration = int64(time.Since(*task.StartTime).Seconds())
		}
	}

	// 查询创建人
	if task.CreateBy > 0 {
		var creator struct {
			Nickname string `gorm:"column:nickname"`
		}
		_ = e.Orm.Table("sys_user").
			Where("user_id = ?", task.CreateBy).
			Select("nickname").
			Take(&creator).Error
		if creator.Nickname != "" {
			response.Creator = creator.Nickname
		}
	}

	// 查询产品名称
	if task.ProductKey != "" {
		var product struct {
			Name string `gorm:"column:product_name"`
		}
		_ = e.Orm.Table("ota_product").
			Where("product_key = ?", task.ProductKey).
			Select("product_name").
			Take(&product).Error
		response.ProductName = product.Name
	}

	// 解析设备筛选条件
	if task.DeviceFilter != "" {
		var filter map[string]interface{}
		if err := json.Unmarshal([]byte(task.DeviceFilter), &filter); err == nil {
			response.DeviceFilter = filter
		}
	}

	// 解析排除设备列表
	if task.ExcludeDevices != "" {
		var exclude []int64
		if err := json.Unmarshal([]byte(task.ExcludeDevices), &exclude); err == nil {
			response.ExcludeDevices = exclude
		}
	}

	// 3. 执行数据汇总 - 统计各状态设备数量
	statusCounts := make(map[string]int64)
	statusFields := []string{"pending", "downloading", "downloaded", "upgrading", "success", "failed", "timeout", "cancelled"}

	for _, status := range statusFields {
		var count int64
		_ = e.Orm.Table("ota_upgrade_device").
			Where("task_id = ? AND status = ?", task.ID, status).
			Count(&count).Error
		statusCounts[status] = count
	}

	response.Pending = statusCounts["pending"]
	response.Downloading = statusCounts["downloading"]
	response.Downloaded = statusCounts["downloaded"]
	response.Upgrading = statusCounts["upgrading"]
	response.Success = statusCounts["success"]
	response.Failed = statusCounts["failed"]
	response.Timeout = statusCounts["timeout"]
	response.Cancelled = statusCounts["cancelled"]

	// 4. 目标设备列表获取
	response.TotalDevices = response.Pending + response.Downloading + response.Downloaded + response.Upgrading + response.Success + response.Failed + response.Timeout + response.Cancelled

	// 5. 结果详情获取 - 获取设备列表（如果请求）
	if in.WithDevices {
		var devices []struct {
			DeviceID    int64      `gorm:"column:device_id"`
			DeviceName  string     `gorm:"column:device_name"`
			DeviceSn    string     `gorm:"column:device_sn"`
			DeviceModel string     `gorm:"column:device_model"`
			CurrentVer  string     `gorm:"column:current_version"`
			TargetVer   string     `gorm:"column:target_version"`
			Status      string     `gorm:"column:status"`
			StartTime   *time.Time `gorm:"column:start_time"`
			EndTime     *time.Time `gorm:"column:end_time"`
			ErrorCode   string     `gorm:"column:error_code"`
			ErrorMsg    string     `gorm:"column:error_msg"`
			RetryCount  int        `gorm:"column:retry_count"`
		}

		q := e.Orm.Table("ota_upgrade_device").
			Select("device_id, device_name, device_sn, device_model, current_version, target_version, status, start_time, end_time, error_code, error_msg, retry_count").
			Where("task_id = ?", task.ID).
			Limit(100) // 限制返回数量，避免过多

		if err := q.Find(&devices).Error; err != nil {
			return nil, err
		}

		deviceList := make([]OTATaskDeviceInfo, 0, len(devices))
		for _, dev := range devices {
			info := OTATaskDeviceInfo{
				DeviceID:    dev.DeviceID,
				DeviceName:  dev.DeviceName,
				DeviceSn:    dev.DeviceSn,
				DeviceModel: dev.DeviceModel,
				CurrentVer:  dev.CurrentVer,
				TargetVer:   dev.TargetVer,
				Status:      dev.Status,
				StatusText:  otaDeviceStatusText(dev.Status),
				RetryCount:  dev.RetryCount,
			}

			// 计算耗时
			if dev.StartTime != nil {
				info.StartTime = dev.StartTime.Format("2006-01-02 15:04:05")
				if dev.EndTime != nil {
					info.EndTime = dev.EndTime.Format("2006-01-02 15:04:05")
					info.Duration = int64(dev.EndTime.Sub(*dev.StartTime).Seconds())
				} else {
					info.Duration = int64(time.Since(*dev.StartTime).Seconds())
				}
			}

			// 错误信息
			if dev.ErrorCode != "" {
				info.ErrorCode = dev.ErrorCode
				info.ErrorMsg = dev.ErrorMsg
			}

			// 检查设备在线状态
			var deviceOnline struct {
				LastActiveAt *time.Time `gorm:"column:last_active_at"`
			}
			_ = e.Orm.Table("device").
				Where("id = ?", dev.DeviceID).
				Select("last_active_at").
				Take(&deviceOnline).Error
			if deviceOnline.LastActiveAt != nil && time.Since(*deviceOnline.LastActiveAt) <= 5*time.Minute {
				info.IsOnline = true
			}

			deviceList = append(deviceList, info)
		}
		response.DeviceList = deviceList
	}

	// 6. 日志信息关联
	if in.WithLogs {
		var logs []struct {
			ID        int64     `gorm:"column:id"`
			TaskID    int64     `gorm:"column:task_id"`
			Operator  string    `gorm:"column:operator"`
			Operation string    `gorm:"column:operation"`
			Content   string    `gorm:"column:content"`
			IpAddress string    `gorm:"column:ip_address"`
			CreatedAt time.Time `gorm:"column:created_at"`
		}

		if err := e.Orm.Table("ota_upgrade_task_log").
			Select("id, task_id, operator, operation, content, ip_address, created_at").
			Where("task_id = ?", task.ID).
			Order("created_at ASC").
			Limit(50). // 限制返回数量
			Find(&logs).Error; err != nil {
			return nil, err
		}

		taskLogs := make([]OTATaskLog, 0, len(logs))
		for _, log := range logs {
			taskLogs = append(taskLogs, OTATaskLog{
				LogID:     log.ID,
				TaskID:    log.TaskID,
				Operator:  log.Operator,
				Operation: log.Operation,
				Content:   log.Content,
				IpAddress: log.IpAddress,
				CreatedAt: log.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
		response.Logs = taskLogs
	}

	// 7. 进度信息计算
	if response.TotalDevices > 0 {
		completed := response.Success + response.Failed + response.Timeout + response.Cancelled
		response.Progress = float64(completed) / float64(response.TotalDevices) * 100

		if completed > 0 {
			response.SuccessRate = float64(response.Success) / float64(completed) * 100
		}

		// 计算平均耗时（仅成功的设备）
		if response.Success > 0 {
			var totalDuration int64
			_ = e.Orm.Table("ota_upgrade_device").
				Select("SUM(TIMESTAMPDIFF(SECOND, start_time, end_time))").
				Where("task_id = ? AND status = 'success'", task.ID).
				Scan(&totalDuration).Error
			if totalDuration > 0 {
				response.AvgDuration = float64(totalDuration) / float64(response.Success)
			}
		}

		// 估算剩余时间
		if response.AvgDuration > 0 && response.Pending > 0 {
			response.EstimatedRemaining = int64(response.AvgDuration) * response.Pending
		}
	}

	return response, nil
}

func otaTaskTypeText(taskType string) string {
	switch taskType {
	case "manual":
		return "手动创建"
	case "scheduled":
		return "定时创建"
	case "rule":
		return "规则触发"
	default:
		return taskType
	}
}

func otaTaskStatus(status int16) string {
	switch status {
	case 0:
		return "waiting"
	case 1:
		return "running"
	case 2:
		return "paused"
	case 3:
		return "completed"
	case 4:
		return "failed"
	case 5:
		return "cancelled"
	default:
		return "unknown"
	}
}

func otaTaskStatusText(status int16) string {
	switch status {
	case 0:
		return "等待中"
	case 1:
		return "执行中"
	case 2:
		return "执行暂停"
	case 3:
		return "已完成"
	case 4:
		return "执行失败"
	case 5:
		return "已取消"
	default:
		return "未知"
	}
}

func otaDeviceStatusText(status string) string {
	switch status {
	case "pending":
		return "待下发"
	case "downloading":
		return "下载中"
	case "downloaded":
		return "已下载"
	case "upgrading":
		return "升级中"
	case "success":
		return "成功"
	case "failed":
		return "失败"
	case "timeout":
		return "超时"
	case "cancelled":
		return "已取消"
	default:
		return status
	}
}

// OTATaskCancelRequest OTA 任务取消请求
type OTATaskCancelRequest struct {
	TaskID     int64
	Confirm    bool
	Reason     string
	CancelType string // all: 全部取消，pending_only: 仅取消待下发
	Operator   int64
}

// OTATaskCancelResponse OTA 任务取消响应
type OTATaskCancelResponse struct {
	Success         bool                   `json:"success"`
	TaskID          int64                  `json:"task_id"`
	TaskName        string                 `json:"task_name"`
	Status          string                 `json:"status"`
	AffectedDevices OTATaskAffectedDevices `json:"affected_devices"`
	CancelTime      string                 `json:"cancel_time"`
	Message         string                 `json:"message"`
}

// OTATaskAffectedDevices 受影响设备统计
type OTATaskAffectedDevices struct {
	Pending     int64 `json:"pending"`     // 待下发数
	Downloading int64 `json:"downloading"` // 下载中数
	Upgrading   int64 `json:"upgrading"`   // 升级中数
	Cancelled   int64 `json:"cancelled"`   // 已取消数
	Success     int64 `json:"success"`     // 已成功数
}

// OTATaskCancel OTA 任务取消
// 处理流程：
// 1. 任务定位，查询任务记录，校验任务状态
// 2. 影响评估，统计各状态设备数量
// 3. 设备指令撤回，向已下发指令的设备发送取消指令
// 4. 下载中断，向正在下载的设备发送中断指令
// 5. 缓存清理，清理设备固件缓存
// 6. 任务状态更新，更新为已取消状态
// 7. 设备状态清理，更新设备升级状态
// 8. 结果统计更新，计算最终结果
// 9. 通知推送，推送取消通知
// 10. 日志记录，记录完整操作
func (e *PlatformDeviceService) OTATaskCancel(in OTATaskCancelRequest) (*OTATaskCancelResponse, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}

	// 1. 任务定位
	var task struct {
		ID         int64      `gorm:"column:id"`
		TaskName   string     `gorm:"column:task_name"`
		Status     int16      `gorm:"column:status"`
		ProductKey string     `gorm:"column:product_key"`
		StartTime  *time.Time `gorm:"column:start_time"`
		EndTime    *time.Time `gorm:"column:end_time"`
	}

	if err := e.Orm.Table("ota_upgrade_task").
		Select("id, task_name, status, product_key, start_time, end_time").
		Where("id = ?", in.TaskID).
		Take(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOTATaskNotFound
		}
		return nil, err
	}

	// 校验任务状态：仅 waiting(0) 和 running(1) 可取消
	if task.Status == 3 { // completed
		return nil, ErrOTATaskAlreadyCompleted
	}
	if task.Status == 5 { // cancelled
		return nil, ErrOTATaskAlreadyCancelled
	}
	if task.Status != 0 && task.Status != 1 {
		return nil, fmt.Errorf("当前状态不可取消")
	}

	// 2. 影响评估 - 统计各状态设备数量
	affected := OTATaskAffectedDevices{}
	statusCounts := make(map[string]int64)
	statusFields := []string{"pending", "downloading", "downloaded", "upgrading", "success", "failed", "timeout", "cancelled"}

	for _, status := range statusFields {
		var count int64
		_ = e.Orm.Table("ota_upgrade_device").
			Where("task_id = ? AND status = ?", in.TaskID, status).
			Count(&count).Error
		statusCounts[status] = count
	}

	affected.Pending = statusCounts["pending"]
	affected.Downloading = statusCounts["downloading"]
	affected.Upgrading = statusCounts["upgrading"]
	affected.Cancelled = statusCounts["cancelled"]
	affected.Success = statusCounts["success"]

	// 3. 设备指令撤回 - 向 pending/downloading/upgrading 状态的设备发送取消指令
	cancelCount := int64(0)
	if in.CancelType == "" || in.CancelType == "all" {
		// 向待下发、下载中、升级中的设备发送取消指令
		targetStatuses := []string{"pending", "downloading", "upgrading"}

		for _, status := range targetStatuses {
			var devices []struct {
				DeviceID int64  `gorm:"column:device_id"`
				DeviceSn string `gorm:"column:device_sn"`
			}
			_ = e.Orm.Table("ota_upgrade_device").
				Select("device_id, device_sn").
				Where("task_id = ? AND status = ?", in.TaskID, status).
				Find(&devices).Error

			for _, dev := range devices {
				// 通过 MQTT 下发取消指令
				go e.sendOTACancelCommand(dev.DeviceSn, in.TaskID)
				cancelCount++
			}
		}
	} else if in.CancelType == "pending_only" {
		// 仅取消待下发的设备
		var devices []struct {
			DeviceID int64  `gorm:"column:device_id"`
			DeviceSn string `gorm:"column:device_sn"`
		}
		_ = e.Orm.Table("ota_upgrade_device").
			Select("device_id, device_sn").
			Where("task_id = ? AND status = 'pending'", in.TaskID).
			Find(&devices).Error

		for _, dev := range devices {
			go e.sendOTACancelCommand(dev.DeviceSn, in.TaskID)
			cancelCount++
		}
	}

	// 4. 下载中断 - 向 downloading 状态的设备发送中断指令
	// 已在步骤 3 中处理

	// 5. 缓存清理 - 向设备发送清理缓存指令
	// 可在设备端收到取消指令后自动清理，或单独下发清理指令

	// 6. 任务状态更新 - 更新为已取消状态
	now := time.Now()
	updateData := map[string]interface{}{
		"status":          5, // cancelled
		"end_time":        now,
		"cancel_time":     now,
		"cancel_operator": fmt.Sprintf("%d", in.Operator),
		"updated_at":      now,
	}
	if in.Reason != "" {
		updateData["cancel_reason"] = in.Reason
	}

	if err := e.Orm.Table("ota_upgrade_task").
		Where("id = ?", in.TaskID).
		Updates(updateData).Error; err != nil {
		return nil, ErrOTATaskCancelFailed
	}

	// 7. 设备状态清理 - 更新设备升级状态为已取消
	updateDeviceStatus := false
	if in.CancelType == "" || in.CancelType == "all" {
		updateDeviceStatus = true
	}

	if updateDeviceStatus {
		// 更新 pending/downloading 状态的设备为 cancelled
		_ = e.Orm.Table("ota_upgrade_device").
			Where("task_id = ? AND status IN ('pending', 'downloading')", in.TaskID).
			Updates(map[string]interface{}{
				"status":     "cancelled",
				"end_time":   now,
				"updated_at": now,
			}).Error
	}

	// 8. 结果统计更新 - 重新统计各状态数量
	finalCounts := make(map[string]int64)
	for _, status := range statusFields {
		var count int64
		_ = e.Orm.Table("ota_upgrade_device").
			Where("task_id = ? AND status = ?", in.TaskID, status).
			Count(&count).Error
		finalCounts[status] = count
	}

	affected.Cancelled = finalCounts["cancelled"]

	// 9. 通知推送 - 推送取消通知
	go e.sendOTATaskCancelNotification(task.ID, task.TaskName, in.Reason, affected)

	// 10. 日志记录 - 记录取消操作
	go e.recordOTATaskCancelLog(task.ID, in.Operator, in.Reason, affected)

	return &OTATaskCancelResponse{
		Success:         true,
		TaskID:          task.ID,
		TaskName:        task.TaskName,
		Status:          "cancelled",
		AffectedDevices: affected,
		CancelTime:      now.Format("2006-01-02 15:04:05"),
		Message:         fmt.Sprintf("任务取消成功，影响%d 个设备", cancelCount),
	}, nil
}

// sendOTACancelCommand 发送 OTA 取消指令
func (e *PlatformDeviceService) sendOTACancelCommand(deviceSn string, taskID int64) {
	// 通过 MQTT 下发取消指令
	mqttBody := map[string]interface{}{
		"command":   "cancel_upgrade",
		"task_id":   taskID,
		"timestamp": time.Now().Unix(),
	}
	_ = mqttBody // 避免未使用变量警告

	// TODO: 使用 MQTT 客户端发送
	// mqttJSON, _ := json.Marshal(mqttBody)
	// topic := fmt.Sprintf("device/%s/ota/cancel", deviceSn)
	// cli := mqttadmin.Client()
	// if cli != nil {
	// 	_ = cli.Publish(topic, 0, false, mqttJSON)
	// }

	// 记录下发日志
	_ = e.Orm.Create(&map[string]interface{}{
		"task_id":    taskID,
		"device_sn":  deviceSn,
		"operation":  "send_cancel_command",
		"content":    fmt.Sprintf("发送取消指令到设备：%s", deviceSn),
		"created_at": time.Now(),
	}).Error
}

// sendOTATaskCancelNotification 发送任务取消通知
func (e *PlatformDeviceService) sendOTATaskCancelNotification(taskID int64, taskName, reason string, affected OTATaskAffectedDevices) {
	// TODO: 实现通知推送逻辑
	// 可推送给任务创建人或管理员
	// 通知内容包含：任务名称、取消原因、影响统计
}

// recordOTATaskCancelLog 记录任务取消日志
func (e *PlatformDeviceService) recordOTATaskCancelLog(taskID int64, operator int64, reason string, affected OTATaskAffectedDevices) {
	logContent := map[string]interface{}{
		"operator":         operator,
		"reason":           reason,
		"affected_devices": affected,
		"operation":        "cancel_task",
	}
	logJSON, _ := json.Marshal(logContent)

	// 记录到任务日志表
	_ = e.Orm.Create(&map[string]interface{}{
		"task_id":    taskID,
		"operator":   fmt.Sprintf("%d", operator),
		"operation":  "cancel",
		"content":    string(logJSON),
		"created_at": time.Now(),
	}).Error
}

// OTATaskProgressRequest OTA 任务进度查询请求
type OTATaskProgressRequest struct {
	TaskID   int64 // 任务 ID
	DeviceID int64 // 设备 ID
	Refresh  bool  // 是否强制刷新
}

// OTATaskProgressResponse OTA 任务进度响应
type OTATaskProgressResponse struct {
	QueryType  string                 `json:"query_type"` // task: 任务维度，device: 设备维度
	TaskInfo   *OTATaskProgressInfo   `json:"task_info,omitempty"`
	DeviceInfo *OTADeviceProgressInfo `json:"device_info,omitempty"`
}

// OTATaskProgressInfo 任务进度信息
type OTATaskProgressInfo struct {
	TaskID             int64              `json:"task_id"`
	TaskName           string             `json:"task_name"`
	Status             string             `json:"status"`
	Progress           float64            `json:"progress"` // 完成百分比
	TotalDevices       int64              `json:"total_devices"`
	StateStatistics    OTAStateStatistics `json:"state_statistics"`
	SuccessRate        float64            `json:"success_rate"`
	AvgDuration        float64            `json:"avg_duration"`
	StartTime          string             `json:"start_time"`
	EstimatedRemaining int64              `json:"estimated_remaining"` // 预计剩余时间（秒）
	UpdatedAt          string             `json:"updated_at"`
}

// OTADeviceProgressInfo 设备进度信息
type OTADeviceProgressInfo struct {
	DeviceID    int64   `json:"device_id"`
	SN          string  `json:"sn"`
	TaskID      int64   `json:"task_id"`
	Status      string  `json:"status"`
	Progress    float64 `json:"progress"`
	CurrentStep string  `json:"current_step"`
	StepDetail  string  `json:"step_detail"`
	StartedAt   string  `json:"started_at"`
	CompletedAt string  `json:"completed_at"`
	Duration    int64   `json:"duration"`
	ErrorCode   string  `json:"error_code"`
	ErrorMsg    string  `json:"error_msg"`
	RetryCount  int     `json:"retry_count"`
	NextRetry   string  `json:"next_retry"`
}

// OTAStateStatistics 状态统计
type OTAStateStatistics struct {
	Pending     int64 `json:"pending"`
	Downloading int64 `json:"downloading"`
	Downloaded  int64 `json:"downloaded"`
	Upgrading   int64 `json:"upgrading"`
	Success     int64 `json:"success"`
	Failed      int64 `json:"failed"`
	Timeout     int64 `json:"timeout"`
	Cancelled   int64 `json:"cancelled"`
}

// OTATaskProgress OTA 任务进度查询
// 处理流程：
// 1. 任务定位，根据任务 ID 或设备 ID 定位任务记录
// 2. 状态统计，实时统计任务关联设备的各状态数量
// 3. 在线设备心跳，查询已下发指令但未响应的设备
// 4. 离线设备处理，查询设备离线前的最后状态
// 5. 预估计算，根据已完成设备的平均耗时预估完成时间
// 6. 缓存更新，将进度信息更新到缓存
// 7. 返回数据，根据查询维度组装对应数据
func (e *PlatformDeviceService) OTATaskProgress(in OTATaskProgressRequest) (*OTATaskProgressResponse, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}

	var taskID int64
	var deviceID int64

	// 1. 任务定位
	if in.TaskID > 0 {
		// 按任务 ID 查询
		taskID = in.TaskID
	} else if in.DeviceID > 0 {
		// 按设备 ID 查询任务
		var deviceTask struct {
			TaskID int64 `gorm:"column:task_id"`
		}
		if err := e.Orm.Table("ota_upgrade_device").
			Select("task_id").
			Where("device_id = ?", in.DeviceID).
			Order("created_at desc").
			Take(&deviceTask).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrPlatformDeviceNotFound
			}
			return nil, err
		}
		taskID = deviceTask.TaskID
		deviceID = in.DeviceID
	}

	if taskID <= 0 {
		return nil, ErrOTATaskNotFound
	}

	// 查询任务基本信息
	var task struct {
		ID        int64      `gorm:"column:id"`
		TaskName  string     `gorm:"column:task_name"`
		Status    int16      `gorm:"column:status"`
		StartTime *time.Time `gorm:"column:start_time"`
		EndTime   *time.Time `gorm:"column:end_time"`
	}

	if err := e.Orm.Table("ota_upgrade_task").
		Select("id, task_name, status, start_time, end_time").
		Where("id = ?", taskID).
		Take(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOTATaskNotFound
		}
		return nil, err
	}

	// 2. 状态统计 - 实时统计各状态设备数量
	stats := OTAStateStatistics{}
	statusFields := []string{"pending", "downloading", "downloaded", "upgrading", "success", "failed", "timeout", "cancelled"}

	for _, status := range statusFields {
		var count int64
		_ = e.Orm.Table("ota_upgrade_device").
			Where("task_id = ? AND status = ?", taskID, status).
			Count(&count).Error
		switch status {
		case "pending":
			stats.Pending = count
		case "downloading":
			stats.Downloading = count
		case "downloaded":
			stats.Downloaded = count
		case "upgrading":
			stats.Upgrading = count
		case "success":
			stats.Success = count
		case "failed":
			stats.Failed = count
		case "timeout":
			stats.Timeout = count
		case "cancelled":
			stats.Cancelled = count
		}
	}

	// 计算总数
	totalDevices := stats.Pending + stats.Downloading + stats.Downloaded + stats.Upgrading +
		stats.Success + stats.Failed + stats.Timeout + stats.Cancelled

	// 3. 在线设备心跳 - 查询已下发指令但未响应的设备
	// 这里可以检查设备最后心跳时间，判断是否超时
	// TODO: 实现设备心跳检查逻辑

	// 4. 离线设备处理 - 查询设备离线前的最后状态
	// TODO: 实现离线设备状态更新逻辑

	// 5. 预估计算 - 计算完成进度和预计剩余时间
	// 完成进度 = (success + failed + cancelled) / total * 100
	completedCount := stats.Success + stats.Failed + stats.Cancelled
	var progress float64
	if totalDevices > 0 {
		progress = float64(completedCount) / float64(totalDevices) * 100
	}

	// 计算成功率
	var successRate float64
	if stats.Success+stats.Failed > 0 {
		successRate = float64(stats.Success) / float64(stats.Success+stats.Failed) * 100
	}

	// 计算平均耗时
	var avgDuration float64
	var avgResult struct {
		AvgSeconds float64 `gorm:"column:avg_seconds"`
	}
	_ = e.Orm.Table("ota_upgrade_device").
		Select("AVESTAMP(TIMESTAMPDIFF(SECOND, started_at, end_time)) as avg_seconds").
		Where("task_id = ? AND status IN ('success', 'failed')", taskID).
		Take(&avgResult).Error
	avgDuration = avgResult.AvgSeconds

	// 预估剩余时间
	var estimatedRemaining int64
	if avgDuration > 0 && (stats.Pending+stats.Downloading+stats.Upgrading) > 0 {
		remainingDevices := stats.Pending + stats.Downloading + stats.Upgrading
		estimatedRemaining = int64(float64(remainingDevices) * avgDuration)
	}

	// 6. 缓存更新 - 将进度信息更新到缓存
	// TODO: 实现缓存更新逻辑

	// 7. 返回数据
	response := &OTATaskProgressResponse{
		QueryType: "task",
		TaskInfo: &OTATaskProgressInfo{
			TaskID:             task.ID,
			TaskName:           task.TaskName,
			Status:             otaTaskStatus(task.Status),
			Progress:           progress,
			TotalDevices:       totalDevices,
			StateStatistics:    stats,
			SuccessRate:        successRate,
			AvgDuration:        avgDuration,
			StartTime:          "",
			EstimatedRemaining: estimatedRemaining,
			UpdatedAt:          time.Now().Format("2006-01-02 15:04:05"),
		},
	}

	if task.StartTime != nil {
		response.TaskInfo.StartTime = task.StartTime.Format("2006-01-02 15:04:05")
	}

	// 如果是按设备 ID 查询，补充设备维度信息
	if deviceID > 0 {
		var device struct {
			DeviceID    int64      `gorm:"column:device_id"`
			DeviceSn    string     `gorm:"column:device_sn"`
			Status      string     `gorm:"column:status"`
			Progress    float64    `gorm:"column:progress"`
			CurrentStep string     `gorm:"column:current_step"`
			StepDetail  string     `gorm:"column:step_detail"`
			StartTime   *time.Time `gorm:"column:start_time"`
			EndTime     *time.Time `gorm:"column:end_time"`
			ErrorCode   string     `gorm:"column:error_code"`
			ErrorMsg    string     `gorm:"column:error_msg"`
			RetryCount  int        `gorm:"column:retry_count"`
			NextRetry   *time.Time `gorm:"column:next_retry"`
		}

		if err := e.Orm.Table("ota_upgrade_device").
			Select("device_id, device_sn, status, progress, current_step, step_detail, start_time, end_time, error_code, error_msg, retry_count, next_retry").
			Where("task_id = ? AND device_id = ?", taskID, deviceID).
			Take(&device).Error; err == nil {

			deviceInfo := &OTADeviceProgressInfo{
				DeviceID:    device.DeviceID,
				SN:          device.DeviceSn,
				TaskID:      taskID,
				Status:      device.Status,
				Progress:    device.Progress,
				CurrentStep: device.CurrentStep,
				StepDetail:  device.StepDetail,
				StartedAt:   "",
				CompletedAt: "",
				Duration:    0,
				ErrorCode:   device.ErrorCode,
				ErrorMsg:    device.ErrorMsg,
				RetryCount:  device.RetryCount,
				NextRetry:   "",
			}

			if device.StartTime != nil {
				deviceInfo.StartedAt = device.StartTime.Format("2006-01-02 15:04:05")
				if device.EndTime != nil {
					deviceInfo.CompletedAt = device.EndTime.Format("2006-01-02 15:04:05")
					deviceInfo.Duration = int64(device.EndTime.Sub(*device.StartTime).Seconds())
				} else {
					deviceInfo.Duration = int64(time.Since(*device.StartTime).Seconds())
				}
			}

			if device.NextRetry != nil {
				deviceInfo.NextRetry = device.NextRetry.Format("2006-01-02 15:04:05")
			}

			response.DeviceInfo = deviceInfo
		}
	}

	return response, nil
}

// OTATaskListRequest OTA 任务列表请求
type OTATaskListRequest struct {
	ProductKey     string
	TaskType       string
	Status         int16
	StartTimeBegin string
	StartTimeEnd   string
	Keyword        string
	Page           int
	PageSize       int
	SortBy         string
	SortOrder      string
}

// OTATaskListResponse OTA 任务列表响应
type OTATaskListResponse struct {
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	List     []OTATaskListItem `json:"list"`
}

// OTATaskListItem OTA 任务列表项
type OTATaskListItem struct {
	TaskID        int64   `json:"task_id"`
	TaskName      string  `json:"task_name"`
	TaskType      string  `json:"task_type"`
	TaskTypeText  string  `json:"task_type_text"`
	ProductKey    string  `json:"product_key"`
	ProductName   string  `json:"product_name"`
	TargetVersion string  `json:"target_version"`
	TotalDevices  int64   `json:"total_devices"`
	Pending       int64   `json:"pending"`
	Downloading   int64   `json:"downloading"`
	Success       int64   `json:"success"`
	Failed        int64   `json:"failed"`
	Progress      float64 `json:"progress"`
	Status        string  `json:"status"`
	StatusText    string  `json:"status_text"`
	ForceUpdate   bool    `json:"force_update"`
	CreatedAt     string  `json:"created_at"`
	Creator       string  `json:"creator"`
	StartTime     string  `json:"start_time"`
	EndTime       string  `json:"end_time"`
	CompletedAt   string  `json:"completed_at"`
	CancelTime    string  `json:"cancel_time"`
	Remark        string  `json:"remark"`
}

// OTATaskList OTA 任务列表查询
// 处理流程：
// 1. 参数解析，解析筛选条件和分页参数
// 2. 权限校验，根据操作人权限过滤可查看的产品线
// 3. 构建查询条件，组合筛选条件
// 4. 执行分页查询，统计总数和查询当前页数据
// 5. 实时进度统计，统计各状态设备数量
// 6. 关联产品信息，获取产品名称
// 7. 格式化返回，转换枚举值和时间格式
func (e *PlatformDeviceService) OTATaskList(in OTATaskListRequest) (*OTATaskListResponse, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}

	// 1. 参数解析
	page := in.Page
	if page <= 0 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	sortBy := in.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := in.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	// 2. 权限校验 - 根据操作人权限过滤可查看的产品线
	// TODO: 实现权限过滤逻辑，管理员可查看所有产品
	// 这里暂时不做过滤

	// 3. 构建查询条件
	db := e.Orm.Table("ota_upgrade_task").Select(`
		id, task_name, task_type, product_key, target_version, 
		force_update, status, created_at, create_by, 
		start_time, end_time, cancel_time, remark
	`)

	// 产品线筛选
	if in.ProductKey != "" {
		db = db.Where("product_key = ?", in.ProductKey)
	}

	// 任务类型筛选
	if in.TaskType != "" {
		db = db.Where("task_type = ?", in.TaskType)
	}

	// 任务状态筛选
	if in.Status != 0 {
		db = db.Where("status = ?", in.Status)
	}

	// 时间范围筛选
	if in.StartTimeBegin != "" {
		db = db.Where("start_time >= ?", in.StartTimeBegin)
	}
	if in.StartTimeEnd != "" {
		db = db.Where("start_time <= ?", in.StartTimeEnd)
	}

	// 关键词筛选
	if in.Keyword != "" {
		db = db.Where("task_name LIKE ? OR remark LIKE ?",
			"%"+in.Keyword+"%", "%"+in.Keyword+"%")
	}

	// 4. 执行分页查询
	// 统计总数
	var total int64
	countDB := db
	if err := countDB.Count(&total).Error; err != nil {
		return nil, err
	}

	// 排序
	orderExpr := fmt.Sprintf("%s %s", sortBy, sortOrder)
	db = db.Order(orderExpr)

	// 分页查询
	var tasks []struct {
		ID            int64      `gorm:"column:id"`
		TaskName      string     `gorm:"column:task_name"`
		TaskType      string     `gorm:"column:task_type"`
		ProductKey    string     `gorm:"column:product_key"`
		TargetVersion string     `gorm:"column:target_version"`
		ForceUpdate   bool       `gorm:"column:force_update"`
		Status        int16      `gorm:"column:status"`
		CreatedAt     time.Time  `gorm:"column:created_at"`
		CreateBy      int        `gorm:"column:create_by"`
		StartTime     *time.Time `gorm:"column:start_time"`
		EndTime       *time.Time `gorm:"column:end_time"`
		CancelTime    *time.Time `gorm:"column:cancel_time"`
		Remark        string     `gorm:"column:remark"`
	}

	if err := db.Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, err
	}

	// 5. 实时进度统计 & 6. 关联产品信息
	list := make([]OTATaskListItem, 0, len(tasks))
	for _, task := range tasks {
		// 统计各状态设备数量
		var pending, downloading, success, failed, totalDevices int64

		_ = e.Orm.Table("ota_upgrade_device").
			Where("task_id = ?", task.ID).
			Count(&totalDevices).Error

		_ = e.Orm.Table("ota_upgrade_device").
			Where("task_id = ? AND status = 'pending'", task.ID).
			Count(&pending).Error

		_ = e.Orm.Table("ota_upgrade_device").
			Where("task_id = ? AND status = 'downloading'", task.ID).
			Count(&downloading).Error

		_ = e.Orm.Table("ota_upgrade_device").
			Where("task_id = ? AND status = 'success'", task.ID).
			Count(&success).Error

		_ = e.Orm.Table("ota_upgrade_device").
			Where("task_id = ? AND status = 'failed'", task.ID).
			Count(&failed).Error

		// 计算完成进度
		var progress float64
		if totalDevices > 0 {
			progress = float64(success+failed) / float64(totalDevices) * 100
		}

		// 关联产品信息
		var productName string
		_ = e.Orm.Table("product").
			Select("product_name").
			Where("product_key = ?", task.ProductKey).
			Take(&productName).Error

		// 获取创建人信息
		var creator string
		_ = e.Orm.Table("sys_user").
			Select("username").
			Where("user_id = ?", task.CreateBy).
			Take(&creator).Error

		item := OTATaskListItem{
			TaskID:        task.ID,
			TaskName:      task.TaskName,
			TaskType:      task.TaskType,
			TaskTypeText:  otaTaskTypeText(task.TaskType),
			ProductKey:    task.ProductKey,
			ProductName:   productName,
			TargetVersion: task.TargetVersion,
			TotalDevices:  totalDevices,
			Pending:       pending,
			Downloading:   downloading,
			Success:       success,
			Failed:        failed,
			Progress:      progress,
			Status:        otaTaskStatus(task.Status),
			StatusText:    otaTaskStatusText(task.Status),
			ForceUpdate:   task.ForceUpdate,
			CreatedAt:     task.CreatedAt.Format("2006-01-02 15:04:05"),
			Creator:       creator,
			Remark:        task.Remark,
		}

		// 时间字段格式化
		if task.StartTime != nil {
			item.StartTime = task.StartTime.Format("2006-01-02 15:04:05")
		}
		if task.EndTime != nil {
			item.EndTime = task.EndTime.Format("2006-01-02 15:04:05")
			item.CompletedAt = item.EndTime
		}
		if task.CancelTime != nil {
			item.CancelTime = task.CancelTime.Format("2006-01-02 15:04:05")
		}

		list = append(list, item)
	}

	return &OTATaskListResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		List:     list,
	}, nil
}

// DeviceTask 设备定时任务管理
const (
	TaskTypeSingle = "single" // 单次任务
	TaskTypeCron   = "cron"   // Cron 重复任务
	TaskStatusDis  = 0        // 禁用
	TaskStatusAct  = 1        // 启用
)

// CreateTaskIn 创建任务输入
type CreateTaskIn struct {
	TaskName   string                 `json:"task_name"`
	TaskType   string                 `json:"task_type"`
	DeviceIds  []int64                `json:"device_ids"`
	ActionType string                 `json:"action_type"`
	ActionExec map[string]interface{} `json:"action_exec"`
	ExecTime   *time.Time             `json:"exec_time"`
	CronExpr   string                 `json:"cron_expr"`
	Timezone   string                 `json:"timezone"`
	Remark     string                 `json:"remark"`
	Operator   string                 `json:"operator"`
}

// CreateTaskOut 创建任务输出
type CreateTaskOut struct {
	TaskId   int    `json:"task_id"`
	TaskName string `json:"task_name"`
	Message  string `json:"message"`
}

// CreateTask 创建设备定时任务
func (e *PlatformDeviceService) CreateTask(in *CreateTaskIn) (*CreateTaskOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || strings.TrimSpace(in.TaskName) == "" {
		return nil, ErrPlatformDeviceInvalid
	}
	if len(in.DeviceIds) == 0 {
		return nil, fmt.Errorf("设备列表不能为空")
	}

	// 校验任务类型
	if in.TaskType != TaskTypeSingle && in.TaskType != TaskTypeCron {
		return nil, fmt.Errorf("任务类型必须为 single 或 cron")
	}

	// 校验执行时间或 Cron 表达式
	if in.TaskType == TaskTypeSingle && (in.ExecTime == nil || in.ExecTime.IsZero()) {
		return nil, fmt.Errorf("单次任务必须指定执行时间")
	}
	if in.TaskType == TaskTypeCron && strings.TrimSpace(in.CronExpr) == "" {
		return nil, fmt.Errorf("重复任务必须指定 Cron 表达式")
	}

	// 校验设备是否存在
	for _, deviceId := range in.DeviceIds {
		var devCount int64
		if err := e.Orm.Table("device").Where("id = ? AND deleted_at IS NULL", deviceId).Count(&devCount).Error; err != nil {
			return nil, err
		}
		if devCount == 0 {
			return nil, fmt.Errorf("设备 %d 不存在", deviceId)
		}
	}

	// 序列化设备 ID 列表
	deviceIdsStr := make([]string, len(in.DeviceIds))
	for i, id := range in.DeviceIds {
		deviceIdsStr[i] = strconv.FormatInt(id, 10)
	}

	// 序列化执行动作
	actionExecJSON, err := json.Marshal(in.ActionExec)
	if err != nil {
		return nil, fmt.Errorf("执行动作格式错误")
	}

	// 计算下次执行时间
	var nextExecAt *time.Time
	if in.TaskType == TaskTypeSingle {
		nextExecAt = in.ExecTime
	} else if in.TaskType == TaskTypeCron {
		// TODO: 使用 cron 库计算下次执行时间
		// 这里简化处理，暂不计算
	}

	// 创建任务
	task := models.DeviceTask{
		TaskName:    in.TaskName,
		TaskType:    in.TaskType,
		DeviceIds:   strings.Join(deviceIdsStr, ","),
		DeviceNames: "", // 可后续补充
		ActionType:  in.ActionType,
		ActionExec:  string(actionExecJSON),
		ExecTime:    in.ExecTime,
		CronExpr:    in.CronExpr,
		Timezone:    in.Timezone,
		Status:      TaskStatusAct,
		NextExecAt:  nextExecAt,
		Remark:      in.Remark,
		CreateBy:    0, // 从上下文获取
	}

	if err := e.Orm.Create(&task).Error; err != nil {
		return nil, err
	}

	// TODO: 同步到定时任务调度中心
	// scheduler.AddTask(task.Id, task.CronExpr, task.ExecTime)

	return &CreateTaskOut{
		TaskId:   task.Id,
		TaskName: task.TaskName,
		Message:  "任务创建成功",
	}, nil
}

// UpdateTaskIn 更新任务输入
type UpdateTaskIn struct {
	TaskId     int                    `json:"task_id"`
	TaskName   string                 `json:"task_name"`
	DeviceIds  []int64                `json:"device_ids"`
	ActionType string                 `json:"action_type"`
	ActionExec map[string]interface{} `json:"action_exec"`
	ExecTime   *time.Time             `json:"exec_time"`
	CronExpr   string                 `json:"cron_expr"`
	Timezone   string                 `json:"timezone"`
	Remark     string                 `json:"remark"`
	Operator   string                 `json:"operator"`
}

// UpdateTaskOut 更新任务输出
type UpdateTaskOut struct {
	TaskId  int    `json:"task_id"`
	Message string `json:"message"`
}

// UpdateTask 更新设备定时任务
func (e *PlatformDeviceService) UpdateTask(in *UpdateTaskIn) (*UpdateTaskOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.TaskId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询任务
	var task models.DeviceTask
	if err := e.Orm.Table("device_task").Where("id = ?", in.TaskId).Take(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, err
	}

	// 更新字段
	updateData := make(map[string]interface{})
	if strings.TrimSpace(in.TaskName) != "" {
		updateData["task_name"] = in.TaskName
	}
	if len(in.DeviceIds) > 0 {
		// 校验设备
		for _, deviceId := range in.DeviceIds {
			var devCount int64
			if err := e.Orm.Table("device").Where("id = ? AND deleted_at IS NULL", deviceId).Count(&devCount).Error; err != nil {
				return nil, err
			}
			if devCount == 0 {
				return nil, fmt.Errorf("设备 %d 不存在", deviceId)
			}
		}

		deviceIdsStr := make([]string, len(in.DeviceIds))
		for i, id := range in.DeviceIds {
			deviceIdsStr[i] = strconv.FormatInt(id, 10)
		}
		updateData["device_ids"] = strings.Join(deviceIdsStr, ",")
	}
	if in.ActionType != "" {
		updateData["action_type"] = in.ActionType
	}
	if in.ActionExec != nil {
		actionExecJSON, _ := json.Marshal(in.ActionExec)
		updateData["action_exec"] = string(actionExecJSON)
	}
	if in.ExecTime != nil {
		updateData["exec_time"] = in.ExecTime
	}
	if strings.TrimSpace(in.CronExpr) != "" {
		updateData["cron_expr"] = in.CronExpr
	}
	if strings.TrimSpace(in.Timezone) != "" {
		updateData["timezone"] = in.Timezone
	}
	updateData["remark"] = in.Remark
	updateData["updated_at"] = time.Now()

	if err := e.Orm.Model(&models.DeviceTask{}).Where("id = ?", in.TaskId).Updates(updateData).Error; err != nil {
		return nil, err
	}

	// TODO: 更新定时任务调度中心
	// scheduler.UpdateTask(task.Id, ...)

	return &UpdateTaskOut{
		TaskId:  in.TaskId,
		Message: "任务更新成功",
	}, nil
}

// DeleteTaskIn 删除任务输入
type DeleteTaskIn struct {
	TaskId   int    `json:"task_id"`
	Operator string `json:"operator"`
}

// DeleteTaskOut 删除任务输出
type DeleteTaskOut struct {
	TaskId  int    `json:"task_id"`
	Message string `json:"message"`
}

// DeleteTask 删除设备定时任务
func (e *PlatformDeviceService) DeleteTask(in *DeleteTaskIn) (*DeleteTaskOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.TaskId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询任务
	var task models.DeviceTask
	if err := e.Orm.Table("device_task").Where("id = ?", in.TaskId).Take(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, err
	}

	// 软删除
	if err := e.Orm.Delete(&models.DeviceTask{}, in.TaskId).Error; err != nil {
		return nil, err
	}

	// TODO: 从定时任务调度中心移除
	// scheduler.RemoveTask(task.Id)

	return &DeleteTaskOut{
		TaskId:  in.TaskId,
		Message: "任务删除成功",
	}, nil
}

// ToggleTaskIn 切换任务状态输入
type ToggleTaskIn struct {
	TaskId   int    `json:"task_id"`
	Status   int16  `json:"status"`
	Operator string `json:"operator"`
}

// ToggleTaskOut 切换任务状态输出
type ToggleTaskOut struct {
	TaskId  int    `json:"task_id"`
	Status  int16  `json:"status"`
	Message string `json:"message"`
}

// ToggleTask 切换设备定时任务状态
func (e *PlatformDeviceService) ToggleTask(in *ToggleTaskIn) (*ToggleTaskOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.TaskId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询任务
	var task models.DeviceTask
	if err := e.Orm.Table("device_task").Where("id = ?", in.TaskId).Take(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, err
	}

	// 更新状态
	if err := e.Orm.Model(&models.DeviceTask{}).Where("id = ?", in.TaskId).Updates(map[string]interface{}{
		"status":     in.Status,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return nil, err
	}

	// TODO: 同步到定时任务调度中心
	// if in.Status == TaskStatusAct {
	//     scheduler.EnableTask(task.Id)
	// } else {
	//     scheduler.DisableTask(task.Id)
	// }

	return &ToggleTaskOut{
		TaskId:  in.TaskId,
		Status:  in.Status,
		Message: "任务状态更新成功",
	}, nil
}

// GetTaskListIn 获取任务列表输入
type GetTaskListIn struct {
	TaskType string `json:"task_type"`
	Status   *int16 `json:"status"`
	Keyword  string `json:"keyword"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

// GetTaskListOut 获取任务列表输出
type GetTaskListOut struct {
	List  []models.DeviceTask `json:"list"`
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Size  int                 `json:"size"`
}

// GetTaskList 获取设备定时任务列表
func (e *PlatformDeviceService) GetTaskList(in *GetTaskListIn) (*GetTaskListOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}

	query := e.Orm.Table("device_task").Order("created_at DESC")

	if in.TaskType != "" {
		query = query.Where("task_type = ?", in.TaskType)
	}
	if in.Status != nil {
		query = query.Where("status = ?", *in.Status)
	}
	if strings.TrimSpace(in.Keyword) != "" {
		query = query.Where("task_name LIKE ?", "%"+in.Keyword+"%")
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	page := in.Page
	if page <= 0 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 查询列表
	var tasks []models.DeviceTask
	if err := query.Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, err
	}

	return &GetTaskListOut{
		List:  tasks,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// GetTaskDetailIn 获取任务详情输入
type GetTaskDetailIn struct {
	TaskId int `json:"task_id"`
}

// GetTaskDetailOut 获取任务详情输出
type GetTaskDetailOut struct {
	Task models.DeviceTask `json:"task"`
}

// GetTaskDetail 获取设备定时任务详情
func (e *PlatformDeviceService) GetTaskDetail(in *GetTaskDetailIn) (*GetTaskDetailOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.TaskId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	var task models.DeviceTask
	if err := e.Orm.Table("device_task").Where("id = ?", in.TaskId).Take(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, err
	}

	return &GetTaskDetailOut{
		Task: task,
	}, nil
}

// GetTaskExecLogIn 获取任务执行日志输入
type GetTaskExecLogIn struct {
	TaskId   int `json:"task_id"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// GetTaskExecLogOut 获取任务执行日志输出
type GetTaskExecLogOut struct {
	List  []models.DeviceTaskExecLog `json:"list"`
	Total int64                      `json:"total"`
	Page  int                        `json:"page"`
	Size  int                        `json:"size"`
}

// GetTaskExecLog 获取设备定时任务执行日志
func (e *PlatformDeviceService) GetTaskExecLog(in *GetTaskExecLogIn) (*GetTaskExecLogOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}

	query := e.Orm.Table("device_task_exec_log").Order("created_at DESC")

	if in.TaskId > 0 {
		query = query.Where("task_id = ?", in.TaskId)
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	page := in.Page
	if page <= 0 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 查询列表
	var logs []models.DeviceTaskExecLog
	if err := query.Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, err
	}

	return &GetTaskExecLogOut{
		List:  logs,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// ExecuteTask 执行定时任务（调度中心调用）
func (e *PlatformDeviceService) ExecuteTask(taskId int) error {
	if e.Orm == nil {
		return fmt.Errorf("orm nil")
	}

	// 查询任务
	var task models.DeviceTask
	if err := e.Orm.Table("device_task").Where("id = ? AND status = ?", taskId, TaskStatusAct).Take(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("任务不存在或已禁用")
		}
		return err
	}

	now := time.Now()
	startTime := time.Now()

	// 创建执行日志
	execLog := models.DeviceTaskExecLog{
		TaskId:     task.Id,
		TaskName:   task.TaskName,
		DeviceIds:  task.DeviceIds,
		ExecStatus: 0, // 执行中
	}

	if err := e.Orm.Create(&execLog).Error; err != nil {
		return err
	}

	// 异步执行任务
	go func() {
		var errMsg string
		var execStatus int16 = 1 // 执行成功
		var execResult string

		// 解析设备 ID 列表
		deviceIds := strings.Split(task.DeviceIds, ",")
		var successDevices []string
		var failedDevices []string

		// 解析执行动作
		var actionExec map[string]interface{}
		if task.ActionExec != "" {
			json.Unmarshal([]byte(task.ActionExec), &actionExec)
		}

		// 执行动作
		if task.ActionType == "device_control" {
			// 设备控制
			for _, deviceIdStr := range deviceIds {
				deviceId, _ := strconv.ParseInt(deviceIdStr, 10, 64)
				if deviceId <= 0 {
					failedDevices = append(failedDevices, deviceIdStr)
					continue
				}

				// 查询设备
				var dev struct {
					Sn string `gorm:"column:sn"`
				}
				if err := e.Orm.Table("device").Where("id = ?", deviceId).Select("sn").Take(&dev).Error; err != nil {
					failedDevices = append(failedDevices, deviceIdStr)
					continue
				}

				// 通过 MQTT 下发控制指令
				if cli := mqttadmin.Client(); cli != nil {
					command := map[string]interface{}{
						"cmd":       "task_control",
						"task_id":   task.Id,
						"action":    actionExec["action"],
						"params":    actionExec["params"],
						"timestamp": now.Unix(),
					}
					b, _ := json.Marshal(command)
					topic := fmt.Sprintf("device/%s/command", dev.Sn)
					if err := cli.Publish(topic, 1, false, b); err != nil {
						failedDevices = append(failedDevices, deviceIdStr)
					} else {
						successDevices = append(successDevices, deviceIdStr)
					}
				} else {
					failedDevices = append(failedDevices, deviceIdStr)
				}
			}
		} else if task.ActionType == "mqtt_command" {
			// MQTT 命令
			for _, deviceIdStr := range deviceIds {
				deviceId, _ := strconv.ParseInt(deviceIdStr, 10, 64)
				if deviceId <= 0 {
					failedDevices = append(failedDevices, deviceIdStr)
					continue
				}

				var dev struct {
					Sn string `gorm:"column:sn"`
				}
				if err := e.Orm.Table("device").Where("id = ?", deviceId).Select("sn").Take(&dev).Error; err != nil {
					failedDevices = append(failedDevices, deviceIdStr)
					continue
				}

				// 通过 MQTT 下发自定义命令
				if cli := mqttadmin.Client(); cli != nil {
					command := map[string]interface{}{
						"cmd":       actionExec["cmd"],
						"task_id":   task.Id,
						"params":    actionExec["params"],
						"timestamp": now.Unix(),
					}
					b, _ := json.Marshal(command)
					topic := fmt.Sprintf("device/%s/command", dev.Sn)
					if err := cli.Publish(topic, 1, false, b); err != nil {
						failedDevices = append(failedDevices, deviceIdStr)
					} else {
						successDevices = append(successDevices, deviceIdStr)
					}
				} else {
					failedDevices = append(failedDevices, deviceIdStr)
				}
			}
		}

		// 构建执行结果
		if len(failedDevices) > 0 {
			execStatus = 2 // 执行失败
			errMsg = fmt.Sprintf("成功 %d 个设备，失败 %d 个设备", len(successDevices), len(failedDevices))
		}
		execResult = fmt.Sprintf("成功设备：%s, 失败设备：%s", strings.Join(successDevices, ","), strings.Join(failedDevices, ","))

		// 计算耗时
		duration := int(time.Since(startTime).Milliseconds())

		// 更新执行日志
		updateData := map[string]interface{}{
			"exec_status": execStatus,
			"exec_result": execResult,
			"duration":    duration,
		}
		if errMsg != "" {
			updateData["err_msg"] = errMsg
		}
		e.Orm.Model(&models.DeviceTaskExecLog{}).Where("id = ?", execLog.Id).Updates(updateData)

		// 更新任务执行统计
		e.Orm.Model(&models.DeviceTask{}).Where("id = ?", task.Id).Updates(map[string]interface{}{
			"exec_count":   gorm.Expr("exec_count + 1"),
			"last_exec_at": now,
			"updated_at":   now,
		})
	}()

	return nil
}

// DeviceScene 设备场景联动管理
const (
	TriggerTypeDeviceStatus = "device_status" // 设备状态触发
	TriggerTypeTimer        = "timer"         // 定时触发
	TriggerTypeManual       = "manual"        // 手动触发

	ActionTypeDeviceControl = "device_control" // 设备控制
	ActionTypeMQTTCommand   = "mqtt_command"   // MQTT 命令
	ActionTypeSceneTrigger  = "scene_trigger"  // 场景触发
)

// CreateSceneIn 创建场景输入
type CreateSceneIn struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	TriggerType string                 `json:"trigger_type"` // device_status/timer/manual
	TriggerCond map[string]interface{} `json:"trigger_cond"` // 触发条件
	ActionType  string                 `json:"action_type"`  // device_control/mqtt_command/scene_trigger
	ActionExec  map[string]interface{} `json:"action_exec"`  // 执行动作
	DeviceIds   []int64                `json:"device_ids"`
	Operator    string                 `json:"operator"`
}

// CreateSceneOut 创建场景输出
type CreateSceneOut struct {
	SceneId int    `json:"scene_id"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// CreateScene 创建场景联动规则
func (e *PlatformDeviceService) CreateScene(in *CreateSceneIn) (*CreateSceneOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("场景名称不能为空")
	}
	if in.TriggerType == "" {
		return nil, fmt.Errorf("触发类型不能为空")
	}
	if in.ActionType == "" {
		return nil, fmt.Errorf("执行动作类型不能为空")
	}
	if len(in.DeviceIds) == 0 {
		return nil, fmt.Errorf("至少关联一个设备")
	}

	// 校验触发类型
	validTriggerTypes := map[string]bool{
		TriggerTypeDeviceStatus: true,
		TriggerTypeTimer:        true,
		TriggerTypeManual:       true,
	}
	if !validTriggerTypes[in.TriggerType] {
		return nil, fmt.Errorf("无效的触发类型：%s", in.TriggerType)
	}

	// 校验动作类型
	validActionTypes := map[string]bool{
		ActionTypeDeviceControl: true,
		ActionTypeMQTTCommand:   true,
		ActionTypeSceneTrigger:  true,
	}
	if !validActionTypes[in.ActionType] {
		return nil, fmt.Errorf("无效的执行动作类型：%s", in.ActionType)
	}

	// 校验设备存在且在线
	for _, deviceId := range in.DeviceIds {
		var devCount int64
		if err := e.Orm.Table("device").Where("id = ? AND status = 1", deviceId).Count(&devCount).Error; err != nil {
			return nil, err
		}
		if devCount == 0 {
			return nil, fmt.Errorf("设备 %d 不存在或已禁用", deviceId)
		}
	}

	// 序列化触发条件和执行动作
	triggerCondJSON, _ := json.Marshal(in.TriggerCond)
	actionExecJSON, _ := json.Marshal(in.ActionExec)

	// 序列化设备 ID 列表
	deviceIdsStr := ""
	for i, id := range in.DeviceIds {
		if i > 0 {
			deviceIdsStr += ","
		}
		deviceIdsStr += fmt.Sprintf("%d", id)
	}

	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "admin"
	}

	// 创建场景
	scene := models.DeviceScene{
		Name:        strings.TrimSpace(in.Name),
		Description: strings.TrimSpace(in.Description),
		TriggerType: in.TriggerType,
		TriggerCond: string(triggerCondJSON),
		ActionType:  in.ActionType,
		ActionExec:  string(actionExecJSON),
		DeviceIds:   deviceIdsStr,
		Status:      1,
		ExecCount:   0,
		CreateBy:    0,
		UpdateBy:    0,
	}

	if err := e.Orm.Create(&scene).Error; err != nil {
		return nil, err
	}

	return &CreateSceneOut{
		SceneId: scene.Id,
		Name:    scene.Name,
		Message: "场景创建成功",
	}, nil
}

// UpdateSceneIn 更新场景输入
type UpdateSceneIn struct {
	SceneId     int                    `json:"scene_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	TriggerType string                 `json:"trigger_type"`
	TriggerCond map[string]interface{} `json:"trigger_cond"`
	ActionType  string                 `json:"action_type"`
	ActionExec  map[string]interface{} `json:"action_exec"`
	DeviceIds   []int64                `json:"device_ids"`
	Status      *int16                 `json:"status"`
	Operator    string                 `json:"operator"`
}

// UpdateSceneOut 更新场景输出
type UpdateSceneOut struct {
	SceneId int    `json:"scene_id"`
	Message string `json:"message"`
}

// UpdateScene 更新场景联动规则
func (e *PlatformDeviceService) UpdateScene(in *UpdateSceneIn) (*UpdateSceneOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.SceneId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询场景
	var scene models.DeviceScene
	if err := e.Orm.Table("device_scene").Where("id = ?", in.SceneId).Take(&scene).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("场景不存在")
		}
		return nil, err
	}

	// 构建更新数据
	updateData := make(map[string]interface{})
	if strings.TrimSpace(in.Name) != "" {
		updateData["name"] = strings.TrimSpace(in.Name)
	}
	if in.Description != "" {
		updateData["description"] = strings.TrimSpace(in.Description)
	}
	if in.TriggerType != "" {
		validTriggerTypes := map[string]bool{
			TriggerTypeDeviceStatus: true,
			TriggerTypeTimer:        true,
			TriggerTypeManual:       true,
		}
		if !validTriggerTypes[in.TriggerType] {
			return nil, fmt.Errorf("无效的触发类型：%s", in.TriggerType)
		}
		updateData["trigger_type"] = in.TriggerType
	}
	if in.TriggerCond != nil {
		triggerCondJSON, _ := json.Marshal(in.TriggerCond)
		updateData["trigger_cond"] = string(triggerCondJSON)
	}
	if in.ActionType != "" {
		validActionTypes := map[string]bool{
			ActionTypeDeviceControl: true,
			ActionTypeMQTTCommand:   true,
			ActionTypeSceneTrigger:  true,
		}
		if !validActionTypes[in.ActionType] {
			return nil, fmt.Errorf("无效的执行动作类型：%s", in.ActionType)
		}
		updateData["action_type"] = in.ActionType
	}
	if in.ActionExec != nil {
		actionExecJSON, _ := json.Marshal(in.ActionExec)
		updateData["action_exec"] = string(actionExecJSON)
	}
	if in.DeviceIds != nil && len(in.DeviceIds) > 0 {
		deviceIdsStr := ""
		for i, id := range in.DeviceIds {
			if i > 0 {
				deviceIdsStr += ","
			}
			deviceIdsStr += fmt.Sprintf("%d", id)
		}
		updateData["device_ids"] = deviceIdsStr
	}
	if in.Status != nil && (*in.Status == 0 || *in.Status == 1) {
		updateData["status"] = *in.Status
	}
	updateData["updated_at"] = time.Now()

	if err := e.Orm.Model(&models.DeviceScene{}).Where("id = ?", in.SceneId).Updates(updateData).Error; err != nil {
		return nil, err
	}

	return &UpdateSceneOut{
		SceneId: in.SceneId,
		Message: "场景更新成功",
	}, nil
}

// DeleteSceneIn 删除场景输入
type DeleteSceneIn struct {
	SceneId  int    `json:"scene_id"`
	Operator string `json:"operator"`
}

// DeleteSceneOut 删除场景输出
type DeleteSceneOut struct {
	SceneId int    `json:"scene_id"`
	Message string `json:"message"`
}

// DeleteScene 删除场景联动规则
func (e *PlatformDeviceService) DeleteScene(in *DeleteSceneIn) (*DeleteSceneOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.SceneId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询场景
	var scene models.DeviceScene
	if err := e.Orm.Table("device_scene").Where("id = ?", in.SceneId).Take(&scene).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("场景不存在")
		}
		return nil, err
	}

	now := time.Now()
	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "admin"
	}

	err := e.Orm.Transaction(func(tx *gorm.DB) error {
		// 删除场景
		if err := tx.Delete(&models.DeviceScene{}, in.SceneId).Error; err != nil {
			return err
		}

		// 记录操作日志
		log := map[string]interface{}{
			"scene_id":     in.SceneId,
			"scene_name":   scene.Name,
			"operator":     op,
			"operate_time": now,
			"action":       "delete_scene",
		}
		logJSON, _ := json.Marshal(log)
		return tx.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator) VALUES (0, '', 'admin_delete_scene', ?, ?)`, string(logJSON), op).Error
	})

	if err != nil {
		return nil, err
	}

	return &DeleteSceneOut{
		SceneId: in.SceneId,
		Message: "场景删除成功",
	}, nil
}

// GetSceneListIn 获取场景列表输入
type GetSceneListIn struct {
	TriggerType string `json:"trigger_type"`
	Status      *int16 `json:"status"`
	Keyword     string `json:"keyword"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
}

// GetSceneListOut 获取场景列表输出
type GetSceneListOut struct {
	List  []models.DeviceScene `json:"list"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	Size  int                  `json:"size"`
}

// GetSceneList 获取场景联动规则列表
func (e *PlatformDeviceService) GetSceneList(in *GetSceneListIn) (*GetSceneListOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}

	query := e.Orm.Table("device_scene").Order("created_at DESC")

	if in.TriggerType != "" {
		query = query.Where("trigger_type = ?", in.TriggerType)
	}

	if in.Status != nil {
		query = query.Where("status = ?", *in.Status)
	}

	if strings.TrimSpace(in.Keyword) != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+strings.TrimSpace(in.Keyword)+"%", "%"+strings.TrimSpace(in.Keyword)+"%")
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	page := in.Page
	if page <= 0 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 查询列表
	var list []models.DeviceScene
	if err := query.Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}

	return &GetSceneListOut{
		List:  list,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// GetSceneDetailIn 获取场景详情输入
type GetSceneDetailIn struct {
	SceneId int `json:"scene_id"`
}

// GetSceneDetailOut 获取场景详情输出
type GetSceneDetailOut struct {
	Scene       models.DeviceScene       `json:"scene"`
	Devices     []map[string]interface{} `json:"devices"`
	TriggerCond map[string]interface{}   `json:"trigger_cond"`
	ActionExec  map[string]interface{}   `json:"action_exec"`
}

// GetSceneDetail 获取场景详情
func (e *PlatformDeviceService) GetSceneDetail(in *GetSceneDetailIn) (*GetSceneDetailOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.SceneId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询场景
	var scene models.DeviceScene
	if err := e.Orm.Table("device_scene").Where("id = ?", in.SceneId).Take(&scene).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("场景不存在")
		}
		return nil, err
	}

	// 解析触发条件和执行动作
	var triggerCond map[string]interface{}
	var actionExec map[string]interface{}
	if scene.TriggerCond != "" {
		json.Unmarshal([]byte(scene.TriggerCond), &triggerCond)
	}
	if scene.ActionExec != "" {
		json.Unmarshal([]byte(scene.ActionExec), &actionExec)
	}

	// 查询关联设备
	var devices []map[string]interface{}
	if scene.DeviceIds != "" {
		deviceIds := strings.Split(scene.DeviceIds, ",")
		var deviceList []struct {
			ID           int64      `gorm:"column:id"`
			Sn           string     `gorm:"column:sn"`
			Model        string     `gorm:"column:model"`
			OnlineStatus int16      `gorm:"column:online_status"`
			Status       int16      `gorm:"column:status"`
			LastActiveAt *time.Time `gorm:"column:last_active_at"`
		}
		if err := e.Orm.Table("device").Select("id, sn, model, online_status, status, last_active_at").Where("id IN ?", deviceIds).Find(&deviceList).Error; err != nil {
			return nil, err
		}

		for _, dev := range deviceList {
			devices = append(devices, map[string]interface{}{
				"id":             dev.ID,
				"sn":             dev.Sn,
				"model":          dev.Model,
				"online_status":  dev.OnlineStatus,
				"status":         dev.Status,
				"last_active_at": dev.LastActiveAt,
			})
		}
	}

	return &GetSceneDetailOut{
		Scene:       scene,
		Devices:     devices,
		TriggerCond: triggerCond,
		ActionExec:  actionExec,
	}, nil
}

// ExecuteSceneIn 执行场景输入
type ExecuteSceneIn struct {
	SceneId     int                    `json:"scene_id"`
	TriggerSrc  string                 `json:"trigger_src"` // device/timer/manual/api
	TriggerData map[string]interface{} `json:"trigger_data"`
}

// ExecuteSceneOut 执行场景输出
type ExecuteSceneOut struct {
	SceneId   int    `json:"scene_id"`
	ExecLogId int    `json:"exec_log_id"`
	Message   string `json:"message"`
}

// ExecuteScene 执行场景联动
func (e *PlatformDeviceService) ExecuteScene(in *ExecuteSceneIn) (*ExecuteSceneOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.SceneId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询场景
	var scene models.DeviceScene
	if err := e.Orm.Table("device_scene").Where("id = ? AND status = 1", in.SceneId).Take(&scene).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("场景不存在或已禁用")
		}
		return nil, err
	}

	now := time.Now()
	startTime := time.Now()

	// 创建执行日志
	execLog := models.DeviceSceneExecLog{
		SceneId:    scene.Id,
		SceneName:  scene.Name,
		TriggerSrc: in.TriggerSrc,
		ExecStatus: 0, // 执行中
	}
	if in.TriggerData != nil {
		triggerDataJSON, _ := json.Marshal(in.TriggerData)
		execLog.TriggerData = string(triggerDataJSON)
	}

	if err := e.Orm.Create(&execLog).Error; err != nil {
		return nil, err
	}

	// 异步执行场景动作
	go func() {
		var errMsg string
		var execStatus int16 = 1 // 执行成功
		var execResult string

		// 解析设备 ID 列表
		deviceIds := strings.Split(scene.DeviceIds, ",")
		var successDevices []string
		var failedDevices []string

		// 解析执行动作
		var actionExec map[string]interface{}
		if scene.ActionExec != "" {
			json.Unmarshal([]byte(scene.ActionExec), &actionExec)
		}

		// 执行动作
		if scene.ActionType == ActionTypeDeviceControl {
			// 设备控制
			for _, deviceIdStr := range deviceIds {
				deviceId, _ := strconv.ParseInt(deviceIdStr, 10, 64)
				if deviceId <= 0 {
					failedDevices = append(failedDevices, deviceIdStr)
					continue
				}

				// 查询设备
				var dev struct {
					Sn string `gorm:"column:sn"`
				}
				if err := e.Orm.Table("device").Where("id = ?", deviceId).Select("sn").Take(&dev).Error; err != nil {
					failedDevices = append(failedDevices, deviceIdStr)
					continue
				}

				// 通过 MQTT 下发控制指令
				if cli := mqttadmin.Client(); cli != nil {
					command := map[string]interface{}{
						"cmd":       "scene_control",
						"scene_id":  scene.Id,
						"action":    actionExec["action"],
						"params":    actionExec["params"],
						"timestamp": now.Unix(),
					}
					b, _ := json.Marshal(command)
					topic := fmt.Sprintf("device/%s/command", dev.Sn)
					if err := cli.Publish(topic, 1, false, b); err != nil {
						failedDevices = append(failedDevices, deviceIdStr)
					} else {
						successDevices = append(successDevices, deviceIdStr)
					}
				} else {
					failedDevices = append(failedDevices, deviceIdStr)
				}
			}
		} else if scene.ActionType == ActionTypeMQTTCommand {
			// MQTT 命令
			for _, deviceIdStr := range deviceIds {
				deviceId, _ := strconv.ParseInt(deviceIdStr, 10, 64)
				if deviceId <= 0 {
					failedDevices = append(failedDevices, deviceIdStr)
					continue
				}

				var dev struct {
					Sn string `gorm:"column:sn"`
				}
				if err := e.Orm.Table("device").Where("id = ?", deviceId).Select("sn").Take(&dev).Error; err != nil {
					failedDevices = append(failedDevices, deviceIdStr)
					continue
				}

				// 通过 MQTT 下发自定义命令
				if cli := mqttadmin.Client(); cli != nil {
					command := map[string]interface{}{
						"cmd":       actionExec["cmd"],
						"scene_id":  scene.Id,
						"params":    actionExec["params"],
						"timestamp": now.Unix(),
					}
					b, _ := json.Marshal(command)
					topic := fmt.Sprintf("device/%s/command", dev.Sn)
					if err := cli.Publish(topic, 1, false, b); err != nil {
						failedDevices = append(failedDevices, deviceIdStr)
					} else {
						successDevices = append(successDevices, deviceIdStr)
					}
				} else {
					failedDevices = append(failedDevices, deviceIdStr)
				}
			}
		}

		// 构建执行结果
		if len(failedDevices) > 0 {
			execStatus = 2 // 执行失败
			errMsg = fmt.Sprintf("成功 %d 个设备，失败 %d 个设备", len(successDevices), len(failedDevices))
		}
		execResult = fmt.Sprintf("成功设备：%s, 失败设备：%s", strings.Join(successDevices, ","), strings.Join(failedDevices, ","))

		// 计算耗时
		duration := int(time.Since(startTime).Milliseconds())

		// 更新执行日志
		updateData := map[string]interface{}{
			"exec_status": execStatus,
			"exec_result": execResult,
			"devices":     scene.DeviceIds,
			"duration":    duration,
		}
		if errMsg != "" {
			updateData["err_msg"] = errMsg
		}
		e.Orm.Model(&models.DeviceSceneExecLog{}).Where("id = ?", execLog.Id).Updates(updateData)

		// 更新场景执行统计
		e.Orm.Model(&models.DeviceScene{}).Where("id = ?", scene.Id).Updates(map[string]interface{}{
			"exec_count":   gorm.Expr("exec_count + 1"),
			"last_exec_at": now,
			"updated_at":   now,
		})
	}()

	return &ExecuteSceneOut{
		SceneId:   scene.Id,
		ExecLogId: execLog.Id,
		Message:   "场景执行中",
	}, nil
}

// GetSceneExecLogIn 获取场景执行日志输入
type GetSceneExecLogIn struct {
	SceneId   int       `json:"scene_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Page      int       `json:"page"`
	PageSize  int       `json:"page_size"`
}

// GetSceneExecLogOut 获取场景执行日志输出
type GetSceneExecLogOut struct {
	List  []models.DeviceSceneExecLog `json:"list"`
	Total int64                       `json:"total"`
	Page  int                         `json:"page"`
	Size  int                         `json:"size"`
}

// GetSceneExecLog 获取场景执行日志
func (e *PlatformDeviceService) GetSceneExecLog(in *GetSceneExecLogIn) (*GetSceneExecLogOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}

	query := e.Orm.Table("device_scene_exec_log").Where("scene_id = ?", in.SceneId)

	if !in.StartTime.IsZero() {
		query = query.Where("created_at >= ?", in.StartTime)
	}
	if !in.EndTime.IsZero() {
		query = query.Where("created_at <= ?", in.EndTime)
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	page := in.Page
	if page <= 0 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 查询列表
	var list []models.DeviceSceneExecLog
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}

	return &GetSceneExecLogOut{
		List:  list,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// CheckSceneTrigger 检查设备状态是否触发场景（供设备上报时调用）
func (e *PlatformDeviceService) CheckSceneTrigger(deviceId int64, deviceStatus map[string]interface{}) error {
	if e.Orm == nil {
		return fmt.Errorf("orm nil")
	}

	// 查询所有启用的设备状态触发场景
	var scenes []models.DeviceScene
	if err := e.Orm.Table("device_scene").
		Where("status = 1 AND trigger_type = ?", TriggerTypeDeviceStatus).
		Find(&scenes).Error; err != nil {
		return err
	}

	for _, scene := range scenes {
		// 检查设备是否在场景中
		if !strings.Contains(scene.DeviceIds, fmt.Sprintf("%d", deviceId)) {
			continue
		}

		// 解析触发条件
		var triggerCond map[string]interface{}
		if scene.TriggerCond != "" {
			json.Unmarshal([]byte(scene.TriggerCond), &triggerCond)
		}

		// 检查是否满足触发条件
		if e.checkTriggerCondition(deviceStatus, triggerCond) {
			// 触发场景执行
			go func() {
				in := &ExecuteSceneIn{
					SceneId:    scene.Id,
					TriggerSrc: "device",
					TriggerData: map[string]interface{}{
						"device_id": deviceId,
						"status":    deviceStatus,
					},
				}
				_, _ = e.ExecuteScene(in)
			}()
		}
	}

	return nil
}

// checkTriggerCondition 检查触发条件（简化实现）
func (e *PlatformDeviceService) checkTriggerCondition(deviceStatus map[string]interface{}, triggerCond map[string]interface{}) bool {
	// 简化实现：检查所有条件是否匹配
	for key, value := range triggerCond {
		if deviceStatus[key] != value {
			return false
		}
	}
	return true
}

// DeviceGroup 设备分组管理
const (
	GroupMaxLevel      = 5  // 最大分组层级
	MaxGroupsPerDevice = 10 // 单个设备最大分组数
)

// CreateGroupIn 创建分组输入
type CreateGroupIn struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ParentId    *int   `json:"parent_id"`
	Sort        int    `json:"sort"`
	Operator    string `json:"operator"`
}

// CreateGroupOut 创建分组输出
type CreateGroupOut struct {
	GroupId  int    `json:"group_id"`
	Name     string `json:"name"`
	ParentId *int   `json:"parent_id"`
	Level    int16  `json:"level"`
	Message  string `json:"message"`
}

// CreateGroup 创建设备分组
func (e *PlatformDeviceService) CreateGroup(in *CreateGroupIn) (*CreateGroupOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	if strings.TrimSpace(in.Name) == "" {
		return nil, fmt.Errorf("分组名称不能为空")
	}

	// 校验父分组
	var level int16 = 1
	if in.ParentId != nil && *in.ParentId > 0 {
		var parent models.DeviceGroup
		if err := e.Orm.Table("device_group").Where("id = ?", *in.ParentId).Take(&parent).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("父分组不存在")
			}
			return nil, err
		}
		if parent.Status != 1 {
			return nil, fmt.Errorf("父分组已禁用")
		}
		level = parent.Level + 1
		if level > GroupMaxLevel {
			return nil, fmt.Errorf("分组层级不能超过 %d 级", GroupMaxLevel)
		}
	}

	// 校验名称唯一性（同一父分组下名称唯一）
	var count int64
	query := e.Orm.Table("device_group").Where("name = ?", strings.TrimSpace(in.Name))
	if in.ParentId != nil && *in.ParentId > 0 {
		query = query.Where("parent_id = ?", *in.ParentId)
	} else {
		query = query.Where("parent_id IS NULL")
	}
	if err := query.Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("分组名称已存在")
	}

	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "admin"
	}

	// 创建分组
	group := models.DeviceGroup{
		Name:        strings.TrimSpace(in.Name),
		Description: strings.TrimSpace(in.Description),
		ParentId:    in.ParentId,
		Level:       level,
		Sort:        in.Sort,
		DeviceCount: 0,
		Status:      1,
		CreateBy:    0,
		UpdateBy:    0,
	}

	if err := e.Orm.Create(&group).Error; err != nil {
		return nil, err
	}

	return &CreateGroupOut{
		GroupId:  group.Id,
		Name:     group.Name,
		ParentId: group.ParentId,
		Level:    group.Level,
		Message:  "分组创建成功",
	}, nil
}

// UpdateGroupIn 更新分组输入
type UpdateGroupIn struct {
	GroupId     int    `json:"group_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
	Status      *int16 `json:"status"`
	Operator    string `json:"operator"`
}

// UpdateGroupOut 更新分组输出
type UpdateGroupOut struct {
	GroupId int    `json:"group_id"`
	Message string `json:"message"`
}

// UpdateGroup 更新设备分组
func (e *PlatformDeviceService) UpdateGroup(in *UpdateGroupIn) (*UpdateGroupOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.GroupId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询分组
	var group models.DeviceGroup
	if err := e.Orm.Table("device_group").Where("id = ?", in.GroupId).Take(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("分组不存在")
		}
		return nil, err
	}

	// 校验名称唯一性
	if strings.TrimSpace(in.Name) != "" && in.Name != group.Name {
		var count int64
		query := e.Orm.Table("device_group").Where("name = ?", strings.TrimSpace(in.Name))
		if group.ParentId != nil && *group.ParentId > 0 {
			query = query.Where("parent_id = ?", *group.ParentId)
		} else {
			query = query.Where("parent_id IS NULL")
		}
		query = query.Where("id != ?", in.GroupId)
		if err := query.Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, fmt.Errorf("分组名称已存在")
		}
	}

	// 更新分组
	updateData := make(map[string]interface{})
	if strings.TrimSpace(in.Name) != "" {
		updateData["name"] = strings.TrimSpace(in.Name)
	}
	updateData["description"] = strings.TrimSpace(in.Description)
	updateData["sort"] = in.Sort
	if in.Status != nil && (*in.Status == 0 || *in.Status == 1) {
		updateData["status"] = *in.Status
	}
	updateData["updated_at"] = time.Now()

	if err := e.Orm.Model(&models.DeviceGroup{}).Where("id = ?", in.GroupId).Updates(updateData).Error; err != nil {
		return nil, err
	}

	return &UpdateGroupOut{
		GroupId: in.GroupId,
		Message: "分组更新成功",
	}, nil
}

// DeleteGroupIn 删除分组输入
type DeleteGroupIn struct {
	GroupId  int    `json:"group_id"`
	Operator string `json:"operator"`
}

// DeleteGroupOut 删除分组输出
type DeleteGroupOut struct {
	GroupId      int    `json:"group_id"`
	DeletedCount int64  `json:"deleted_count"`
	Message      string `json:"message"`
}

// DeleteGroup 删除设备分组
func (e *PlatformDeviceService) DeleteGroup(in *DeleteGroupIn) (*DeleteGroupOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.GroupId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询分组
	var group models.DeviceGroup
	if err := e.Orm.Table("device_group").Where("id = ?", in.GroupId).Take(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("分组不存在")
		}
		return nil, err
	}

	// 检查是否有子分组
	var childCount int64
	if err := e.Orm.Table("device_group").Where("parent_id = ?", in.GroupId).Count(&childCount).Error; err != nil {
		return nil, err
	}
	if childCount > 0 {
		return nil, fmt.Errorf("分组下存在子分组，无法删除")
	}

	now := time.Now()
	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "admin"
	}

	var deletedCount int64
	err := e.Orm.Transaction(func(tx *gorm.DB) error {
		// 删除分组
		if err := tx.Delete(&models.DeviceGroup{}, in.GroupId).Error; err != nil {
			return err
		}

		// 删除设备分组关联
		result := tx.Where("group_id = ?", in.GroupId).Delete(&models.DeviceGroupRelation{})
		deletedCount = result.RowsAffected

		// 记录操作日志
		log := map[string]interface{}{
			"group_id":     in.GroupId,
			"group_name":   group.Name,
			"operator":     op,
			"operate_time": now,
			"action":       "delete_group",
		}
		logJSON, _ := json.Marshal(log)
		return tx.Exec(`INSERT INTO device_event_log (device_id, sn, event_type, content, operator) VALUES (0, '', 'admin_delete_group', ?, ?)`, string(logJSON), op).Error
	})

	if err != nil {
		return nil, err
	}

	return &DeleteGroupOut{
		GroupId:      in.GroupId,
		DeletedCount: deletedCount,
		Message:      "分组删除成功",
	}, nil
}

// BatchManageDevicesIn 批量管理设备输入
type BatchManageDevicesIn struct {
	GroupId   int      `json:"group_id"`
	DeviceIds []int64  `json:"device_ids"`
	Sns       []string `json:"sns"`
	Operation string   `json:"operation"` // add/remove/replace
	Operator  string   `json:"operator"`
}

// BatchManageDevicesOut 批量管理设备输出
type BatchManageDevicesOut struct {
	GroupId        int      `json:"group_id"`
	SuccessCount   int      `json:"success_count"`
	FailCount      int      `json:"fail_count"`
	SuccessDevices []string `json:"success_devices"`
	FailDevices    []string `json:"fail_devices"`
	Message        string   `json:"message"`
}

// BatchManageDevices 批量管理设备分组
func (e *PlatformDeviceService) BatchManageDevices(in *BatchManageDevicesIn) (*BatchManageDevicesOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.GroupId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}
	if len(in.DeviceIds) == 0 && len(in.Sns) == 0 {
		return nil, fmt.Errorf("device_ids 或 sns 至少提供一个")
	}
	if in.Operation != "add" && in.Operation != "remove" && in.Operation != "replace" {
		return nil, fmt.Errorf("operation 仅支持 add/remove/replace")
	}

	// 查询分组
	var group models.DeviceGroup
	if err := e.Orm.Table("device_group").Where("id = ?", in.GroupId).Take(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("分组不存在")
		}
		return nil, err
	}
	if group.Status != 1 {
		return nil, fmt.Errorf("分组已禁用")
	}

	op := strings.TrimSpace(in.Operator)
	if op == "" {
		op = "admin"
	}

	var successDevices []string
	var failDevices []string

	err := e.Orm.Transaction(func(tx *gorm.DB) error {
		// 解析设备 ID 列表
		var deviceIds []int64
		if len(in.DeviceIds) > 0 {
			deviceIds = in.DeviceIds
		} else if len(in.Sns) > 0 {
			var devices []struct {
				ID int64  `gorm:"column:id"`
				Sn string `gorm:"column:sn"`
			}
			if err := tx.Table("device").Select("id, sn").Where("sn IN ?", in.Sns).Find(&devices).Error; err != nil {
				return err
			}
			for _, dev := range devices {
				deviceIds = append(deviceIds, dev.ID)
				successDevices = append(successDevices, dev.Sn)
			}
			// 找出无效 SN
			for _, sn := range in.Sns {
				found := false
				for _, dev := range devices {
					if dev.Sn == sn {
						found = true
						break
					}
				}
				if !found {
					failDevices = append(failDevices, sn)
				}
			}
		}

		if len(deviceIds) == 0 {
			return fmt.Errorf("无有效设备")
		}

		if in.Operation == "add" {
			// 添加设备到分组
			for _, deviceId := range deviceIds {
				// 检查设备是否存在
				var devCount int64
				if err := tx.Table("device").Where("id = ? AND deleted_at IS NULL", deviceId).Count(&devCount).Error; err != nil {
					failDevices = append(failDevices, fmt.Sprintf("id:%d", deviceId))
					continue
				}
				if devCount == 0 {
					failDevices = append(failDevices, fmt.Sprintf("id:%d", deviceId))
					continue
				}

				// 检查设备分组数
				var groupCount int64
				if err := tx.Table("device_group_relation").Where("device_id = ?", deviceId).Count(&groupCount).Error; err != nil {
					failDevices = append(failDevices, fmt.Sprintf("id:%d", deviceId))
					continue
				}
				if groupCount >= MaxGroupsPerDevice {
					failDevices = append(failDevices, fmt.Sprintf("id:%d", deviceId))
					continue
				}

				// 检查是否已在分组中
				var existCount int64
				if err := tx.Table("device_group_relation").Where("group_id = ? AND device_id = ?", in.GroupId, deviceId).Count(&existCount).Error; err != nil {
					failDevices = append(failDevices, fmt.Sprintf("id:%d", deviceId))
					continue
				}
				if existCount > 0 {
					failDevices = append(failDevices, fmt.Sprintf("id:%d", deviceId))
					continue
				}

				// 查询设备 SN
				var sn string
				tx.Table("device").Where("id = ?", deviceId).Select("sn").Scan(&sn)

				// 添加关联
				relation := models.DeviceGroupRelation{
					GroupId:  in.GroupId,
					DeviceId: deviceId,
					DeviceSn: sn,
				}
				if err := tx.Create(&relation).Error; err != nil {
					failDevices = append(failDevices, fmt.Sprintf("id:%d", deviceId))
					continue
				}

				// 更新设备影子（可选）
				e.updateDeviceShadow(tx, deviceId, map[string]interface{}{"group_id": in.GroupId})
			}
		} else if in.Operation == "remove" {
			// 从分组移除设备
			for _, deviceId := range deviceIds {
				result := tx.Where("group_id = ? AND device_id = ?", in.GroupId, deviceId).Delete(&models.DeviceGroupRelation{})
				if result.RowsAffected == 0 {
					failDevices = append(failDevices, fmt.Sprintf("id:%d", deviceId))
				}
			}
		} else if in.Operation == "replace" {
			// 替换分组所有设备
			// 先删除所有关联
			if err := tx.Where("group_id = ?", in.GroupId).Delete(&models.DeviceGroupRelation{}).Error; err != nil {
				return err
			}

			// 添加新关联
			for _, deviceId := range deviceIds {
				var sn string
				tx.Table("device").Where("id = ?", deviceId).Select("sn").Scan(&sn)

				relation := models.DeviceGroupRelation{
					GroupId:  in.GroupId,
					DeviceId: deviceId,
					DeviceSn: sn,
				}
				if err := tx.Create(&relation).Error; err != nil {
					failDevices = append(failDevices, fmt.Sprintf("id:%d", deviceId))
				}
			}
		}

		// 更新分组设备数量
		var count int64
		if err := tx.Table("device_group_relation").Where("group_id = ?", in.GroupId).Count(&count).Error; err != nil {
			return err
		}
		return tx.Model(&models.DeviceGroup{}).Where("id = ?", in.GroupId).Update("device_count", count).Error
	})

	if err != nil {
		return nil, err
	}

	return &BatchManageDevicesOut{
		GroupId:        in.GroupId,
		SuccessCount:   len(successDevices),
		FailCount:      len(failDevices),
		SuccessDevices: successDevices,
		FailDevices:    failDevices,
		Message:        fmt.Sprintf("操作完成，成功 %d 个，失败 %d 个", len(successDevices), len(failDevices)),
	}, nil
}

// GetGroupListIn 获取分组列表输入
type GetGroupListIn struct {
	ParentId *int   `json:"parent_id"`
	Status   *int16 `json:"status"`
	Keyword  string `json:"keyword"`
}

// GetGroupListOut 获取分组列表输出
type GetGroupListOut struct {
	List []models.DeviceGroup `json:"list"`
}

// GetGroupList 获取设备分组列表
func (e *PlatformDeviceService) GetGroupList(in *GetGroupListIn) (*GetGroupListOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}

	query := e.Orm.Table("device_group").Order("level ASC, sort ASC, id DESC")

	if in.ParentId != nil {
		if *in.ParentId > 0 {
			query = query.Where("parent_id = ?", *in.ParentId)
		} else {
			query = query.Where("parent_id IS NULL")
		}
	}

	if in.Status != nil {
		query = query.Where("status = ?", *in.Status)
	}

	if strings.TrimSpace(in.Keyword) != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+strings.TrimSpace(in.Keyword)+"%", "%"+strings.TrimSpace(in.Keyword)+"%")
	}

	var list []models.DeviceGroup
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}

	return &GetGroupListOut{
		List: list,
	}, nil
}

// GetGroupDetailIn 获取分组详情输入
type GetGroupDetailIn struct {
	GroupId int `json:"group_id"`
}

// GetGroupDetailOut 获取分组详情输出
type GetGroupDetailOut struct {
	Group   models.DeviceGroup       `json:"group"`
	Devices []map[string]interface{} `json:"devices"`
}

// GetGroupDetail 获取分组详情
func (e *PlatformDeviceService) GetGroupDetail(in *GetGroupDetailIn) (*GetGroupDetailOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.GroupId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询分组
	var group models.DeviceGroup
	if err := e.Orm.Table("device_group").Where("id = ?", in.GroupId).Take(&group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("分组不存在")
		}
		return nil, err
	}

	// 查询分组设备
	var relations []models.DeviceGroupRelation
	if err := e.Orm.Table("device_group_relation").Where("group_id = ?", in.GroupId).Find(&relations).Error; err != nil {
		return nil, err
	}

	var devices []map[string]interface{}
	if len(relations) > 0 {
		var deviceIds []int64
		for _, rel := range relations {
			deviceIds = append(deviceIds, rel.DeviceId)
		}

		var deviceList []struct {
			ID           int64      `gorm:"column:id"`
			Sn           string     `gorm:"column:sn"`
			Model        string     `gorm:"column:model"`
			OnlineStatus int16      `gorm:"column:online_status"`
			Status       int16      `gorm:"column:status"`
			LastActiveAt *time.Time `gorm:"column:last_active_at"`
		}
		if err := e.Orm.Table("device").Select("id, sn, model, online_status, status, last_active_at").Where("id IN ?", deviceIds).Find(&deviceList).Error; err != nil {
			return nil, err
		}

		for _, dev := range deviceList {
			devices = append(devices, map[string]interface{}{
				"id":             dev.ID,
				"sn":             dev.Sn,
				"model":          dev.Model,
				"online_status":  dev.OnlineStatus,
				"status":         dev.Status,
				"last_active_at": dev.LastActiveAt,
			})
		}
	}

	return &GetGroupDetailOut{
		Group:   group,
		Devices: devices,
	}, nil
}

// updateDeviceShadow 更新设备影子（简化实现）
func (e *PlatformDeviceService) updateDeviceShadow(tx *gorm.DB, deviceId int64, shadow map[string]interface{}) {
	// 实际实现可能需要更新 Redis 或设备影子表
	// 这里仅做示意
	shadowJSON, _ := json.Marshal(shadow)
	_ = shadowJSON
	// 可以调用 Redis 更新
}

// validateLogType 校验日志类型
func validateLogType(logType string) error {
	validTypes := map[string]bool{
		LogTypeSystem:    true,
		LogTypeError:     true,
		LogTypeOperation: true,
		LogTypeStatus:    true,
	}
	if !validTypes[logType] {
		return fmt.Errorf("无效的日志类型：%s", logType)
	}
	return nil
}

// truncateLogContent 截断日志内容（防止过长）
func truncateLogContent(content string, maxLen int) string {
	if utf8.RuneCountInString(content) <= maxLen {
		return content
	}
	return string([]rune(content)[:maxLen])
}

// sendAlert 发送告警
func (e *PlatformDeviceService) sendAlert(log models.DeviceLog, sn string) {
	// 更新告警发送状态
	e.Orm.Model(&models.DeviceLog{}).
		Where("id = ?", log.Id).
		Update("alert_sent", 1)

	// TODO: 实现告警通知逻辑（邮件、短信、钉钉等）
	// 示例：发送邮件给管理员
	// mailService.SendAlert(
	// 	"设备异常告警",
	// 	fmt.Sprintf("设备 %s 上报错误日志：%s", sn, log.Content),
	// 	[]string{"admin@example.com"},
	// )

	// 使用标准日志记录告警
	fmt.Printf("设备 %s 上报错误日志，已触发告警：%s\n", sn, log.Content)
}

// GetDeviceLogListIn 设备日志列表输入
type GetDeviceLogListIn struct {
	DeviceID  int64     `json:"device_id"`
	Sn        string    `json:"sn"`
	LogType   string    `json:"log_type"`
	LogLevel  string    `json:"log_level"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Processed *int16    `json:"processed"`
	AlertSent *int16    `json:"alert_sent"`
	Page      int       `json:"page"`
	PageSize  int       `json:"page_size"`
	SortBy    string    `json:"sort_by"`
	SortOrder string    `json:"sort_order"`
}

// GetDeviceLogListOut 设备日志列表输出
type GetDeviceLogListOut struct {
	List  []models.DeviceLog `json:"list"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Size  int                `json:"size"`
}

// GetDeviceLogList 获取设备日志列表
func (e *PlatformDeviceService) GetDeviceLogList(in *GetDeviceLogListIn) (*GetDeviceLogListOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil {
		return nil, ErrPlatformDeviceInvalid
	}
	if in.DeviceID <= 0 && strings.TrimSpace(in.Sn) == "" {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询设备
	var dev struct {
		ID int64 `gorm:"column:id"`
	}
	q := e.Orm.Table("device").Select("id")
	if in.DeviceID > 0 {
		q = q.Where("id = ?", in.DeviceID)
	} else {
		q = q.Where("sn = ?", strings.TrimSpace(in.Sn))
	}
	if err := q.Take(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}

	// 构建查询条件
	query := e.Orm.Table("device_log").Where("device_id = ?", dev.ID)

	// 筛选条件
	if in.LogType != "" {
		query = query.Where("log_type = ?", in.LogType)
	}
	if in.LogLevel != "" {
		query = query.Where("log_level = ?", in.LogLevel)
	}
	if !in.StartTime.IsZero() {
		query = query.Where("report_time >= ?", in.StartTime)
	}
	if !in.EndTime.IsZero() {
		query = query.Where("report_time <= ?", in.EndTime)
	}
	if in.Processed != nil {
		query = query.Where("processed = ?", *in.Processed)
	}
	if in.AlertSent != nil {
		query = query.Where("alert_sent = ?", *in.AlertSent)
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	page := in.Page
	if page <= 0 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 排序
	sortBy := in.SortBy
	if sortBy == "" {
		sortBy = "report_time"
	}
	sortOrder := in.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// 查询列表
	var logs []models.DeviceLog
	if err := query.Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, err
	}

	return &GetDeviceLogListOut{
		List:  logs,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// DeviceShare 设备共享管理
const (
	ShareTypeTemp   = int16(1) // 临时共享
	ShareTypePerm   = int16(2) // 永久共享
	ShareStatusWait = int16(0) // 待确认
	ShareStatusAct  = int16(1) // 已生效
	ShareStatusCan  = int16(2) // 已取消
	ShareStatusExp  = int16(3) // 已过期
)

// CreateShareIn 创建共享输入
type CreateShareIn struct {
	DeviceId    int64           `json:"device_id"`
	SharedUser  string          `json:"shared_user"`
	ShareType   int16           `json:"share_type"`
	Permissions map[string]bool `json:"permissions"`
	ExpireTime  *time.Time      `json:"expire_time"`
	Remark      string          `json:"remark"`
	Operator    string          `json:"operator"`
}

// CreateShareOut 创建共享输出
type CreateShareOut struct {
	ShareId  int    `json:"share_id"`
	DeviceId int64  `json:"device_id"`
	DeviceSn string `json:"device_sn"`
	Message  string `json:"message"`
}

// CreateShare 创建设备共享
func (e *PlatformDeviceService) CreateShare(in *CreateShareIn) (*CreateShareOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.DeviceId <= 0 || strings.TrimSpace(in.SharedUser) == "" {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询设备
	var device models.Device
	if err := e.Orm.Table("device").Where("id = ? AND deleted_at IS NULL", in.DeviceId).Take(&device).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPlatformDeviceNotFound
		}
		return nil, err
	}

	// 查询设备主人
	var ownerBind struct {
		UserId int64 `gorm:"column:user_id"`
	}
	if err := e.Orm.Table("user_device_bind").Where("device_id = ? AND status = 1", in.DeviceId).Take(&ownerBind).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("设备未绑定用户")
		}
		return nil, err
	}

	// 校验被共享人账号是否存在
	var sharedUser struct {
		UserId int64  `gorm:"column:user_id"`
		Nick   string `gorm:"column:nickname"`
	}
	if err := e.Orm.Table("sys_user").Where("username = ? OR email = ? OR phone = ?", in.SharedUser, in.SharedUser, in.SharedUser).Take(&sharedUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("被共享人账号不存在")
		}
		return nil, err
	}

	// 检查是否已存在共享记录
	var existCount int64
	if err := e.Orm.Table("device_share").
		Where("device_id = ? AND shared_user_id = ? AND status IN (?)", in.DeviceId, sharedUser.UserId, []int16{ShareStatusWait, ShareStatusAct}).
		Count(&existCount).Error; err != nil {
		return nil, err
	}
	if existCount > 0 {
		return nil, fmt.Errorf("该用户已存在共享记录")
	}

	// 序列化权限
	permJSON, err := json.Marshal(in.Permissions)
	if err != nil {
		return nil, fmt.Errorf("权限格式错误")
	}

	// 创建共享记录
	share := models.DeviceShare{
		DeviceId:     int(in.DeviceId),
		DeviceSn:     device.Sn,
		DeviceName:   "", // 可后续补充
		OwnerId:      int(ownerBind.UserId),
		OwnerName:    "", // 可后续补充
		SharedUserId: int(sharedUser.UserId),
		SharedUser:   in.SharedUser,
		ShareType:    in.ShareType,
		Permissions:  string(permJSON),
		Status:       ShareStatusWait,
		ExpireTime:   in.ExpireTime,
		Remark:       in.Remark,
		CreateBy:     int(ownerBind.UserId),
	}

	if err := e.Orm.Create(&share).Error; err != nil {
		return nil, err
	}

	// 记录操作日志
	shareLog := models.DeviceShareLog{
		ShareId:    share.Id,
		DeviceId:   int(in.DeviceId),
		DeviceName: "",
		OpType:     "create",
		OpContent:  fmt.Sprintf("创建共享，被共享人：%s，共享类型：%d", in.SharedUser, in.ShareType),
		Operator:   in.Operator,
		CreateBy:   int(ownerBind.UserId),
	}
	e.Orm.Create(&shareLog)

	return &CreateShareOut{
		ShareId:  share.Id,
		DeviceId: in.DeviceId,
		DeviceSn: device.Sn,
		Message:  "共享创建成功，待对方确认",
	}, nil
}

// CancelShareIn 取消共享输入
type CancelShareIn struct {
	ShareId  int    `json:"share_id"`
	Operator string `json:"operator"`
}

// CancelShareOut 取消共享输出
type CancelShareOut struct {
	ShareId int    `json:"share_id"`
	Message string `json:"message"`
}

// CancelShare 取消设备共享
func (e *PlatformDeviceService) CancelShare(in *CancelShareIn) (*CancelShareOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.ShareId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询共享记录
	var share models.DeviceShare
	if err := e.Orm.Table("device_share").Where("id = ?", in.ShareId).Take(&share).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("共享记录不存在")
		}
		return nil, err
	}

	if share.Status == ShareStatusCan {
		return nil, fmt.Errorf("共享已取消")
	}
	if share.Status == ShareStatusExp {
		return nil, fmt.Errorf("共享已过期")
	}

	now := time.Now()

	// 更新共享状态
	if err := e.Orm.Model(&models.DeviceShare{}).Where("id = ?", in.ShareId).Updates(map[string]interface{}{
		"status":     ShareStatusCan,
		"updated_at": now,
	}).Error; err != nil {
		return nil, err
	}

	// 记录操作日志
	shareLog := models.DeviceShareLog{
		ShareId:    share.Id,
		DeviceId:   share.DeviceId,
		DeviceName: share.DeviceName,
		OpType:     "cancel",
		OpContent:  "取消共享",
		Operator:   in.Operator,
	}
	e.Orm.Create(&shareLog)

	return &CancelShareOut{
		ShareId: in.ShareId,
		Message: "共享已取消",
	}, nil
}

// ConfirmShareIn 确认共享输入
type ConfirmShareIn struct {
	ShareId  int    `json:"share_id"`
	UserId   int64  `json:"user_id"`
	Operator string `json:"operator"`
}

// ConfirmShareOut 确认共享输出
type ConfirmShareOut struct {
	ShareId int    `json:"share_id"`
	Message string `json:"message"`
}

// ConfirmShare 确认设备共享
func (e *PlatformDeviceService) ConfirmShare(in *ConfirmShareIn) (*ConfirmShareOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.ShareId <= 0 || in.UserId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	// 查询共享记录
	var share models.DeviceShare
	if err := e.Orm.Table("device_share").Where("id = ?", in.ShareId).Take(&share).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("共享记录不存在")
		}
		return nil, err
	}

	if share.Status != ShareStatusWait {
		return nil, fmt.Errorf("共享状态不正确")
	}
	if share.SharedUserId != int(in.UserId) {
		return nil, fmt.Errorf("无权确认此共享")
	}

	// 检查共享是否过期
	if share.ExpireTime != nil && time.Now().After(*share.ExpireTime) {
		// 已过期
		e.Orm.Model(&models.DeviceShare{}).Where("id = ?", in.ShareId).Updates(map[string]interface{}{
			"status": ShareStatusExp,
		})
		return nil, fmt.Errorf("共享已过期")
	}

	now := time.Now()

	// 更新共享状态
	if err := e.Orm.Model(&models.DeviceShare{}).Where("id = ?", in.ShareId).Updates(map[string]interface{}{
		"status":       ShareStatusAct,
		"confirm_time": now,
		"updated_at":   now,
	}).Error; err != nil {
		return nil, err
	}

	// 创建用户设备绑定关系
	bind := models.UserDeviceBind{
		UserId:   int(in.UserId),
		DeviceId: share.DeviceId,
		DeviceSn: share.DeviceSn,
		Status:   1,
		BoundAt:  &now,
		CreateBy: int(in.UserId),
	}
	if err := e.Orm.Create(&bind).Error; err != nil {
		return nil, err
	}

	// 记录操作日志
	shareLog := models.DeviceShareLog{
		ShareId:    share.Id,
		DeviceId:   share.DeviceId,
		DeviceName: share.DeviceName,
		OpType:     "confirm",
		OpContent:  "确认共享",
		Operator:   in.Operator,
	}
	e.Orm.Create(&shareLog)

	return &ConfirmShareOut{
		ShareId: in.ShareId,
		Message: "共享已确认",
	}, nil
}

// GetShareListIn 获取共享列表输入
type GetShareListIn struct {
	UserId   int64  `json:"user_id"` // 用户 ID（查询共享给他人或他人共享给我的）
	Role     string `json:"role"`    // owner: 我共享给他人的 / shared: 他人共享给我的
	Status   *int16 `json:"status"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

// GetShareListOut 获取共享列表输出
type GetShareListOut struct {
	List  []models.DeviceShare `json:"list"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	Size  int                  `json:"size"`
}

// GetShareList 获取设备共享列表
func (e *PlatformDeviceService) GetShareList(in *GetShareListIn) (*GetShareListOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.UserId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	query := e.Orm.Table("device_share").Order("created_at DESC")

	// 根据角色筛选
	if in.Role == "owner" {
		// 我共享给他人的
		query = query.Where("owner_id = ?", in.UserId)
	} else if in.Role == "shared" {
		// 他人共享给我的
		query = query.Where("shared_user_id = ?", in.UserId)
	} else {
		// 全部
		query = query.Where("owner_id = ? OR shared_user_id = ?", in.UserId, in.UserId)
	}

	// 状态筛选
	if in.Status != nil {
		query = query.Where("status = ?", *in.Status)
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	page := in.Page
	if page <= 0 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 查询列表
	var shares []models.DeviceShare
	if err := query.Offset(offset).Limit(pageSize).Find(&shares).Error; err != nil {
		return nil, err
	}

	return &GetShareListOut{
		List:  shares,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// GetShareDetailIn 获取共享详情输入
type GetShareDetailIn struct {
	ShareId int `json:"share_id"`
}

// GetShareDetailOut 获取共享详情输出
type GetShareDetailOut struct {
	Share models.DeviceShare `json:"share"`
}

// GetShareDetail 获取共享详情
func (e *PlatformDeviceService) GetShareDetail(in *GetShareDetailIn) (*GetShareDetailOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}
	if in == nil || in.ShareId <= 0 {
		return nil, ErrPlatformDeviceInvalid
	}

	var share models.DeviceShare
	if err := e.Orm.Table("device_share").Where("id = ?", in.ShareId).Take(&share).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("共享记录不存在")
		}
		return nil, err
	}

	return &GetShareDetailOut{
		Share: share,
	}, nil
}

// GetShareLogListIn 获取共享日志列表输入
type GetShareLogListIn struct {
	ShareId  int   `json:"share_id"`
	DeviceId int64 `json:"device_id"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// GetShareLogListOut 获取共享日志列表输出
type GetShareLogListOut struct {
	List  []models.DeviceShareLog `json:"list"`
	Total int64                   `json:"total"`
	Page  int                     `json:"page"`
	Size  int                     `json:"size"`
}

// GetShareLogList 获取共享日志列表
func (e *PlatformDeviceService) GetShareLogList(in *GetShareLogListIn) (*GetShareLogListOut, error) {
	if e.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}

	query := e.Orm.Table("device_share_log").Order("created_at DESC")

	if in.ShareId > 0 {
		query = query.Where("share_id = ?", in.ShareId)
	}
	if in.DeviceId > 0 {
		query = query.Where("device_id = ?", in.DeviceId)
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	page := in.Page
	if page <= 0 {
		page = 1
	}
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 查询列表
	var logs []models.DeviceShareLog
	if err := query.Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, err
	}

	return &GetShareLogListOut{
		List:  logs,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// CheckShareExpired 检查共享过期状态（定时任务调用）
func (e *PlatformDeviceService) CheckShareExpired() error {
	if e.Orm == nil {
		return fmt.Errorf("orm nil")
	}

	now := time.Now()

	// 查询已过期但未标记的共享记录
	var shares []models.DeviceShare
	if err := e.Orm.Table("device_share").
		Where("status = ? AND expire_time IS NOT NULL AND expire_time < ?", ShareStatusAct, now).
		Find(&shares).Error; err != nil {
		return err
	}

	for _, share := range shares {
		// 更新状态为过期
		e.Orm.Model(&models.DeviceShare{}).Where("id = ?", share.Id).Updates(map[string]interface{}{
			"status": ShareStatusExp,
		})

		// 记录日志
		shareLog := models.DeviceShareLog{
			ShareId:    share.Id,
			DeviceId:   share.DeviceId,
			DeviceName: share.DeviceName,
			OpType:     "expire",
			OpContent:  "共享过期",
		}
		e.Orm.Create(&shareLog)
	}

	return nil
}
