package apis

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/api"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"

	"go-admin/app/admin/device/service"
)

// PlatformDevice 平台设备管理 API
type PlatformDevice struct {
	api.Api
}

func parseTimeQuery(c *gin.Context, key string) (*time.Time, error) {
	s := strings.TrimSpace(c.Query(key))
	if s == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// List 设备分页列表
// @Summary 平台设备列表
// @Tags 平台设备
// @Router /api/v1/platform-device/list [get]
// @Security Bearer
func (e PlatformDevice) List(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "服务初始化失败")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSizeStr := c.DefaultQuery("pageSize", "")
	if pageSizeStr == "" {
		pageSizeStr = c.DefaultQuery("page_size", "20")
	}
	pageSize, _ := strconv.Atoi(pageSizeStr)

	var uid int64
	if v := strings.TrimSpace(c.Query("user_id")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			uid = n
		}
	}
	var st *int16
	if v := strings.TrimSpace(c.Query("status")); v != "" {
		n, err := strconv.ParseInt(v, 10, 16)
		if err == nil {
			x := int16(n)
			st = &x
		}
	}
	var on *int16
	if v := strings.TrimSpace(c.Query("online_status")); v != "" {
		n, err := strconv.ParseInt(v, 10, 16)
		if err == nil {
			x := int16(n)
			on = &x
		}
	}
	from, err := parseTimeQuery(c, "created_from")
	if err != nil {
		e.Error(400, err, "created_from 格式应为 YYYY-MM-DD")
		return
	}
	to, err := parseTimeQuery(c, "created_to")
	if err != nil {
		e.Error(400, err, "created_to 格式应为 YYYY-MM-DD")
		return
	}

	snMode := strings.TrimSpace(c.Query("sn_mode"))
	snExact := snMode == "" || snMode == "exact"
	userQuery := strings.TrimSpace(c.Query("user_query"))
	fw := strings.TrimSpace(c.Query("firmware_version"))
	var bindSt *int16
	if v := strings.TrimSpace(c.Query("bind_status")); v != "" {
		n, err := strconv.ParseInt(v, 10, 16)
		if err == nil && (n == 0 || n == 1) {
			x := int16(n)
			bindSt = &x
		}
	}
	sortBy := strings.TrimSpace(c.Query("sort_by"))
	sortOrder := strings.TrimSpace(c.Query("sort_order"))

	f := service.PlatformDeviceListFilter{
		Sn:           strings.TrimSpace(c.Query("sn")),
		SnExact:      snExact,
		UserID:       uid,
		UserQuery:    userQuery,
		Status:       st,
		OnlineStatus: on,
		ProductKey:   strings.TrimSpace(c.Query("product_key")),
		FirmwareVer:  fw,
		BindStatus:   bindSt,
		CreatedFrom:  from,
		CreatedTo:    to,
		SortBy:       sortBy,
		SortOrder:    sortOrder,
	}
	list, total, err := s.ListDevices(page, pageSize, f)
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询失败")
		return
	}
	e.PageOK(list, int(total), page, pageSize, "查询成功")
}

// Detail 设备详情
// @Router /api/v1/platform-device/detail [get]
// @Security Bearer
func (e PlatformDevice) Detail(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var deviceID int64
	if q := strings.TrimSpace(c.Query("device_id")); q != "" {
		if v, err := strconv.ParseInt(q, 10, 64); err == nil && v > 0 {
			deviceID = v
		}
	}
	// 先读 query 再读 path：部分环境下 /platform-device/detail?sn=... 会落到 /:sn（sn=「detail」），
	// 若优先 path 会把真实 SN 覆盖成「detail」，导致「设备不存在」。
	rawQuerySn := strings.TrimSpace(c.Query("sn"))
	rawPathSn := strings.TrimSpace(c.Param("sn"))
	sn := rawQuerySn
	if sn == "" {
		sn = rawPathSn
	}
	if sn == "detail" {
		sn = ""
	}
	if deviceID <= 0 && sn == "" {
		e.Error(400, errors.New("sn or device_id required"), "请传 sn 或 device_id")
		return
	}
	out, err := s.GetDeviceDetail(sn, deviceID)
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, err.Error())
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, "查询失败")
		return
	}
	e.OK(out, "查询成功")
}

// CreateTask 创建设备定时任务
// @Summary 创建设备定时任务
// @Tags 平台设备
// @Router /api/v1/platform-device/task/create [post]
// @Security Bearer
// @Param task body service.CreateTaskIn true "任务信息"
func (e PlatformDevice) CreateTask(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.CreateTaskIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	// 自动填充操作人
	userID := user.GetUserId(c)
	if userID > 0 {
		in.Operator = c.GetString("user_name")
		if in.Operator == "" {
			in.Operator = fmt.Sprintf("user_%d", userID)
		}
	}

	if len(in.DeviceIds) == 0 {
		e.Error(400, nil, "设备列表不能为空")
		return
	}

	out, err := s.CreateTask(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "创建任务失败")
		return
	}

	e.OK(out, "任务创建成功")
}

// UpdateTask 更新设备定时任务
// @Summary 更新设备定时任务
// @Tags 平台设备
// @Router /api/v1/platform-device/task/update [post]
// @Security Bearer
// @Param task body service.UpdateTaskIn true "任务信息"
func (e PlatformDevice) UpdateTask(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.UpdateTaskIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	// 自动填充操作人
	userID := user.GetUserId(c)
	if userID > 0 {
		in.Operator = c.GetString("user_name")
		if in.Operator == "" {
			in.Operator = fmt.Sprintf("user_%d", userID)
		}
	}

	if in.TaskId <= 0 {
		e.Error(400, nil, "task_id 必须大于 0")
		return
	}

	out, err := s.UpdateTask(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "更新任务失败")
		return
	}

	e.OK(out, "任务更新成功")
}

// DeleteTask 删除设备定时任务
// @Summary 删除设备定时任务
// @Tags 平台设备
// @Router /api/v1/platform-device/task/delete [post]
// @Security Bearer
// @Param task body service.DeleteTaskIn true "删除参数"
func (e PlatformDevice) DeleteTask(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.DeleteTaskIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	// 自动填充操作人
	userID := user.GetUserId(c)
	if userID > 0 {
		in.Operator = c.GetString("user_name")
		if in.Operator == "" {
			in.Operator = fmt.Sprintf("user_%d", userID)
		}
	}

	if in.TaskId <= 0 {
		e.Error(400, nil, "task_id 必须大于 0")
		return
	}

	out, err := s.DeleteTask(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "删除任务失败")
		return
	}

	e.OK(out, "任务删除成功")
}

// ToggleTask 切换设备定时任务状态
// @Summary 切换任务状态
// @Tags 平台设备
// @Router /api/v1/platform-device/task/toggle [post]
// @Security Bearer
// @Param task body service.ToggleTaskIn true "切换状态参数"
func (e PlatformDevice) ToggleTask(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.ToggleTaskIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	// 自动填充操作人
	userID := user.GetUserId(c)
	if userID > 0 {
		in.Operator = c.GetString("user_name")
		if in.Operator == "" {
			in.Operator = fmt.Sprintf("user_%d", userID)
		}
	}

	if in.TaskId <= 0 {
		e.Error(400, nil, "task_id 必须大于 0")
		return
	}

	out, err := s.ToggleTask(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "切换任务状态失败")
		return
	}

	e.OK(out, "任务状态更新成功")
}

