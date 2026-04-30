package apis

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"

	"go-admin/app/admin/device/service"
)

// FirmwareDeleteReq 固件删除请求体
type FirmwareDeleteReq struct {
	FirmwareID int64  `json:"firmware_id"`
	ProductKey string `json:"product_key"`
	Version    string `json:"version"`
	Confirm    bool   `json:"confirm"`
	Force      bool   `json:"force"`
	Reason     string `json:"reason"`
}

// FirmwareList 固件包分页列表
// @Summary 固件列表
// @Tags 平台设备-固件
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Param product_key query string false "产品线 product_key 精确"
// @Param version query string false "版本号（模糊）"
// @Param version_exact query bool false "true 时版本精确匹配"
// @Param version_code_min query int false "版本码下限"
// @Param version_code_max query int false "版本码上限"
// @Param device_model query string false "适用型号（模糊匹配 device_models）"
// @Param status query int false "1=启用 2=禁用，不传则全部"
// @Param keyword query string false "搜索版本号或说明"
// @Param created_from query string false "创建起 YYYY-MM-DD"
// @Param created_to query string false "创建止 YYYY-MM-DD"
// @Param sort_by query string false "created_at|version_code|download_count|version|device_models"
// @Param sort_order query string false "asc|desc 默认 desc"
// @Param include_deleted query bool false "true 时包含已软删固件（审计）"
// @Router /api/v1/platform-device/firmware/list [get]
// @Security Bearer
func (e PlatformDevice) FirmwareList(c *gin.Context) {
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	f := service.FirmwareListFilter{
		ProductKey:     strings.TrimSpace(c.Query("product_key")),
		Version:        strings.TrimSpace(c.Query("version")),
		VersionExact:   strings.EqualFold(strings.TrimSpace(c.Query("version_exact")), "true") || c.Query("version_exact") == "1",
		DeviceModel:    strings.TrimSpace(c.Query("device_model")),
		Keyword:        strings.TrimSpace(c.Query("keyword")),
		SortBy:         strings.TrimSpace(c.Query("sort_by")),
		SortOrder:      strings.TrimSpace(c.Query("sort_order")),
		IncludeDeleted: strings.EqualFold(strings.TrimSpace(c.Query("include_deleted")), "true") || c.Query("include_deleted") == "1",
	}
	if v := strings.TrimSpace(c.Query("version_code_min")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.VersionCodeMin = &n
		}
	}
	if v := strings.TrimSpace(c.Query("version_code_max")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.VersionCodeMax = &n
		}
	}
	if v := strings.TrimSpace(c.Query("status")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 16); err == nil && (n == 1 || n == 2) {
			x := int16(n)
			f.Status = &x
		} else {
			e.Error(400, errors.New("bad status"), "status 仅支持 1=启用 2=禁用")
			return
		}
	}
	if s := strings.TrimSpace(c.Query("created_from")); s != "" {
		t, err := time.ParseInLocation("2006-01-02", s, time.Local)
		if err != nil {
			e.Error(400, err, "created_from 格式 YYYY-MM-DD")
			return
		}
		f.CreatedFrom = &t
	}
	if s := strings.TrimSpace(c.Query("created_to")); s != "" {
		t, err := time.ParseInLocation("2006-01-02", s, time.Local)
		if err != nil {
			e.Error(400, err, "created_to 格式 YYYY-MM-DD")
			return
		}
		f.CreatedTo = &t
	}

	list, total, err := svc.ListFirmware(page, pageSize, f)
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询失败")
		return
	}
	e.PageOK(list, int(total), page, pageSize, "ok")
}

