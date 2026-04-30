package apis

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"go-admin/app/admin/device/service"
)

// StatusLogsList 设备定时状态上报历史（分页）
// @Summary 设备状态上报日志列表
// @Tags 平台设备
// @Param sn query string false "设备 SN（与 device_id 二选一；优先 device_id）"
// @Param device_id query int false "设备主键 ID（与详情 device.id 一致时推荐，避免 SN 条件不一致）"
// @Param page query int false "页码"
// @Param pageSize query int false "每页条数"
// @Param from query string false "创建时间起（RFC3339 或 2006-01-02）"
// @Param to query string false "创建时间止（RFC3339 或 2006-01-02，日期含当日全天）"
// @Router /api/v1/platform-device/status-logs [get]
// @Security Bearer
func (e PlatformDevice) StatusLogsList(c *gin.Context) {
	var deviceID int64
	if q := strings.TrimSpace(c.Query("device_id")); q != "" {
		if v, err := strconv.ParseInt(q, 10, 64); err == nil && v > 0 {
			deviceID = v
		}
	}
	// 与 Detail 一致：先 query 再 path，避免 /platform-device/status-logs 被误解析为 /:sn 时 path_sn=「status-logs」覆盖真实 SN
	querySn := strings.TrimSpace(c.Query("sn"))
	pathSn := strings.TrimSpace(c.Param("sn"))
	sn := querySn
	if sn == "" {
		sn = pathSn
	}
	if sn == "status-logs" || sn == "detail" {
		sn = ""
	}
	if sn == "" && deviceID == 0 {
		e.Error(400, errors.New("sn or device_id required"), "请传设备 SN 或 device_id")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSizeStr := c.DefaultQuery("pageSize", "")
	if pageSizeStr == "" {
		pageSizeStr = c.DefaultQuery("page_size", "20")
	}
	pageSize, _ := strconv.Atoi(pageSizeStr)

	fromPtr, err := parseDateTimeQueryFlexible(c, "from", false)
	if err != nil {
		e.Error(400, err, "from 时间格式无效")
		return
	}
	toPtr, err := parseDateTimeQueryFlexible(c, "to", true)
	if err != nil {
		e.Error(400, err, "to 时间格式无效")
		return
	}

	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	list, total, err := svc.ListDeviceStatusLogs(sn, deviceID, page, pageSize, fromPtr, toPtr)
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			// 使用 400 + 业务提示，避免与「路由不存在」的 HTTP 404 混淆
			e.Error(400, err, "设备不存在")
			return
		}
		if errors.Is(err, service.ErrPlatformDeviceInvalid) {
			e.Error(400, err, "参数无效")
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, "查询失败")
		return
	}
	e.PageOK(list, int(total), page, pageSize, "查询成功")
}

type manualStatusReportBody struct {
	DeviceID int64 `json:"device_id"`
	Sn       string `json:"sn"`

	BatteryLevel       int    `json:"battery_level"`
	StorageUsed        int64  `json:"storage_used"`
	StorageTotal       int64  `json:"storage_total"`
	SpeakerCount       int    `json:"speaker_count"`
	UwbX               *float64 `json:"uwb_x"`
	UwbY               *float64 `json:"uwb_y"`
	UwbZ               *float64 `json:"uwb_z"`
	AcousticCalibrated int    `json:"acoustic_calibrated"`
	AcousticOffset     *float64 `json:"acoustic_offset"`
	ReportedAt         string `json:"reported_at"`
}

// ManualStatusReport 管理员手动填报一条状态记录（写入 device_status_logs，report_type=manual）
// @Summary 手动填报设备状态
// @Tags 平台设备
// @Accept json
// @Produce json
// @Param body body manualStatusReportBody true "device_id 与 sn 二选一；其余字段与设备上报一致"
// @Router /api/v1/platform-device/manual-status-report [post]
// @Security Bearer
func (e PlatformDevice) ManualStatusReport(c *gin.Context) {
	var body manualStatusReportBody
	if err := c.ShouldBindJSON(&body); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	if body.DeviceID <= 0 && strings.TrimSpace(body.Sn) == "" {
		e.Error(400, errors.New("device_id or sn required"), "请提供 device_id 或 sn")
		return
	}

	reportedAt, err := parseReportedAtString(body.ReportedAt)
	if err != nil {
		e.Error(400, err, "reported_at 时间格式无效")
		return
	}

	ac := int16(body.AcousticCalibrated)
	if ac != 0 && ac != 1 {
		e.Error(400, errors.New("bad acoustic"), "acoustic_calibrated 仅支持 0 或 1")
		return
	}

	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	out, err := svc.ManualInsertDeviceStatusReport(&service.ManualDeviceStatusReportIn{
		DeviceID:           body.DeviceID,
		Sn:                 body.Sn,
		BatteryLevel:       body.BatteryLevel,
		StorageUsed:        body.StorageUsed,
		StorageTotal:       body.StorageTotal,
		SpeakerCount:       body.SpeakerCount,
		UwbX:               body.UwbX,
		UwbY:               body.UwbY,
		UwbZ:               body.UwbZ,
		AcousticCalibrated: ac,
		AcousticOffset:     body.AcousticOffset,
		ReportedAt:         reportedAt,
	})
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(400, err, "设备不存在")
			return
		}
		if errors.Is(err, service.ErrPlatformDeviceInvalid) {
			e.Error(400, err, err.Error())
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, "录入失败")
		return
	}
	e.OK(out, "录入成功")
}

func parseReportedAtString(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Now(), nil
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, e := time.ParseInLocation(layout, s, time.Local); e == nil {
			return t, nil
		}
	}
	if t, e := time.ParseInLocation("2006-01-02", s, time.Local); e == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
	}
	return time.Time{}, fmt.Errorf("invalid reported_at")
}

// from：日期仅 2006-01-02 时取当日 00:00:00；to 且 endOfDayIfDate=true 时取当日 23:59:59.999999999
func parseDateTimeQueryFlexible(c *gin.Context, key string, endOfDayIfDate bool) (*time.Time, error) {
	s := strings.TrimSpace(c.Query(key))
	if s == "" {
		return nil, nil
	}
	if t, err := time.ParseInLocation(time.RFC3339, s, time.Local); err == nil {
		return &t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.Local); err == nil {
		return &t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return &t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		if endOfDayIfDate {
			eod := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.Local)
			return &eod, nil
		}
		return &t, nil
	}
	return nil, fmt.Errorf("invalid %s", key)
}