// GetTaskList 获取设备定时任务列表
// @Summary 任务列表
// @Tags 平台设备
// @Router /api/v1/platform-device/task/list [get]
// @Security Bearer
// @Param task_type query string false "任务类型 (single/cron)"
// @Param status query int false "状态 (0-禁用 1-启用)"
// @Param keyword query string false "任务名称关键词"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
func (e PlatformDevice) GetTaskList(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	taskType := c.Query("task_type")
	statusStr := c.Query("status")
	var status *int16
	if statusStr != "" {
		n, err := strconv.ParseInt(statusStr, 10, 16)
		if err == nil {
			x := int16(n)
			status = &x
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	in := &service.GetTaskListIn{
		TaskType: taskType,
		Status:   status,
		Keyword:  c.Query("keyword"),
		Page:     page,
		PageSize: pageSize,
	}

	out, err := s.GetTaskList(in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询任务列表失败")
		return
	}

	e.OK(out, "查询成功")
}

// GetTaskDetail 获取设备定时任务详情
// @Summary 任务详情
// @Tags 平台设备
// @Router /api/v1/platform-device/task/detail [get]
// @Security Bearer
// @Param task_id query int true "任务 ID"
func (e PlatformDevice) GetTaskDetail(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	taskIdStr := c.Query("task_id")
	if taskIdStr == "" {
		e.Error(400, nil, "task_id 不能为空")
		return
	}

	taskId, err := strconv.Atoi(taskIdStr)
	if err != nil {
		e.Error(400, err, "task_id 格式错误")
		return
	}

	if taskId <= 0 {
		e.Error(400, nil, "task_id 必须大于 0")
		return
	}

	in := &service.GetTaskDetailIn{
		TaskId: taskId,
	}

	out, err := s.GetTaskDetail(in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "查询任务详情失败")
		return
	}

	e.OK(out, "查询成功")
}

// GetTaskExecLog 获取设备定时任务执行日志
// @Summary 任务执行日志
// @Tags 平台设备
// @Router /api/v1/platform-device/task/log [get]
// @Security Bearer
// @Param task_id query int false "任务 ID"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
func (e PlatformDevice) GetTaskExecLog(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	taskIdStr := c.Query("task_id")
	var taskId int
	if taskIdStr != "" {
		var err error
		taskId, err = strconv.Atoi(taskIdStr)
		if err != nil {
			e.Error(400, err, "task_id 格式错误")
			return
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	in := &service.GetTaskExecLogIn{
		TaskId:   taskId,
		Page:     page,
		PageSize: pageSize,
	}

	out, err := s.GetTaskExecLog(in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询任务日志失败")
		return
	}

	e.OK(out, "查询成功")
}

// CreateShare 创建设备共享
// @Summary 创建设备共享
// @Tags 平台设备
// @Router /api/v1/platform-device/share/create [post]
// @Security Bearer
// @Param share body service.CreateShareIn true "共享信息"
func (e PlatformDevice) CreateShare(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.CreateShareIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	// 自动填充操作人
	userID := user.GetUserId(c)
	if userID > 0 {
		in.Operator = c.GetString("user_name")
		if in.Operator == "" {
			in.Operator = fmt.Sprintf("user_%d", userID)
		}
	}

	if in.DeviceId <= 0 {
		e.Error(400, nil, "device_id 必须大于 0")
		return
	}

	if strings.TrimSpace(in.SharedUser) == "" {
		e.Error(400, nil, "shared_user 不能为空")
		return
	}

	out, err := s.CreateShare(&in)
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, "设备不存在")
			return
		}
		if errors.Is(err, service.ErrPlatformDeviceInvalid) {
			e.Error(400, err, "参数无效")
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, "创建共享失败")
		return
	}

	e.OK(out, "共享创建成功")
}

// CancelShare 取消设备共享
// @Summary 取消设备共享
// @Tags 平台设备
// @Router /api/v1/platform-device/share/cancel [post]
// @Security Bearer
// @Param share body service.CancelShareIn true "取消共享参数"
func (e PlatformDevice) CancelShare(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.CancelShareIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	// 自动填充操作人
	userID := user.GetUserId(c)
	if userID > 0 {
		in.Operator = c.GetString("user_name")
		if in.Operator == "" {
			in.Operator = fmt.Sprintf("user_%d", userID)
		}
	}

	if in.ShareId <= 0 {
		e.Error(400, nil, "share_id 必须大于 0")
		return
	}

	out, err := s.CancelShare(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "取消共享失败")
		return
	}

	e.OK(out, "共享已取消")
}

// ConfirmShare 确认设备共享
// @Summary 确认设备共享
// @Tags 平台设备
// @Router /api/v1/platform-device/share/confirm [post]
// @Security Bearer
// @Param share body service.ConfirmShareIn true "确认共享参数"
func (e PlatformDevice) ConfirmShare(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.ConfirmShareIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	// 自动填充操作人和用户 ID
	userID := user.GetUserId(c)
	if userID > 0 {
		in.UserId = int64(userID)
		in.Operator = c.GetString("user_name")
		if in.Operator == "" {
			in.Operator = fmt.Sprintf("user_%d", userID)
		}
	}

	if in.ShareId <= 0 {
		e.Error(400, nil, "share_id 必须大于 0")
		return
	}

	out, err := s.ConfirmShare(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "确认共享失败")
		return
	}

	e.OK(out, "共享已确认")
}