// FirmwareDetail 固件详情
// @Summary 固件详情
// @Tags 平台设备-固件
// @Param id path int true "固件 ID"
// @Router /api/v1/platform-device/firmware/{id} [get]
// @Security Bearer
func (e PlatformDevice) FirmwareDetail(c *gin.Context) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		e.Error(400, errors.New("bad id"), "id 无效")
		return
	}
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	out, err := svc.GetFirmwareDetail(id)
	if err != nil {
		if errors.Is(err, service.ErrFirmwareNotFound) {
			e.Error(404, err, "固件不存在")
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
	e.OK(out, "ok")
}

func parseFirmwareBool(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "1" || s == "true" || s == "yes" || s == "on"
}

// FirmwareUpload 上传固件包（multipart：product_key、version、file 必填）
// @Summary 固件上传
// @Tags 平台设备-固件
// @Accept multipart/form-data
// @Param product_key formData string true "产品线 product_key"
// @Param version formData string true "版本号，如 v1.2.3"
// @Param file formData file true "固件文件 .bin 或 .zip"
// @Param version_code formData int false "整型版本码，不传则根据 version 推算"
// @Param device_models formData string false "适用型号，逗号分隔或 JSON"
// @Param force_update formData bool false "是否强制升级"
// @Param min_sys_version formData string false "最低系统版本"
// @Param description formData string false "版本说明"
// @Param status formData int false "1=启用 2=禁用，默认 1"
// @Router /api/v1/platform-device/firmware/upload [post]
// @Security Bearer
func (e PlatformDevice) FirmwareUpload(c *gin.Context) {
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	maxBytes := service.MaxFirmwareUploadBytes
	if err := c.Request.ParseMultipartForm(maxBytes + (1 << 20)); err != nil {
		e.Error(400, err, "解析表单失败")
		return
	}

	productKey := strings.TrimSpace(c.PostForm("product_key"))
	version := strings.TrimSpace(c.PostForm("version"))
	if productKey == "" || version == "" {
		e.Error(400, service.ErrPlatformDeviceInvalid, "product_key 与 version 为必填")
		return
	}
	if err := service.ValidateFirmwareVersionFormat(version); err != nil {
		e.Error(400, err, err.Error())
		return
	}
	if err := svc.ValidateProductKeyForFirmware(productKey); err != nil {
		if errors.Is(err, service.ErrFirmwareProductKey) {
			e.Error(400, err, err.Error())
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, "校验产品线失败")
		return
	}

	var versionCode *int
	if s := strings.TrimSpace(c.PostForm("version_code")); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			e.Error(400, errors.New("bad version_code"), "version_code 须为正整数")
			return
		}
		versionCode = &n
	}

	fwStatus := int16(1)
	if s := strings.TrimSpace(c.PostForm("status")); s != "" {
		n, err := strconv.ParseInt(s, 10, 16)
		if err != nil || (n != 1 && n != 2) {
			e.Error(400, errors.New("bad status"), "status 仅支持 1=启用 2=禁用")
			return
		}
		fwStatus = int16(n)
	}

	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		e.Error(400, errors.New("missing file"), "请上传固件文件 file")
		return
	}

	src, err := fh.Open()
	if err != nil {
		e.Error(400, err, "读取文件失败")
		return
	}
	defer src.Close()

	rel, size, md5hex, sha256hex, err := service.SaveFirmwareUploadedFileWithPK(productKey, fh.Filename, src, maxBytes)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFirmwareFileType):
			e.Error(400, err, err.Error())
		case errors.Is(err, service.ErrFirmwareFileTooLarge):
			e.Error(400, err, err.Error())
		default:
			e.Logger.Error(err)
			e.Error(500, err, "上传失败，请重试")
		}
		return
	}
	fullPath := filepath.FromSlash(rel)
	defer func() {
		if fullPath != "" {
			_ = os.Remove(fullPath)
		}
	}()

	op := strconv.Itoa(user.GetUserId(c))
	if op == "0" {
		op = "admin"
	}

	out, err := svc.UploadFirmware(&service.FirmwareUploadIn{
		ProductKey:    productKey,
		Version:       version,
		VersionCode:   versionCode,
		DeviceModels:  strings.TrimSpace(c.PostForm("device_models")),
		ForceUpdate:   parseFirmwareBool(c.PostForm("force_update")),
		MinSysVersion: strings.TrimSpace(c.PostForm("min_sys_version")),
		Description:   strings.TrimSpace(c.PostForm("description")),
		FwStatus:      fwStatus,
		FilePath:      rel,
		FileSize:      size,
		MD5Hex:        md5hex,
		SHA256Hex:     sha256hex,
		Operator:      op,
	})
	if err != nil {
		e.Logger.Error(err)
		switch {
		case errors.Is(err, service.ErrFirmwareDuplicateVersion):
			e.Error(400, err, err.Error())
		case errors.Is(err, service.ErrFirmwareVersionFormat):
			e.Error(400, err, err.Error())
		case errors.Is(err, service.ErrFirmwareProductKey):
			e.Error(400, err, err.Error())
		case errors.Is(err, service.ErrIotProductDisabled):
			e.Error(400, err, err.Error())
		case errors.Is(err, service.ErrFirmwareChecksum):
			e.Error(400, err, err.Error())
		case errors.Is(err, service.ErrPlatformDeviceInvalid):
			e.Error(400, err, "参数无效")
		default:
			e.Error(500, err, "保存固件信息失败")
		}
		return
	}

	fullPath = ""
	e.OK(out, out.Message)
}

