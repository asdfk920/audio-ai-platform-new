package apis

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"

	"go-admin/app/admin/device/service"
)

// InstructionExecution 单条指令执行状态全流程（指令 ID 或 sn+command_id）
// @Summary 单条指令执行状态
// @Tags 平台设备-指令
// @Param instruction_id query int false "指令主键 ID（与 sn+command_id 二选一）"
// @Param sn query string false "设备 SN（需与 command_id 同时传）"
// @Param command_id query string false "指令唯一标识 params.command_id，如 cmd_12"
// @Router /api/v1/platform-device/instruction/execution [get]
// @Security Bearer
func (e PlatformDevice) InstructionExecution(c *gin.Context) {
	var q service.InstructionExecutionQuery
	if v := strings.TrimSpace(c.Query("instruction_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			e.Error(400, errors.New("bad instruction_id"), "instruction_id 无效")
			return
		}
		q.InstructionID = id
	}
	q.Sn = strings.TrimSpace(c.Query("sn"))
	q.CommandID = strings.TrimSpace(c.Query("command_id"))

	if q.InstructionID <= 0 && (q.Sn == "" || q.CommandID == "") {
		e.Error(400, errors.New("bad query"), "请传 instruction_id，或同时传 sn 与 command_id")
		return
	}

	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	out, err := svc.GetInstructionExecution(q)
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, "指令不存在")
			return
		}
		e.Logger.Error(err)
		e.Error(400, err, err.Error())
		return
	}
	e.OK(out, "ok")
}

// InstructionList 设备指令历史分页
// @Summary 设备指令历史列表
// @Tags 平台设备-指令
// @Param page query int false "页码"
// @Param page_size query int false "每页条数"
// @Param device_id query int false "设备 ID"
// @Param sn query string false "设备 SN（模糊）"
// @Param sn_mode query string false "exact 精确匹配 SN"
// @Param cmd query string false "指令类型（模糊）"
// @Param status query string false "pending|executing|success|failed|timeout|cancelled|all"
// @Param created_from query string false "创建时间起 YYYY-MM-DD"
// @Param created_to query string false "创建时间止 YYYY-MM-DD"
// @Router /api/v1/platform-device/instructions [get]
// @Security Bearer
func (e PlatformDevice) InstructionList(c *gin.Context) {
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var deviceID int64
	if v := strings.TrimSpace(c.Query("device_id")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			deviceID = n
		}
	}
	sn := strings.TrimSpace(c.Query("sn"))
	snExact := strings.TrimSpace(c.Query("sn_mode")) == "exact"
	cmd := strings.TrimSpace(c.Query("cmd"))
	status := strings.TrimSpace(c.Query("status"))
	if !service.InstructionStatusFilterOK(status) {
		e.Error(400, errors.New("bad status"), "status 取值: pending|executing|success|failed|timeout|cancelled|all")
		return
	}

	var createdFrom, createdTo *time.Time
	if s := strings.TrimSpace(c.Query("created_from")); s != "" {
		t, err := time.ParseInLocation("2006-01-02", s, time.Local)
		if err != nil {
			e.Error(400, err, "created_from 格式 YYYY-MM-DD")
			return
		}
		createdFrom = &t
	}
	if s := strings.TrimSpace(c.Query("created_to")); s != "" {
		t, err := time.ParseInLocation("2006-01-02", s, time.Local)
		if err != nil {
			e.Error(400, err, "created_to 格式 YYYY-MM-DD")
			return
		}
		createdTo = &t
	}

	list, total, err := svc.ListDeviceInstructions(page, pageSize, service.InstructionListFilter{
		DeviceID:    deviceID,
		Sn:          sn,
		SnExact:     snExact,
		Cmd:         cmd,
		Status:      status,
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
	})
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, "查询失败")
		return
	}
	e.PageOK(list, int(total), page, pageSize, "ok")
}

// InstructionDetail 单条指令详情
// @Summary 设备指令详情
// @Tags 平台设备-指令
// @Param id path int true "指令 ID"
// @Router /api/v1/platform-device/instructions/{id} [get]
// @Security Bearer
func (e PlatformDevice) InstructionDetail(c *gin.Context) {
	id, err := service.ParseInstructionID(c.Param("id"))
	if err != nil || id <= 0 {
		e.Error(400, errors.New("bad id"), "id 无效")
		return
	}
	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	out, err := svc.GetDeviceInstructionDetail(id)
	if err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, "记录不存在")
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

type instructionCancelReq struct {
	Reason string `json:"reason"`
}

// InstructionCancel 取消待执行指令
// @Summary 取消设备指令（仅 pending）
// @Tags 平台设备-指令
// @Param id path int true "指令 ID"
// @Param body body instructionCancelReq false "原因"
// @Router /api/v1/platform-device/instructions/{id}/cancel [post]
// @Security Bearer
func (e PlatformDevice) InstructionCancel(c *gin.Context) {
	id, err := service.ParseInstructionID(c.Param("id"))
	if err != nil || id <= 0 {
		e.Error(400, errors.New("bad id"), "id 无效")
		return
	}
	var req instructionCancelReq
	_ = c.ShouldBindJSON(&req)

	svc := service.PlatformDeviceService{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&svc.Service).Errors; err != nil {
		e.Error(500, err, "服务初始化失败")
		return
	}
	op := ""
	if uid := user.GetUserId(c); uid > 0 {
		op = c.GetString("user_name")
		if op == "" {
			op = c.GetString("nickname")
		}
		if op == "" {
			op = strconv.FormatInt(int64(uid), 10)
		}
	}
	if err := svc.CancelDeviceInstruction(&service.CancelDeviceInstructionIn{
		ID:       id,
		Reason:   req.Reason,
		Operator: op,
	}); err != nil {
		if errors.Is(err, service.ErrPlatformDeviceNotFound) {
			e.Error(404, err, "记录不存在")
			return
		}
		e.Logger.Error(err)
		e.Error(400, err, err.Error())
		return
	}
	e.OK(nil, "已取消")
}