// GetShareList 获取设备共享列表
// @Summary 设备共享列表
// @Tags 平台设备
// @Router /api/v1/platform-device/share/list [get]
// @Security Bearer
// @Param role query string false "角色 (owner: 我共享给他人的 / shared: 他人共享给我的)"
// @Param status query int false "状态 (0-待确认 1-已生效 2-已取消 3-已过期)"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
func (e PlatformDevice) GetShareList(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	userID := user.GetUserId(c)
	if userID <= 0 {
		e.Error(401, nil, "未登录")
		return
	}

	role := c.Query("role")
	statusStr := c.Query("status")
	var status *int16
	if statusStr != "" {
		n, err := strconv.ParseInt(statusStr, 10, 16)
		if err == nil {
			x := int16(n)
			status = &x
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	in := &service.GetShareListIn{
		UserId:   int64(userID),
		Role:     role,
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	}

	out, err := s.GetShareList(in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询共享列表失败")
		return
	}

	e.OK(out, "查询成功")
}

// GetShareDetail 获取共享详情
// @Summary 共享详情
// @Tags 平台设备
// @Router /api/v1/platform-device/share/detail [get]
// @Security Bearer
// @Param share_id query int true "共享 ID"
func (e PlatformDevice) GetShareDetail(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	shareIdStr := c.Query("share_id")
	if shareIdStr == "" {
		e.Error(400, nil, "share_id 不能为空")
		return
	}

	shareId, err := strconv.Atoi(shareIdStr)
	if err != nil {
		e.Error(400, err, "share_id 格式错误")
		return
	}

	if shareId <= 0 {
		e.Error(400, nil, "share_id 必须大于 0")
		return
	}

	in := &service.GetShareDetailIn{
		ShareId: shareId,
	}

	out, err := s.GetShareDetail(in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "查询共享详情失败")
		return
	}

	e.OK(out, "查询成功")
}

// GetShareLogList 获取共享日志列表
// @Summary 共享日志列表
// @Tags 平台设备
// @Router /api/v1/platform-device/share/log [get]
// @Security Bearer
// @Param share_id query int false "共享 ID"
// @Param device_id query int64 false "设备 ID"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
func (e PlatformDevice) GetShareLogList(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	shareIdStr := c.Query("share_id")
	deviceIdStr := c.Query("device_id")

	var shareId int
	var deviceId int64
	var err error

	if shareIdStr != "" {
		shareId, err = strconv.Atoi(shareIdStr)
		if err != nil {
			e.Error(400, err, "share_id 格式错误")
			return
		}
	}

	if deviceIdStr != "" {
		deviceId, err = strconv.ParseInt(deviceIdStr, 10, 64)
		if err != nil {
			e.Error(400, err, "device_id 格式错误")
			return
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	in := &service.GetShareLogListIn{
		ShareId:  shareId,
		DeviceId: deviceId,
		Page:     page,
		PageSize: pageSize,
	}

	out, err := s.GetShareLogList(in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询共享日志失败")
		return
	}

	e.OK(out, "查询成功")
}

// CreateScene 创建场景联动规则
// @Summary 创建场景联动规则
// @Tags 平台设备
// @Router /api/v1/platform-device/scene/create [post]
// @Security Bearer
// @Param scene body service.CreateSceneIn true "场景信息"
func (e PlatformDevice) CreateScene(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.CreateSceneIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	// 自动填充操作人
	userID := user.GetUserId(c)
	if userID > 0 {
		in.Operator = c.GetString("user_name")
		if in.Operator == "" {
			in.Operator = fmt.Sprintf("user_%d", userID)
		}
	}

	out, err := s.CreateScene(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "创建场景失败")
		return
	}

	e.OK(out, "场景创建成功")
}

// UpdateScene 更新场景联动规则
// @Summary 更新场景联动规则
// @Tags 平台设备
// @Router /api/v1/platform-device/scene/update [post]
// @Security Bearer
// @Param scene body service.UpdateSceneIn true "场景信息"
func (e PlatformDevice) UpdateScene(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.UpdateSceneIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	out, err := s.UpdateScene(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "更新场景失败")
		return
	}

	e.OK(out, "场景更新成功")
}

// DeleteScene 删除场景联动规则
// @Summary 删除场景联动规则
// @Tags 平台设备
// @Router /api/v1/platform-device/scene/delete [post]
// @Security Bearer
// @Param scene body service.DeleteSceneIn true "场景 ID"
func (e PlatformDevice) DeleteScene(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.DeleteSceneIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	out, err := s.DeleteScene(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "删除场景失败")
		return
	}

	e.OK(out, "场景删除成功")
}

// GetSceneList 获取场景联动规则列表
// @Summary 获取场景联动规则列表
// @Tags 平台设备
// @Router /api/v1/platform-device/scene/list [get]
// @Security Bearer
// @Param trigger_type query string false "触发类型"
// @Param status query int16 false "状态"
// @Param keyword query string false "搜索关键字"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
func (e PlatformDevice) GetSceneList(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	triggerType := c.Query("trigger_type")
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var status *int16
	if statusStr := c.Query("status"); statusStr != "" {
		s, err := strconv.ParseInt(statusStr, 10, 16)
		if err == nil {
			s16 := int16(s)
			status = &s16
		}
	}

	in := &service.GetSceneListIn{
		TriggerType: triggerType,
		Status:      status,
		Keyword:     keyword,
		Page:        page,
		PageSize:    pageSize,
	}

	out, err := s.GetSceneList(in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询场景列表失败")
		return
	}

	e.OK(out, "查询成功")
}

// GetSceneDetail 获取场景详情
// @Summary 获取场景详情
// @Tags 平台设备
// @Router /api/v1/platform-device/scene/detail [get]
// @Security Bearer
// @Param scene_id query int true "场景 ID"
func (e PlatformDevice) GetSceneDetail(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	sceneIdStr := c.Query("scene_id")
	if sceneIdStr == "" {
		e.Error(400, nil, "scene_id 不能为空")
		return
	}

	sceneId, err := strconv.Atoi(sceneIdStr)
	if err != nil {
		e.Error(400, err, "scene_id 格式错误")
		return
	}

	in := &service.GetSceneDetailIn{
		SceneId: sceneId,
	}

	out, err := s.GetSceneDetail(in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "查询场景详情失败")
		return
	}

	e.OK(out, "查询成功")
}

// ExecuteScene 执行场景联动
// @Summary 执行场景联动
// @Tags 平台设备
// @Router /api/v1/platform-device/scene/execute [post]
// @Security Bearer
// @Param execute body service.ExecuteSceneIn true "执行参数"
func (e PlatformDevice) ExecuteScene(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.ExecuteSceneIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	if in.TriggerSrc == "" {
		in.TriggerSrc = "manual"
	}

	out, err := s.ExecuteScene(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "执行场景失败")
		return
	}

	e.OK(out, "场景执行中")
}

// GetSceneExecLog 获取场景执行日志
// @Summary 获取场景执行日志
// @Tags 平台设备
// @Router /api/v1/platform-device/scene/exec-log [get]
// @Security Bearer
// @Param scene_id query int true "场景 ID"
// @Param start_time query string false "开始时间 (RFC3339 格式)"
// @Param end_time query string false "结束时间 (RFC3339 格式)"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
func (e PlatformDevice) GetSceneExecLog(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	sceneIdStr := c.Query("scene_id")
	if sceneIdStr == "" {
		e.Error(400, nil, "scene_id 不能为空")
		return
	}

	sceneId, err := strconv.Atoi(sceneIdStr)
	if err != nil {
		e.Error(400, err, "scene_id 格式错误")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var startTime, endTime time.Time
	if st := c.Query("start_time"); st != "" {
		startTime, _ = time.Parse(time.RFC3339, st)
	}
	if et := c.Query("end_time"); et != "" {
		endTime, _ = time.Parse(time.RFC3339, et)
	}

	in := &service.GetSceneExecLogIn{
		SceneId:   sceneId,
		StartTime: startTime,
		EndTime:   endTime,
		Page:      page,
		PageSize:  pageSize,
	}

	out, err := s.GetSceneExecLog(in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询执行日志失败")
		return
	}

	e.OK(out, "查询成功")
}

// CreateGroup 创建设备分组
// @Summary 创建设备分组
// @Tags 平台设备
// @Router /api/v1/platform-device/group/create [post]
// @Security Bearer
// @Param group body service.CreateGroupIn true "分组信息"
func (e PlatformDevice) CreateGroup(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.CreateGroupIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	// 自动填充操作人
	userID := user.GetUserId(c)
	if userID > 0 {
		in.Operator = c.GetString("user_name")
		if in.Operator == "" {
			in.Operator = fmt.Sprintf("user_%d", userID)
		}
	}

	out, err := s.CreateGroup(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "创建分组失败")
		return
	}

	e.OK(out, "分组创建成功")
}