// FirmwareDelete 固件软删除（需先禁用、未发布或已撤销发布；confirm 须为 true）
// @Summary 固件删除
// @Tags 平台设备-固件
// @Accept json
// @Produce json
// @Param body body FirmwareDeleteReq true "firmware_id 与 (product_key+version) 二选一"
// @Router /api/v1/platform-device/firmware/delete [post]
// @Security Bearer
func (e PlatformDevice) FirmwareDelete(c *gin.Context) {
	var req FirmwareDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "请求体无效")
		return
	}

	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	op := strconv.Itoa(user.GetUserId(c))
	if op == "0" {
		op = "admin"
	}

	out, err := svc.DeleteFirmware(&service.FirmwareDeleteIn{
		FirmwareID: req.FirmwareID,
		ProductKey: strings.TrimSpace(req.ProductKey),
		Version:    strings.TrimSpace(req.Version),
		Confirm:    req.Confirm,
		Force:      req.Force,
		Reason:     strings.TrimSpace(req.Reason),
		Operator:   op,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFirmwareDeleteConfirm):
			e.Error(400, err, err.Error())
		case errors.Is(err, service.ErrFirmwareDeleteNoTarget):
			e.Error(400, err, err.Error())
		case errors.Is(err, service.ErrFirmwareNotFound):
			e.Error(404, err, "固件记录不存在")
		case errors.Is(err, service.ErrFirmwareMustDisable):
			e.Error(400, err, err.Error())
		case errors.Is(err, service.ErrFirmwareMustUnpublish):
			e.Error(400, err, err.Error())
		case errors.Is(err, service.ErrFirmwareDeviceCache):
			e.Error(400, err, err.Error())
		case errors.Is(err, service.ErrFirmwareDeleteStorage):
			e.Error(500, err, err.Error())
		case errors.Is(err, service.ErrPlatformDeviceInvalid):
			e.Error(400, err, "参数无效")
		default:
			e.Logger.Error(err)
			e.Error(500, err, "删除失败")
		}
		return
	}

	if out != nil && !out.Success {
		e.OK(out, out.Message)
		return
	}
	e.OK(out, out.Message)
}

// FirmwareUpdateReq 固件信息更新请求
type FirmwareUpdateReq struct {
	FirmwareID    int64    `json:"firmware_id"`
	ProductKey    string   `json:"product_key"`
	Version       string   `json:"version"`
	VersionDesc   string   `json:"version_description"`
	DeviceModels  []string `json:"device_models"`
	ForceUpdate   *bool    `json:"force_update"`
	MinSysVersion string   `json:"min_sys_version"`
	Status        *int16   `json:"status"`
	Tags          []string `json:"tags"`
	Confirm       bool     `json:"confirm"`
}

// FirmwareUpdate 固件信息修改（不修改固件包文件）
// @Summary 修改固件信息
// @Tags 平台设备 - 固件
// @Accept application/json
// @Param firmware_id body int true "固件 ID"
// @Param version_description body string false "版本说明"
// @Param device_models body array false "适用型号列表"
// @Param force_update body bool false "是否强制升级"
// @Param min_sys_version body string false "最低系统版本"
// @Param status body int false "固件状态 1=启用 2=禁用"
// @Param tags body array false "标签列表"
// @Param confirm body bool true "确认标识"
// @Router /api/v1/platform-device/firmware/update [post]
// @Security Bearer
func (e PlatformDevice) FirmwareUpdate(c *gin.Context) {
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 解析请求体
	var req FirmwareUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "请求参数格式错误")
		return
	}

	// 校验必填参数
	if req.FirmwareID <= 0 && (req.ProductKey == "" || req.Version == "") {
		e.Error(400, errors.New("bad request"), "firmware_id 或 product_key+version 必填其一")
		return
	}

	// 权限校验（通过 JWT 获取操作人）
	operator := user.GetUserId(c)
	if operator <= 0 {
		e.Error(401, errors.New("unauthorized"), "请先登录")
		return
	}

	// 调用服务层更新
	out, err := svc.FirmwareUpdate(service.FirmwareUpdateRequest{
		FirmwareID:    req.FirmwareID,
		ProductKey:    req.ProductKey,
		Version:       req.Version,
		VersionDesc:   req.VersionDesc,
		DeviceModels:  req.DeviceModels,
		ForceUpdate:   req.ForceUpdate,
		MinSysVersion: req.MinSysVersion,
		Status:        req.Status,
		Tags:          req.Tags,
		Confirm:       req.Confirm,
		Operator:      int64(operator),
	})

	if err != nil {
		switch {
		case errors.Is(err, service.ErrFirmwareNotFound):
			e.Error(404, err, "固件记录不存在")
		case errors.Is(err, service.ErrFirmwareVersionProtected):
			e.Error(400, err, "版本号不可修改，请重新上传固件")
		case errors.Is(err, service.ErrFirmwareTaskConflict):
			e.Error(400, err, "固件关联进行中的升级任务，部分字段修改受限")
		default:
			e.Logger.Error(err)
			e.Error(500, err, "更新失败")
		}
		return
	}

	if out == nil || !out.Success {
		msg := "更新失败"
		if out != nil && out.Message != "" {
			msg = out.Message
		}
		e.Error(400, errors.New("update failed"), msg)
		return
	}

	e.OK(gin.H{
		"firmware_id":    out.FirmwareID,
		"updated_fields": out.UpdatedFields,
		"updated_at":     out.UpdatedAt,
		"message":        out.Message,
	}, "固件信息更新成功")
}