// UpdateGroup 更新设备分组
// @Summary 更新设备分组
// @Tags 平台设备
// @Router /api/v1/platform-device/group/update [post]
// @Security Bearer
// @Param group body service.UpdateGroupIn true "分组信息"
func (e PlatformDevice) UpdateGroup(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.UpdateGroupIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	out, err := s.UpdateGroup(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "更新分组失败")
		return
	}

	e.OK(out, "分组更新成功")
}

// DeleteGroup 删除设备分组
// @Summary 删除设备分组
// @Tags 平台设备
// @Router /api/v1/platform-device/group/delete [post]
// @Security Bearer
// @Param group body service.DeleteGroupIn true "分组 ID"
func (e PlatformDevice) DeleteGroup(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.DeleteGroupIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	out, err := s.DeleteGroup(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "删除分组失败")
		return
	}

	e.OK(out, "分组删除成功")
}

// BatchManageDevices 批量管理设备分组
// @Summary 批量管理设备分组
// @Tags 平台设备
// @Router /api/v1/platform-device/group/batch [post]
// @Security Bearer
// @Param devices body service.BatchManageDevicesIn true "设备列表和操作类型"
func (e PlatformDevice) BatchManageDevices(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.BatchManageDevicesIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	// 自动填充操作人
	userID := user.GetUserId(c)
	if userID > 0 {
		in.Operator = c.GetString("user_name")
		if in.Operator == "" {
			in.Operator = fmt.Sprintf("user_%d", userID)
		}
	}

	out, err := s.BatchManageDevices(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "批量操作失败")
		return
	}

	e.OK(out, out.Message)
}

// GetGroupList 获取设备分组列表
// @Summary 获取设备分组列表
// @Tags 平台设备
// @Router /api/v1/platform-device/group/list [get]
// @Security Bearer
// @Param parent_id query int false "父分组 ID"
// @Param status query int16 false "状态"
// @Param keyword query string false "搜索关键字"
func (e PlatformDevice) GetGroupList(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	parentIdStr := c.Query("parent_id")
	statusStr := c.Query("status")
	keyword := c.Query("keyword")

	var parentId *int
	if parentIdStr != "" {
		pid, err := strconv.Atoi(parentIdStr)
		if err == nil {
			parentId = &pid
		}
	}

	var status *int16
	if statusStr != "" {
		s, err := strconv.ParseInt(statusStr, 10, 16)
		if err == nil {
			s16 := int16(s)
			status = &s16
		}
	}

	in := &service.GetGroupListIn{
		ParentId: parentId,
		Status:   status,
		Keyword:  keyword,
	}

	out, err := s.GetGroupList(in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询分组列表失败")
		return
	}

	e.OK(out, "查询成功")
}

// GetGroupDetail 获取分组详情
// @Summary 获取分组详情
// @Tags 平台设备
// @Router /api/v1/platform-device/group/detail [get]
// @Security Bearer
// @Param group_id query int true "分组 ID"
func (e PlatformDevice) GetGroupDetail(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	groupIdStr := c.Query("group_id")
	if groupIdStr == "" {
		e.Error(400, nil, "group_id 不能为空")
		return
	}

	groupId, err := strconv.Atoi(groupIdStr)
	if err != nil {
		e.Error(400, err, "group_id 格式错误")
		return
	}

	in := &service.GetGroupDetailIn{
		GroupId: groupId,
	}

	out, err := s.GetGroupDetail(in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "查询分组详情失败")
		return
	}

	e.OK(out, "查询成功")
}

// StartDiagnosis 开始设备诊断
// @Summary 开始设备诊断
// @Tags 平台设备
// @Router /api/v1/platform-device/diagnosis/start [post]
// @Security Bearer
// @Param diagnosis body service.StartDiagnosisIn true "诊断信息"
func (e PlatformDevice) StartDiagnosis(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.StartDiagnosisIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	if in.DeviceID <= 0 && strings.TrimSpace(in.Sn) == "" {
		e.Error(400, nil, "device_id 或 sn 必须提供一个")
		return
	}

	// 自动填充操作人
	userID := user.GetUserId(c)
	if userID > 0 {
		in.Operator = c.GetString("user_name")
		if in.Operator == "" {
			in.Operator = fmt.Sprintf("user_%d", userID)
		}
	}
	in.IpAddress = c.ClientIP()

	out, err := s.StartDiagnosis(&in)
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, "设备不存在")
			return
		}
		if errors.Is(err, service.ErrPlatformDeviceInvalid) {
			e.Error(400, err, "参数无效")
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, "开始诊断失败")
		return
	}

	e.OK(out, "诊断已开始")
}

// ReportDiagnosis 上报诊断结果
// @Summary 上报诊断结果
// @Tags 平台设备
// @Router /api/v1/platform-device/diagnosis/report [post]
// @Security Bearer
// @Param diagnosis body service.ReportDiagnosisIn true "诊断结果"
func (e PlatformDevice) ReportDiagnosis(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.ReportDiagnosisIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	out, err := s.ReportDiagnosis(&in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "上报诊断结果失败")
		return
	}

	e.OK(out, "诊断结果已保存")
}

// GetDiagnosisResult 获取诊断结果
// @Summary 获取诊断结果
// @Tags 平台设备
// @Router /api/v1/platform-device/diagnosis/result [get]
// @Security Bearer
// @Param diagnosis_id query int true "诊断记录 ID"
func (e PlatformDevice) GetDiagnosisResult(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	diagnosisIdStr := c.Query("diagnosis_id")
	if diagnosisIdStr == "" {
		e.Error(400, nil, "diagnosis_id 不能为空")
		return
	}

	diagnosisId, err := strconv.Atoi(diagnosisIdStr)
	if err != nil {
		e.Error(400, err, "diagnosis_id 格式错误")
		return
	}

	in := &service.GetDiagnosisResultIn{
		DiagnosisId: diagnosisId,
	}

	out, err := s.GetDiagnosisResult(in)
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询诊断结果失败")
		return
	}

	e.OK(out, "查询成功")
}

// GetDiagnosisHistory 设备诊断历史
// @Summary 设备诊断历史
// @Tags 平台设备
// @Router /api/v1/platform-device/diagnosis/history [get]
// @Security Bearer
// @Param device_id query int64 false "设备 ID"
// @Param sn query string false "设备 SN"
// @Param start_time query string false "开始时间 (RFC3339 格式)"
// @Param end_time query string false "结束时间 (RFC3339 格式)"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
func (e PlatformDevice) GetDiagnosisHistory(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	deviceIDStr := c.Query("device_id")
	sn := strings.TrimSpace(c.Query("sn"))

	var deviceID int64
	if deviceIDStr != "" {
		var err error
		deviceID, err = strconv.ParseInt(deviceIDStr, 10, 64)
		if err != nil {
			e.Error(400, err, "device_id 格式错误")
			return
		}
	}

	if deviceID <= 0 && sn == "" {
		e.Error(400, nil, "device_id 或 sn 必须提供一个")
		return
	}

	// 解析时间
	var startTime, endTime time.Time
	if st := c.Query("start_time"); st != "" {
		var err error
		startTime, err = time.Parse(time.RFC3339, st)
		if err != nil {
			e.Error(400, err, "start_time 格式错误，应为 RFC3339 格式")
			return
		}
	}
	if et := c.Query("end_time"); et != "" {
		var err error
		endTime, err = time.Parse(time.RFC3339, et)
		if err != nil {
			e.Error(400, err, "end_time 格式错误，应为 RFC3339 格式")
			return
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	in := &service.GetDiagnosisHistoryIn{
		DeviceID:  deviceID,
		Sn:        sn,
		StartTime: startTime,
		EndTime:   endTime,
		Page:      page,
		PageSize:  pageSize,
	}

	out, err := s.GetDiagnosisHistory(in)
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, "设备不存在")
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, "查询诊断历史失败")
		return
	}

	e.OK(out, "查询成功")
}

// ActivateCloudAuth 云端认证激活（HMAC + 时间窗 + nonce 防重放）→ 未激活(3) 转为正常(1)
// @Summary 云端认证激活设备
// @Tags 平台设备
// @Router /api/v1/platform-device/activate-cloud [post]
// @Security Bearer
func (e PlatformDevice) ActivateCloudAuth(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var in service.CloudActivateIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	op := strconv.Itoa(user.GetUserId(c))
	out, err := s.ActivateDeviceCloudAuth(&in, op)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPlatformDeviceNotFound):
			e.Error(404, err, err.Error())
		case errors.Is(err, service.ErrPlatformDeviceInvalid):
			e.Error(400, err, err.Error())
		case errors.Is(err, service.ErrProductKeyNotActive),
			errors.Is(err, service.ErrProductDisabledOrDraft),
			errors.Is(err, service.ErrDeviceCloudProductKeyMismatch),
			errors.Is(err, service.ErrDeviceCloudNotInactive),
			errors.Is(err, service.ErrDeviceCloudTimestampInvalid),
			errors.Is(err, service.ErrDeviceCloudNonceReplay),
			errors.Is(err, service.ErrDeviceCloudSignatureInvalid),
			errors.Is(err, service.ErrDeviceCloudSecretInvalid):
			e.Error(400, err, err.Error())
		default:
			e.Logger.Error(err)
			e.Error(500, err, "激活失败")
		}
		return
	}
	e.OK(out, "云端认证成功，设备已激活")
}

type platformDeviceActivateCloudTrustedReq struct {
	Sn         string `json:"sn" binding:"required"`
	ProductKey string `json:"product_key" binding:"required"`
}

// ActivateCloudTrusted 后台「云端认证激活」无需填写出厂密钥与 HMAC：依赖已登录管理员身份，将未激活设备置为正常（与设备侧自主 HMAC 接口二选一）。
// @Summary 云端认证激活（管理员可信）
// @Tags 平台设备
// @Router /api/v1/platform-device/activate-cloud-admin [post]
// @Security Bearer
func (e PlatformDevice) ActivateCloudTrusted(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var req platformDeviceActivateCloudTrustedReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	op := strconv.Itoa(user.GetUserId(c))
	out, err := s.AdminTrustedCloudActivate(req.Sn, req.ProductKey, op)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPlatformDeviceNotFound):
			e.Error(404, err, err.Error())
		case errors.Is(err, service.ErrPlatformDeviceInvalid):
			e.Error(400, err, err.Error())
		case errors.Is(err, service.ErrProductKeyNotActive),
			errors.Is(err, service.ErrProductDisabledOrDraft),
			errors.Is(err, service.ErrDeviceCloudProductKeyMismatch),
			errors.Is(err, service.ErrDeviceCloudNotInactive):
			e.Error(400, err, err.Error())
		default:
			e.Logger.Error(err)
			e.Error(500, err, "激活失败")
		}
		return
	}
	e.OK(out, "激活成功，设备状态已为「正常」")
}

type platformDeviceStatusReq struct {
	Sn     string `json:"sn" binding:"required"`
	Status int16  `json:"status" binding:"required"`
}

// SetStatus 禁用/启用/报废
// @Router /api/v1/platform-device/status [post]
// @Security Bearer
func (e PlatformDevice) SetStatus(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var req platformDeviceStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	op := strconv.Itoa(user.GetUserId(c))
	if err := s.SetDeviceStatus(req.Sn, req.Status, op); err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, err.Error())
			return
		}
		e.Error(400, err, err.Error())
		return
	}
	e.OK(nil, "操作成功")
}

type platformDeviceUnbindReq struct {
	Sn string `json:"sn" binding:"required"`
}

// Unbind 强制解绑
// @Router /api/v1/platform-device/unbind [post]
// @Security Bearer
func (e PlatformDevice) Unbind(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var req platformDeviceUnbindReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	op := strconv.Itoa(user.GetUserId(c))
	if err := s.AdminUnbind(req.Sn, op); err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, err.Error())
			return
		}
		e.Error(400, err, err.Error())
		return
	}
	e.OK(nil, "解绑成功")
}

type platformForceUnbindReq struct {
	DeviceID int64  `json:"device_id"`
	Sn       string `json:"sn"`
	Reason   string `json:"reason"`
	Confirm  bool   `json:"confirm"`
}

// ForceUnbind 后台强制解绑（device_id 或 sn + 二次确认 + 幂等）
// @Router /api/v1/platform-device/force-unbind [post]
// @Security Bearer
func (e PlatformDevice) ForceUnbind(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var req platformForceUnbindReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	if req.DeviceID <= 0 && strings.TrimSpace(req.Sn) == "" {
		e.Error(400, errors.New("device_id or sn required"), "请提供 device_id 或 sn")
		return
	}
	op := strconv.Itoa(user.GetUserId(c))
	out, err := s.AdminForceUnbind(&service.AdminForceUnbindIn{
		DeviceID: req.DeviceID,
		Sn:       req.Sn,
		Reason:   req.Reason,
		Confirm:  req.Confirm,
		Operator: op,
	})
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, err.Error())
			return
		}
		if errors.Is(err, service.ErrPlatformDeviceInvalid) {
			e.Error(400, err, err.Error())
			return
		}
		msg := err.Error()
		if strings.Contains(msg, "请确认强制解绑") {
			e.Error(400, err, msg)
			return
		}
		e.Error(400, err, msg)
		return
	}
	e.OK(out, "强制解绑成功")
}

type platformDeviceCommandReq struct {
	Sn      string                 `json:"sn" binding:"required"`
	Command string                 `json:"command" binding:"required"`
	Params  map[string]interface{} `json:"params"`
}