// FirmwareHistoryReq 版本历史查询请求
type FirmwareHistoryReq struct {
	ProductKey    string `json:"product_key"`
	DeviceModel   string `json:"device_model"`
	DateFrom      string `json:"date_from"`
	DateTo        string `json:"date_to"`
	ReleaseType   string `json:"release_type"`
	Status        string `json:"status"`
	Page          int    `json:"page"`
	PageSize      int    `json:"page_size"`
	WithStats     bool   `json:"with_stats"`
	WithChangeLog bool   `json:"with_change_log"`
}

// FirmwareHistory 版本历史列表
// @Summary 版本历史列表
// @Tags 平台设备 - 固件
// @Param product_key query string false "产品标识"
// @Param device_model query string false "设备型号"
// @Param date_from query string false "开始日期 YYYY-MM-DD"
// @Param date_to query string false "结束日期 YYYY-MM-DD"
// @Param release_type query string false "发布类型：formal/test/gray/rollback/emergency"
// @Param status query string false "状态：draft/testing/published/withdrawn/obsolete"
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Param with_stats query bool false "是否包含统计数据"
// @Param with_change_log query bool false "是否包含变更日志"
// @Router /api/v1/platform-device/firmware/history [get]
// @Security Bearer
func (e PlatformDevice) FirmwareHistory(c *gin.Context) {
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 解析请求参数
	req := FirmwareHistoryReq{
		ProductKey:    strings.TrimSpace(c.Query("product_key")),
		DeviceModel:   strings.TrimSpace(c.Query("device_model")),
		DateFrom:      strings.TrimSpace(c.Query("date_from")),
		DateTo:        strings.TrimSpace(c.Query("date_to")),
		ReleaseType:   strings.TrimSpace(c.Query("release_type")),
		Status:        strings.TrimSpace(c.Query("status")),
		Page:          1,
		PageSize:      20,
		WithStats:     false,
		WithChangeLog: false,
	}

	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		req.Page = p
	}
	if ps, err := strconv.Atoi(c.Query("page_size")); err == nil && ps > 0 && ps <= 100 {
		req.PageSize = ps
	}
	if strings.EqualFold(c.Query("with_stats"), "true") || c.Query("with_stats") == "1" {
		req.WithStats = true
	}
	if strings.EqualFold(c.Query("with_change_log"), "true") || c.Query("with_change_log") == "1" {
		req.WithChangeLog = true
	}

	// 参数校验
	if req.ProductKey == "" && req.DeviceModel == "" {
		e.Error(400, errors.New("bad request"), "product_key 或 device_model 至少传一个")
		return
	}

	// 调用服务层查询
	result, err := svc.FirmwareHistory(service.FirmwareHistoryRequest{
		ProductKey:    req.ProductKey,
		DeviceModel:   req.DeviceModel,
		DateFrom:      req.DateFrom,
		DateTo:        req.DateTo,
		ReleaseType:   req.ReleaseType,
		Status:        req.Status,
		Page:          req.Page,
		PageSize:      req.PageSize,
		WithStats:     req.WithStats,
		WithChangeLog: req.WithChangeLog,
	})

	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询版本历史失败")
		return
	}

	e.OK(result, "查询成功")
}