type platformRemoteCommandReq struct {
	DeviceID            int64  `json:"device_id"`
	Sn                  string `json:"sn"`
	Command             string `json:"command" binding:"required"`
	Reason              string `json:"reason"`
	ConfirmFactoryReset bool   `json:"confirm_factory_reset"`
}

// RemoteCommand 后台远程指令（重启 / 恢复出厂 / 诊断 / 立即上报状态）：落库 + MQTT + 审计
// @Summary 后台远程指令
// @Description command：restart、factory_reset、diagnosis、report_status（立即上报状态，MQTT device/{sn}/command）。factory_reset 需 confirm_factory_reset=true。
// @Tags 平台设备
// @Router /api/v1/platform-device/remote-command [post]
// @Security Bearer
func (e PlatformDevice) RemoteCommand(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var req platformRemoteCommandReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	if req.DeviceID <= 0 && strings.TrimSpace(req.Sn) == "" {
		e.Error(400, errors.New("device_id or sn required"), "请提供 device_id 或 sn")
		return
	}
	op := strconv.Itoa(user.GetUserId(c))
	out, err := s.AdminRemoteCommand(&service.AdminRemoteCommandIn{
		DeviceID:            req.DeviceID,
		Sn:                  req.Sn,
		Command:             req.Command,
		Reason:              req.Reason,
		ConfirmFactoryReset: req.ConfirmFactoryReset,
		Operator:            op,
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
		msg := err.Error()
		if strings.Contains(msg, "MQTT") {
			e.Error(500, err, msg)
			return
		}
		e.Error(400, err, msg)
		return
	}
	e.OK(out, "指令已下发")
}

type platformReportStatusReq struct {
	DeviceID int64  `json:"device_id"`
	Sn       string `json:"sn"`
	Reason   string `json:"reason"`
}

// ReportStatusTrigger 手动触发设备立即上报状态（内部 MQTT 下发 cmd=report_status；设备收到后采集并调用 device 服务 POST /api/device/status/report，report_type=manual）
// @Summary 手动触发设备状态上报
// @Description 需登录；与 POST /remote-command 且 command=report_status 等价，语义更清晰。
// @Tags 平台设备
// @Accept json
// @Produce json
// @Param body body platformReportStatusReq true "device_id 与 sn 二选一"
// @Router /api/v1/platform-device/report-status [post]
// @Security Bearer
func (e PlatformDevice) ReportStatusTrigger(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var req platformReportStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	if req.DeviceID <= 0 && strings.TrimSpace(req.Sn) == "" {
		e.Error(400, errors.New("device_id or sn required"), "请提供 device_id 或 sn")
		return
	}
	op := strconv.Itoa(user.GetUserId(c))
	out, err := s.AdminRemoteCommand(&service.AdminRemoteCommandIn{
		DeviceID: req.DeviceID,
		Sn:       req.Sn,
		Command:  "report_status",
		Reason:   strings.TrimSpace(req.Reason),
		Operator: op,
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
		msg := err.Error()
		if strings.Contains(msg, "MQTT") {
			e.Error(500, err, msg)
			return
		}
		e.Error(400, err, msg)
		return
	}
	e.OK(out, "已下发立即上报指令")
}

// Command 远程指令
// @Router /api/v1/platform-device/command [post]
// @Security Bearer
func (e PlatformDevice) Command(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var req platformDeviceCommandReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	op := strconv.Itoa(user.GetUserId(c))
	if err := s.EnqueueCommand(req.Sn, req.Command, req.Params, op); err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, err.Error())
			return
		}
		if errors.Is(err, service.ErrPlatformDeviceInvalid) {
			e.Error(400, err, err.Error())
			return
		}
		e.Error(500, err, err.Error())
		return
	}
	e.OK(nil, "指令已入队")
}

type platformDeviceOTAReq struct {
	Sn      string `json:"sn" binding:"required"`
	Version string `json:"version" binding:"required"`
}

// OTA 推送升级任务
// @Router /api/v1/platform-device/ota [post]
// @Security Bearer
func (e PlatformDevice) OTA(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var req platformDeviceOTAReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	op := strconv.Itoa(user.GetUserId(c))
	if err := s.EnqueueOTA(req.Sn, req.Version, op); err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, err.Error())
			return
		}
		e.Error(400, err, err.Error())
		return
	}
	e.OK(nil, "OTA 任务已创建")
}

// Summary 看板统计
// @Router /api/v1/platform-device/summary [get]
// @Security Bearer
func (e PlatformDevice) Summary(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	out, err := s.GetSummary()
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询失败")
		return
	}
	e.OK(out, "查询成功")
}

// ProductKeys 产品 Key 下拉
// @Router /api/v1/platform-device/product-keys [get]
// @Security Bearer
func (e PlatformDevice) ProductKeys(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	keys, err := s.ListProductKeys()
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询失败")
		return
	}
	e.OK(keys, "查询成功")
}

// Enum 枚举数据（与标准路径 /api/v1/device/enum 对齐：下拉选项）
// @Router /api/v1/device/enum [get]
// @Security Bearer
func (e PlatformDevice) Enum(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	keys, err := s.ListProductKeys()
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询失败")
		return
	}
	e.OK(gin.H{
		"product_keys": keys,
		"device_status": []gin.H{
			{"value": 1, "label": "正常"},
			{"value": 2, "label": "禁用"},
			{"value": 3, "label": "未激活"},
			{"value": 4, "label": "报废"},
		},
		"online_status": []gin.H{
			{"value": 1, "label": "在线"},
			{"value": 0, "label": "离线"},
		},
		"bind_status": []gin.H{
			{"value": 1, "label": "已绑定"},
			{"value": 0, "label": "未绑定"},
		},
	}, "查询成功")
}

type platformDeviceCreateReq struct {
	Sn           string `json:"sn" binding:"required"`
	ProductKey   string `json:"product_key" binding:"required"`
	Model        string `json:"model"`
	Mac          string `json:"mac"` // 可选；填写时须为合法 MAC，并与 RequireMAC 校验一致
	DeviceName   string `json:"device_name"`
	Remark       string `json:"remark"`
	PresetSecret string `json:"preset_secret"`
}

// Create 预注册设备
// @Summary 预注册设备（返回明文密钥仅一次）
// @Router /api/v1/platform-device [post]
// @Security Bearer
func (e PlatformDevice) Create(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var req platformDeviceCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	requireMAC := strings.TrimSpace(req.Mac) != ""
	out, err := s.RegisterDeviceWithOptions(&service.ProvisionIn{
		Sn:                req.Sn,
		ProductKey:        req.ProductKey,
		Model:             req.Model,
		Mac:               req.Mac,
		AdminDisplayName:  req.DeviceName,
		AdminRemark:       req.Remark,
		PlainPresetSecret: req.PresetSecret,
		RequireMAC:        requireMAC,
		CreateBy:          user.GetUserId(c),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPlatformDeviceDuplicate), errors.Is(err, service.ErrPlatformDeviceInvalid),
			errors.Is(err, service.ErrProductKeyNotActive), errors.Is(err, service.ErrPlatformDeviceMacDuplicate),
			errors.Is(err, service.ErrPlatformDeviceSNFormat), errors.Is(err, service.ErrPlatformDeviceMACFormat):
			e.Error(400, err, err.Error())
		default:
			e.Error(500, err, err.Error())
		}
		return
	}
	e.OK(out, "创建成功")
}