// OTATaskListReq OTA 任务列表请求
type OTATaskListReq struct {
	ProductKey     string `json:"product_key"`
	TaskType       string `json:"task_type"`
	Status         int16  `json:"status"`
	StartTimeBegin string `json:"start_time_begin"`
	StartTimeEnd   string `json:"start_time_end"`
	Keyword        string `json:"keyword"`
	Page           int    `json:"page"`
	PageSize       int    `json:"page_size"`
	SortBy         string `json:"sort_by"`
	SortOrder      string `json:"sort_order"`
}

// OTATaskList OTA 任务列表
// @Summary OTA 任务列表
// @Tags 平台设备 - 固件
// @Accept application/json
// @Param product_key query string false "产品标识"
// @Param task_type query string false "任务类型"
// @Param status query int false "任务状态"
// @Param start_time_begin query string false "开始时间起始"
// @Param start_time_end query string false "开始时间结束"
// @Param keyword query string false "关键词"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param sort_by query string false "排序字段"
// @Param sort_order query string false "排序方式"
// @Router /api/v1/platform-device/ota-task/list [get]
// @Security Bearer
func (e PlatformDevice) OTATaskList(c *gin.Context) {
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 解析请求参数
	var req OTATaskListReq
	req.ProductKey = c.Query("product_key")
	req.TaskType = c.Query("task_type")

	statusStr := c.Query("status")
	if statusStr != "" {
		status, _ := strconv.ParseInt(statusStr, 10, 16)
		req.Status = int16(status)
	}

	req.StartTimeBegin = c.Query("start_time_begin")
	req.StartTimeEnd = c.Query("start_time_end")
	req.Keyword = c.Query("keyword")

	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	req.Page = page

	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if pageSize <= 0 {
		pageSize = 20
	}
	req.PageSize = pageSize

	req.SortBy = c.Query("sort_by")
	req.SortOrder = c.Query("sort_order")

	// 调用服务层查询
	result, err := svc.OTATaskList(service.OTATaskListRequest{
		ProductKey:     req.ProductKey,
		TaskType:       req.TaskType,
		Status:         req.Status,
		StartTimeBegin: req.StartTimeBegin,
		StartTimeEnd:   req.StartTimeEnd,
		Keyword:        req.Keyword,
		Page:           req.Page,
		PageSize:       req.PageSize,
		SortBy:         req.SortBy,
		SortOrder:      req.SortOrder,
	})

	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询任务列表失败")
		return
	}

	e.OK(result, "查询成功")
}

// OTATaskCancelReq OTA 任务取消请求
type OTATaskCancelReq struct {
	TaskID     int64  `json:"task_id"`
	Confirm    bool   `json:"confirm"`
	Reason     string `json:"reason"`
	CancelType string `json:"cancel_type"` // all: 全部取消，pending_only: 仅取消待下发
}

// OTATaskCancel OTA 任务取消
// @Summary 取消 OTA 任务
// @Tags 平台设备 - 固件
// @Accept application/json
// @Param task_id body int true "任务 ID"
// @Param confirm body bool true "确认标识"
// @Param reason body string false "取消原因"
// @Param cancel_type body string false "取消类型：all/pending_only"
// @Router /api/v1/platform-device/ota-task/cancel [post]
// @Security Bearer
func (e PlatformDevice) OTATaskCancel(c *gin.Context) {
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 解析请求体
	var req OTATaskCancelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "请求参数格式错误")
		return
	}

	// 校验必填参数
	if req.TaskID <= 0 {
		e.Error(400, errors.New("bad request"), "task_id 必填且为正整数")
		return
	}
	if !req.Confirm {
		e.Error(400, errors.New("bad request"), "confirm 必须为 true，确认已知取消后果")
		return
	}

	// 获取操作人
	operator := user.GetUserId(c)
	if operator <= 0 {
		e.Error(401, errors.New("unauthorized"), "请先登录")
		return
	}

	// 调用服务层取消
	result, err := svc.OTATaskCancel(service.OTATaskCancelRequest{
		TaskID:     req.TaskID,
		Confirm:    req.Confirm,
		Reason:     req.Reason,
		CancelType: req.CancelType,
		Operator:   int64(operator),
	})

	if err != nil {
		switch {
		case errors.Is(err, service.ErrOTATaskNotFound):
			e.Error(404, err, "任务记录不存在")
		case errors.Is(err, service.ErrOTATaskAlreadyCompleted):
			e.Error(400, err, "任务已完成，无法取消，请创建回滚任务")
		case errors.Is(err, service.ErrOTATaskAlreadyCancelled):
			e.Error(400, err, "任务已取消，无需重复操作")
		case errors.Is(err, service.ErrOTATaskCancelFailed):
			e.Error(500, err, "取消操作失败，请重试或联系管理员")
		default:
			e.Logger.Error(err)
			e.Error(500, err, "取消失败")
		}
		return
	}

	if result == nil || !result.Success {
		msg := "取消失败"
		if result != nil && result.Message != "" {
			msg = result.Message
		}
		e.Error(500, errors.New("cancel failed"), msg)
		return
	}

	e.OK(gin.H{
		"success":          result.Success,
		"task_id":          result.TaskID,
		"task_name":        result.TaskName,
		"status":           result.Status,
		"affected_devices": result.AffectedDevices,
		"cancel_time":      result.CancelTime,
		"message":          result.Message,
	}, "任务取消成功")
}