type platformDeviceImportReq struct {
	Items []service.DeviceImportRow `json:"items" binding:"required"`
}

// Import 批量导入
// @Router /api/v1/platform-device/import [post]
// @Security Bearer
func (e PlatformDevice) Import(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var req platformDeviceImportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	if len(req.Items) == 0 {
		e.Error(400, errors.New("empty"), "items 不能为空")
		return
	}
	if len(req.Items) > 500 {
		e.Error(400, errors.New("limit"), "单次最多 500 条")
		return
	}
	ok, fails := s.ImportDevices(req.Items)
	e.OK(gin.H{"success": ok, "errors": fails}, "导入完成")
}

// ImportTemplate 下载批量导入模板 xlsx
// @Summary 下载设备批量导入模板
// @Router /api/v1/platform-device/import/template [get]
// @Security Bearer
func (e PlatformDevice) ImportTemplate(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	b, err := s.ImportJobTemplateXLSX()
	if err != nil {
		e.Error(500, err, "生成模板失败")
		return
	}
	c.Header("Content-Disposition", `attachment; filename="device_import_template.xlsx"`)
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", b)
}

// ImportJobCreate 上传表格异步导入
// @Summary 上传表格创建设备导入任务
// @Router /api/v1/platform-device/import/jobs [post]
// @Security Bearer
func (e PlatformDevice) ImportJobCreate(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		e.Error(400, err, "请上传 file 字段（xlsx）")
		return
	}
	if fh.Size > service.ImportMaxFileBytes() {
		e.Error(400, fmt.Errorf("file too large"), "文件过大")
		return
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".xlsx" && ext != ".xls" {
		e.Error(400, fmt.Errorf("bad ext"), "仅支持 .xlsx / .xls")
		return
	}
	out, err := os.CreateTemp("", "device-import-*"+ext)
	if err != nil {
		e.Error(500, err, "临时文件失败")
		return
	}
	tmpPath := out.Name()
	src, err := fh.Open()
	if err != nil {
		_ = out.Close()
		_ = os.Remove(tmpPath)
		e.Error(500, err, "读取上传失败")
		return
	}
	_, err = io.Copy(out, src)
	_ = src.Close()
	_ = out.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		e.Error(500, err, "保存上传失败")
		return
	}

	jobID, err := s.CreateImportJob(tmpPath, int64(user.GetUserId(c)))
	if err != nil {
		_ = os.Remove(tmpPath)
		if strings.Contains(err.Error(), "无法解析") || strings.Contains(err.Error(), "至少") || strings.Contains(err.Error(), "最多") {
			e.Error(400, err, err.Error())
			return
		}
		e.Error(500, err, err.Error())
		return
	}
	e.OK(gin.H{"job_id": jobID}, "任务已创建")
}

// ImportJobGet 查询导入任务进度
// @Summary 查询设备导入任务
// @Router /api/v1/platform-device/import/jobs/:id [get]
// @Security Bearer
func (e PlatformDevice) ImportJobGet(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		e.Error(400, err, "无效的任务 id")
		return
	}
	j, fails, err := s.GetImportJob(id, user.GetUserId(c))
	if err != nil {
		if errors.Is(err, service.ErrImportJobNotFound) {
			e.Error(404, err, err.Error())
			return
		}
		if errors.Is(err, service.ErrImportJobAccessDenied) {
			e.Error(403, err, err.Error())
			return
		}
		e.Error(500, err, err.Error())
		return
	}
	e.OK(gin.H{
		"id":            j.ID,
		"status":        j.Status,
		"total":         j.Total,
		"processed":     j.Processed,
		"success_count": j.SuccessCount,
		"fail_count":    j.FailCount,
		"error_message": j.ErrorMessage,
		"failures":      fails,
		"created_at":    j.CreatedAt,
		"finished_at":   j.FinishedAt,
	}, "查询成功")
}

// ImportJobDownload 下载成功行密钥 CSV
// @Summary 下载设备导入结果（明文密钥）
// @Router /api/v1/platform-device/import/jobs/:id/download [get]
// @Security Bearer
func (e PlatformDevice) ImportJobDownload(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		e.Error(400, err, "无效的任务 id")
		return
	}
	path, err := s.ImportResultFilePath(id, user.GetUserId(c))
	if err != nil {
		e.Error(400, err, err.Error())
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		e.Error(500, err, "读取结果失败")
		return
	}
	_ = os.Remove(path)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="device_import_%d_result.csv"`, id))
	c.Data(200, "text/csv; charset=utf-8", b)
}

type platformDeviceBatchStatusReq struct {
	Sns    []string `json:"sns" binding:"required"`
	Status int16    `json:"status" binding:"required"`
}

// BatchStatus 批量状态
// @Router /api/v1/platform-device/status/batch [post]
// @Security Bearer
func (e PlatformDevice) BatchStatus(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var req platformDeviceBatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	op := strconv.Itoa(user.GetUserId(c))
	n, err := s.BatchSetStatus(req.Sns, req.Status, op)
	if err != nil {
		e.Error(400, err, err.Error())
		return
	}
	e.OK(gin.H{"updated": n}, "操作成功")
}

type platformDeviceDeleteReq struct {
	DeviceID int64  `json:"device_id"`
	Sn       string `json:"sn"`
	Confirm  bool   `json:"confirm" binding:"required"`
}

// Delete 后台删除设备（软删除）
// @Router /api/v1/platform-device/delete [post]
// @Security Bearer
// @Summary 删除设备
// @Tags 平台设备
// @Accept application/json
// @Param data body platformDeviceDeleteReq true "删除请求"
// @Success 200 {object} object{code=int,msg=string,data=object{device_sn=string,device_id=int,success=bool}} "删除成功"
// @Failure 400 {object} object{code=int,msg=string} "参数错误/设备在线/确认缺失"
// @Failure 404 {object} object{code=int,msg=string} "设备不存在"
// @Failure 500 {object} object{code=int,msg=string} "系统异常"
func (e PlatformDevice) Delete(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	var req platformDeviceDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(400, err, "参数错误")
		return
	}
	if req.DeviceID <= 0 && strings.TrimSpace(req.Sn) == "" {
		e.Error(400, errors.New("device_id or sn required"), "请提供 device_id 或 sn")
		return
	}
	if !req.Confirm {
		e.Error(400, errors.New("confirm required"), "请确认删除设备：传 confirm=true")
		return
	}
	op := strconv.Itoa(user.GetUserId(c))
	out, err := s.AdminDeleteDevice(&service.AdminDeleteDeviceIn{
		DeviceID: req.DeviceID,
		Sn:       req.Sn,
		Confirm:  req.Confirm,
		Operator: op,
	})
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, err.Error())
			return
		}
		if errors.Is(err, service.ErrPlatformDeviceInvalid) {
			e.Error(400, err, err.Error())
			return
		}
		msg := err.Error()
		// 特殊错误处理
		if strings.Contains(msg, "请确认删除设备") {
			e.Error(400, err, msg)
			return
		}
		if strings.Contains(msg, "设备在线") {
			e.Error(400, err, msg)
			return
		}
		if strings.Contains(msg, "设备已禁用或已报废") {
			e.Error(400, err, msg)
			return
		}
		e.Error(400, err, msg)
		return
	}
	e.OK(out, "删除成功")
}

// GetDeviceConfig 获取设备配置
// @Summary 获取设备配置
// @Tags 平台设备
// @Router /api/v1/platform-device/config [get]
// @Security Bearer
// @Param device_id query int64 false "设备 ID"
// @Param sn query string false "设备 SN"
func (e PlatformDevice) GetDeviceConfig(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	deviceIDStr := c.Query("device_id")
	sn := strings.TrimSpace(c.Query("sn"))

	var deviceID int64
	if deviceIDStr != "" {
		var err error
		deviceID, err = strconv.ParseInt(deviceIDStr, 10, 64)
		if err != nil {
			e.Error(400, err, "device_id 格式错误")
			return
		}
	}

	if deviceID <= 0 && sn == "" {
		e.Error(400, nil, "device_id 或 sn 必须提供一个")
		return
	}

	in := &service.GetDeviceConfigIn{
		DeviceID: deviceID,
		Sn:       sn,
	}

	out, err := s.GetDeviceConfig(in)
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, "设备不存在")
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

	e.OK(out, "查询成功")
}

// UpdateDeviceConfig 更新设备配置
// @Summary 更新设备配置
// @Tags 平台设备
// @Router /api/v1/platform-device/config [put]
// @Security Bearer
// @Param config body service.UpdateDeviceConfigIn true "配置信息"
func (e PlatformDevice) UpdateDeviceConfig(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.UpdateDeviceConfigIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	// 自动填充操作人
	userID := user.GetUserId(c)
	if userID > 0 {
		in.Operator = c.GetString("user_name")
		if in.Operator == "" {
			in.Operator = fmt.Sprintf("user_%d", userID)
		}
	}

	if in.DeviceID <= 0 && strings.TrimSpace(in.Sn) == "" {
		e.Error(400, nil, "device_id 或 sn 必须提供一个")
		return
	}

	if len(in.Configs) == 0 {
		e.Error(400, nil, "配置参数不能为空")
		return
	}

	out, err := s.UpdateDeviceConfig(&in)
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, "设备不存在")
			return
		}
		if errors.Is(err, service.ErrPlatformDeviceInvalid) {
			e.Error(400, err, "参数无效")
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, "更新配置失败")
		return
	}

	e.OK(out, "配置更新成功")
}

// ReportDeviceLog 设备日志上报
// @Summary 设备日志上报
// @Tags 平台设备
// @Router /api/v1/platform-device/log/report [post]
// @Security Bearer
// @Param log body service.ReportDeviceLogIn true "日志信息"
func (e PlatformDevice) ReportDeviceLog(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	var in service.ReportDeviceLogIn
	if err := c.ShouldBindJSON(&in); err != nil {
		e.Error(400, err, "参数错误")
		return
	}

	if in.DeviceID <= 0 && strings.TrimSpace(in.Sn) == "" {
		e.Error(400, nil, "device_id 或 sn 必须提供一个")
		return
	}

	if len(in.Logs) == 0 {
		e.Error(400, nil, "日志列表不能为空")
		return
	}

	out, err := s.ReportDeviceLog(&in)
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, "设备不存在")
			return
		}
		if errors.Is(err, service.ErrPlatformDeviceInvalid) {
			e.Error(400, err, "参数无效")
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, "日志上报失败")
		return
	}

	e.OK(out, "日志上报成功")
}

// GetDeviceLogList 设备日志列表
// @Summary 设备日志列表
// @Tags 平台设备
// @Router /api/v1/platform-device/log/list [get]
// @Security Bearer
// @Param device_id query int64 false "设备 ID"
// @Param sn query string false "设备 SN"
// @Param log_type query string false "日志类型 (system/error/operation/status)"
// @Param log_level query string false "日志级别 (debug/info/warn/error/fatal)"
// @Param start_time query string false "开始时间 (RFC3339 格式)"
// @Param end_time query string false "结束时间 (RFC3339 格式)"
// @Param processed query int false "处理状态 (0-未处理 1-已处理 2-已忽略)"
// @Param alert_sent query int false "告警状态 (0-未发送 1-已发送)"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param sort_by query string false "排序字段"
// @Param sort_order query string false "排序方式 (asc/desc)"
func (e PlatformDevice) GetDeviceLogList(c *gin.Context) {
	s := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	deviceIDStr := c.Query("device_id")
	sn := strings.TrimSpace(c.Query("sn"))

	var deviceID int64
	if deviceIDStr != "" {
		var err error
		deviceID, err = strconv.ParseInt(deviceIDStr, 10, 64)
		if err != nil {
			e.Error(400, err, "device_id 格式错误")
			return
		}
	}

	if deviceID <= 0 && sn == "" {
		e.Error(400, nil, "device_id 或 sn 必须提供一个")
		return
	}

	// 解析时间
	var startTime, endTime time.Time
	if st := c.Query("start_time"); st != "" {
		var err error
		startTime, err = time.Parse(time.RFC3339, st)
		if err != nil {
			e.Error(400, err, "start_time 格式错误，应为 RFC3339 格式")
			return
		}
	}
	if et := c.Query("end_time"); et != "" {
		var err error
		endTime, err = time.Parse(time.RFC3339, et)
		if err != nil {
			e.Error(400, err, "end_time 格式错误，应为 RFC3339 格式")
			return
		}
	}

	// 解析可选参数
	var processed *int16
	if p := c.Query("processed"); p != "" {
		n, err := strconv.ParseInt(p, 10, 16)
		if err == nil {
			x := int16(n)
			processed = &x
		}
	}

	var alertSent *int16
	if a := c.Query("alert_sent"); a != "" {
		n, err := strconv.ParseInt(a, 10, 16)
		if err == nil {
			x := int16(n)
			alertSent = &x
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	in := &service.GetDeviceLogListIn{
		DeviceID:  deviceID,
		Sn:        sn,
		LogType:   c.Query("log_type"),
		LogLevel:  c.Query("log_level"),
		StartTime: startTime,
		EndTime:   endTime,
		Processed: processed,
		AlertSent: alertSent,
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.DefaultQuery("sort_by", "report_time"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}

	out, err := s.GetDeviceLogList(in)
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, "设备不存在")
			return
		}
		if errors.Is(err, service.ErrPlatformDeviceInvalid) {
			e.Error(400, err, "参数无效")
			return
		}
		e.Logger.Error(err)
		e.Error(500, err, "查询日志失败")
		return
	}

	e.OK(out, "查询成功")
}