// OTATaskProgressReq OTA 任务进度查询请求
type OTATaskProgressReq struct {
	TaskID   int64 `json:"task_id"`
	DeviceID int64 `json:"device_id"`
	Refresh  bool  `json:"refresh"`
}

// OTATaskProgress OTA 任务进度查询
// @Summary 查询 OTA 任务进度
// @Tags 平台设备 - 固件
// @Accept application/json
// @Param task_id body int false "任务 ID"
// @Param device_id body int false "设备 ID"
// @Param refresh body bool false "是否强制刷新"
// @Router /api/v1/platform-device/ota-task/progress [get]
// @Security Bearer
func (e PlatformDevice) OTATaskProgress(c *gin.Context) {
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 解析请求参数
	taskID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("task_id")), 10, 64)
	deviceID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("device_id")), 10, 64)
	refresh := strings.EqualFold(c.Query("refresh"), "true") || c.Query("refresh") == "1"

	// 校验必填参数：task_id 或 device_id 至少一个
	if taskID <= 0 && deviceID <= 0 {
		e.Error(400, errors.New("bad request"), "task_id 或 device_id 至少必填一个")
		return
	}

	// 调用服务层查询
	result, err := svc.OTATaskProgress(service.OTATaskProgressRequest{
		TaskID:   taskID,
		DeviceID: deviceID,
		Refresh:  refresh,
	})

	if err != nil {
		switch {
		case errors.Is(err, service.ErrOTATaskNotFound):
			e.Error(404, err, "任务不存在")
		case errors.Is(err, service.ErrPlatformDeviceNotFound):
			e.Error(404, err, "设备不存在")
		default:
			e.Logger.Error(err)
			e.Error(500, err, "查询进度失败")
		}
		return
	}

	e.OK(result, "查询成功")
}

// OTATaskDetail 获取 OTA 任务详情
// @Summary OTA 任务详情
// @Tags 平台设备 - 固件
// @Param task_id query int true "任务 ID"
// @Param with_devices query bool false "是否包含设备列表"
// @Param with_logs query bool false "是否包含操作日志"
// @Router /api/v1/platform-device/ota-task/detail [get]
// @Security Bearer
func (e PlatformDevice) OTATaskDetail(c *gin.Context) {
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	// 解析请求参数
	taskID, _ := strconv.ParseInt(strings.TrimSpace(c.Query("task_id")), 10, 64)
	if taskID <= 0 {
		e.Error(400, errors.New("bad request"), "task_id 必填且为正整数")
		return
	}

	withDevices := strings.EqualFold(c.Query("with_devices"), "true") || c.Query("with_devices") == "1"
	withLogs := strings.EqualFold(c.Query("with_logs"), "true") || c.Query("with_logs") == "1"

	// 调用服务层查询
	result, err := svc.OTATaskDetail(service.OTATaskDetailRequest{
		TaskID:      taskID,
		WithDevices: withDevices,
		WithLogs:    withLogs,
	})

	if err != nil {
		switch {
		case errors.Is(err, service.ErrOTATaskNotFound):
			e.Error(404, err, "任务不存在")
		default:
			e.Logger.Error(err)
			e.Error(500, err, "查询任务详情失败")
		}
		return
	}

	e.OK(result, "查询成功")
}
