package apis

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-admin-team/go-admin-core/sdk/api"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	_ "github.com/go-admin-team/go-admin-core/sdk/pkg/response"

	"go-admin/app/admin/service"
	"go-admin/app/admin/service/dto"
	"go-admin/common/rbac"
)

type SysAdmin struct {
	api.Api
}

// #region agent log
func agentDebugNDJSONA35f07API(runID, hypothesisId, location, message string, data map[string]any) {
	type payload struct {
		SessionID    string         `json:"sessionId"`
		RunID        string         `json:"runId"`
		HypothesisID string         `json:"hypothesisId"`
		Location     string         `json:"location"`
		Message      string         `json:"message"`
		Data         map[string]any `json:"data,omitempty"`
		Timestamp    int64          `json:"timestamp"`
	}
	root := os.Getenv("WORKSPACE_ROOT")
	if root == "" {
		root = `C:\Users\Lenovo\Desktop\audio-ai-platform`
	}
	p := filepath.Join(root, "debug-a35f07.log")
	b, err := json.Marshal(payload{
		SessionID:    "a35f07",
		RunID:        runID,
		HypothesisID: hypothesisId,
		Location:     location,
		Message:      message,
		Data:         data,
		Timestamp:    time.Now().UnixMilli(),
	})
	if err != nil {
		return
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
	_ = f.Close()
}

// #endregion

// ===== 公共辅助 =====

// parsePathID 读取 /:id 路径参数并返回正整数；失败时直接写错误响应。
func (e SysAdmin) parsePathID(c *gin.Context) (int, bool) {
	raw := c.Param("id")
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		e.Error(400, errors.New("invalid path id"), "管理员 ID 无效")
		return 0, false
	}
	return id, true
}

// mapSysAdminErr 把 service 哨兵错误映射为 HTTP 语义的响应，返回 true 表示已响应。
func (e SysAdmin) mapSysAdminErr(err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, service.ErrSysAdminNotFound):
		e.Error(404, err, "管理员不存在")
	case errors.Is(err, service.ErrSysAdminBuiltin):
		e.Error(403, err, "系统内置管理员不可变更")
	case errors.Is(err, service.ErrSysAdminSelfProtect):
		e.Error(400, err, "不能对自己执行该操作")
	case errors.Is(err, service.ErrSysAdminPwdBadOld):
		e.Error(400, err, "旧密码不正确")
	case errors.Is(err, service.ErrSysAdminPwdPolicy):
		e.Error(400, err, "密码需 8-20 位，且同时含大小写字母和数字")
	case errors.Is(err, service.ErrSysAdminIPRejected):
		e.Error(403, err, "当前 IP 不在允许登录范围")
	case errors.Is(err, service.ErrSysAdminTimeRejected):
		e.Error(403, err, "当前时间不在允许登录时段")
	default:
		return false
	}
	return true
}

// isSuperAdminFromToken 判断当前登录是否为 super_admin（Casbin bypass role）。
func isSuperAdminFromToken(c *gin.Context) bool {
	claims := jwt.ExtractClaims(c)
	// 兼容两种写法：标准 rolekey / 我们额外透传的 role
	raw := ""
	if v, ok := claims["rolekey"]; ok {
		if s, ok2 := v.(string); ok2 {
			raw = strings.TrimSpace(s)
		}
	}
	if raw == "" {
		if v, ok := claims["role"]; ok {
			if s, ok2 := v.(string); ok2 {
				raw = strings.TrimSpace(s)
			}
		}
	}
	return rbac.IsCasbinBypassRole(raw)
}

// ===== 列表 / 详情 / 创建 / 更新 / 删除 / 状态 =====

// AdminList 管理员列表
func (e SysAdmin) AdminList(c *gin.Context) {
	s := service.SysAdmin{}
	req := dto.SysAdminGetPageReq{}
	if err := e.MakeContext(c).MakeOrm().Bind(&req, binding.Form).MakeService(&s.Service).Errors; err != nil {
		e.Logger.Error(err)
		e.Error(500, err, err.Error())
		return
	}
	list := make([]dto.SysAdminListItem, 0)
	var count int64
	if err := s.GetAdminList(&req, &list, &count); err != nil {
		e.Error(500, err, "查询失败")
		return
	}
	e.PageOK(list, int(count), req.GetPageIndex(), req.GetPageSize(), "查询成功")
}

// AdminDetail 管理员详情，通过 /:id 读取
func (e SysAdmin) AdminDetail(c *gin.Context) {
	id, ok := e.parsePathID(c)
	if !ok {
		return
	}
	s := service.SysAdmin{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, err.Error())
		return
	}
	result, err := s.GetAdminDetail(&dto.SysAdminDetailReq{Id: id})
	if err != nil {
		if e.mapSysAdminErr(err) {
			return
		}
		e.Error(500, err, "查询失败")
		return
	}
	e.OK(result, "查询成功")
}

// AdminCreate 创建管理员
func (e SysAdmin) AdminCreate(c *gin.Context) {
	s := service.SysAdmin{}
	req := dto.SysAdminCreateReq{}
	if err := e.MakeContext(c).MakeOrm().Bind(&req, binding.JSON).MakeService(&s.Service).Errors; err != nil {
		e.Logger.Error(err)
		e.Error(500, err, err.Error())
		return
	}
	req.CreateBy = user.GetUserId(c)
	// #region agent log
	agentDebugNDJSONA35f07API("admin-create-api", "H0", "apis/sys_admin.go:AdminCreate", "bind_ok", map[string]any{
		"createBy":   req.CreateBy,
		"roleIdsLen": len(req.RoleIds),
		"usernameLen": len(strings.TrimSpace(req.Username)),
	})
	// #endregion
	result, err := s.AdminCreate(&req)
	if err != nil {
		// #region agent log
		agentDebugNDJSONA35f07API("admin-create-api", "H5", "apis/sys_admin.go:AdminCreate", "service_error", map[string]any{
			"errType": fmt.Sprintf("%T", err),
			"errMsg":  err.Error(),
		})
		// #endregion
		if e.mapSysAdminErr(err) {
			return
		}
		e.Error(400, err, err.Error())
		return
	}
	e.OK(result, "创建成功")
}

// AdminRegister 为了兼容旧路径 /api/admin/register，实际走 AdminCreate。
func (e SysAdmin) AdminRegister(c *gin.Context) { e.AdminCreate(c) }

// AdminUpdate 更新管理员；同时支持 Body 带 user_id 与 URI /:id。
func (e SysAdmin) AdminUpdate(c *gin.Context) {
	s := service.SysAdmin{}
	req := dto.SysAdminUpdateReq{}
	if err := e.MakeContext(c).MakeOrm().Bind(&req, binding.JSON).MakeService(&s.Service).Errors; err != nil {
		e.Logger.Error(err)
		e.Error(500, err, err.Error())
		return
	}
	if c.Param("id") != "" {
		if id, ok := e.parsePathID(c); ok {
			req.UserId = id
		} else {
			return
		}
	}
	req.UpdateBy = user.GetUserId(c)
	req.ActorIsCasbinBypass = isSuperAdminFromToken(c)
	// #region agent log
	agentDebugNDJSONA35f07API("admin-update-api", "H6", "apis/sys_admin.go:AdminUpdate", "handler_ctx", map[string]any{
		"updateBy":    req.UpdateBy,
		"targetId":    req.UserId,
		"actorBypass": req.ActorIsCasbinBypass,
		"roleIdsLen":  len(req.RoleIds),
	})
	// #endregion
	result, err := s.AdminUpdate(&req)
	if err != nil {
		if e.mapSysAdminErr(err) {
			return
		}
		e.Error(400, err, err.Error())
		return
	}
	e.OK(result, "更新成功")
}

// AdminDelete 删除管理员；支持 /:id 或 body。
func (e SysAdmin) AdminDelete(c *gin.Context) {
	s := service.SysAdmin{}
	req := dto.SysAdminDeleteReq{}
	_ = c.ShouldBindJSON(&req) // 允许无 body
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Logger.Error(err)
		e.Error(500, err, err.Error())
		return
	}
	if c.Param("id") != "" {
		if id, ok := e.parsePathID(c); ok {
			req.UserId = id
		} else {
			return
		}
	}
	req.DeleteBy = user.GetUserId(c)
	if req.UserId <= 0 {
		e.Error(400, errors.New("missing user id"), "管理员 ID 无效")
		return
	}
	result, err := s.AdminDelete(&req)
	if err != nil {
		if e.mapSysAdminErr(err) {
			return
		}
		e.Error(400, err, err.Error())
		return
	}
	e.OK(result, "删除成功")
}

// AdminStatus 启用/禁用；支持 /:id。
func (e SysAdmin) AdminStatus(c *gin.Context) {
	s := service.SysAdmin{}
	req := dto.SysAdminStatusReq{}
	if err := e.MakeContext(c).MakeOrm().Bind(&req, binding.JSON).MakeService(&s.Service).Errors; err != nil {
		e.Logger.Error(err)
		e.Error(500, err, err.Error())
		return
	}
	if c.Param("id") != "" {
		if id, ok := e.parsePathID(c); ok {
			req.UserId = id
		} else {
			return
		}
	}
	req.UpdateBy = user.GetUserId(c)
	result, err := s.AdminStatus(&req)
	if err != nil {
		if e.mapSysAdminErr(err) {
			return
		}
		e.Error(400, err, err.Error())
		return
	}
	e.OK(result, "更新成功")
}

// ===== 新增：批量删除 =====

// AdminBatchDelete POST /api/admin/users/batch-delete
func (e SysAdmin) AdminBatchDelete(c *gin.Context) {
	s := service.SysAdmin{}
	req := dto.SysAdminBatchDeleteReq{}
	if err := e.MakeContext(c).MakeOrm().Bind(&req, binding.JSON).MakeService(&s.Service).Errors; err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "请求参数无效")
		return
	}
	req.UpdateBy = user.GetUserId(c)
	req.DeleteBy = user.GetUserId(c)
	resp, err := s.AdminBatchDelete(&req)
	if err != nil {
		e.Error(500, err, "批量删除失败")
		return
	}
	e.OK(resp, "批量删除完成")
}

// ===== 新增：重置密码（超管重置他人） =====

// AdminResetPassword PUT /api/admin/users/:id/password
func (e SysAdmin) AdminResetPassword(c *gin.Context) {
	id, ok := e.parsePathID(c)
	if !ok {
		return
	}
	s := service.SysAdmin{}
	req := dto.SysAdminResetPasswordReq{}
	if err := e.MakeContext(c).MakeOrm().Bind(&req, binding.JSON).MakeService(&s.Service).Errors; err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "请求参数无效")
		return
	}
	req.UserId = id
	req.UpdateBy = user.GetUserId(c)
	if err := s.AdminResetPassword(&req); err != nil {
		if e.mapSysAdminErr(err) {
			return
		}
		e.Error(400, err, err.Error())
		return
	}
	e.OK(gin.H{"user_id": id}, "重置成功")
}

// ===== 新增：自助修改密码 =====

// AdminChangePasswordSelf POST /api/admin/account/change-password
func (e SysAdmin) AdminChangePasswordSelf(c *gin.Context) {
	s := service.SysAdmin{}
	req := dto.SysAdminChangePasswordReq{}
	if err := e.MakeContext(c).MakeOrm().Bind(&req, binding.JSON).MakeService(&s.Service).Errors; err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "请求参数无效")
		return
	}
	uid := user.GetUserId(c)
	if uid <= 0 {
		e.Error(401, errors.New("unauthorized"), "未登录")
		return
	}
	if err := s.AdminChangePasswordSelf(uid, req.OldPassword, req.NewPassword); err != nil {
		if e.mapSysAdminErr(err) {
			return
		}
		e.Error(400, err, err.Error())
		return
	}
	e.OK(gin.H{"user_id": uid}, "密码已更新")
}

// ===== 新增：安全策略（IP / 时间窗） =====

// AdminSetSecurity PUT /api/admin/users/:id/security
func (e SysAdmin) AdminSetSecurity(c *gin.Context) {
	id, ok := e.parsePathID(c)
	if !ok {
		return
	}
	s := service.SysAdmin{}
	req := dto.SysAdminSecurityReq{}
	if err := e.MakeContext(c).MakeOrm().Bind(&req, binding.JSON).MakeService(&s.Service).Errors; err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "请求参数无效")
		return
	}
	req.UserId = id
	req.UpdateBy = user.GetUserId(c)
	if err := s.AdminSetSecurity(&req); err != nil {
		if e.mapSysAdminErr(err) {
			return
		}
		e.Error(400, err, err.Error())
		return
	}
	e.OK(gin.H{"user_id": id}, "安全策略已更新")
}

// ===== 新增：强制改密标志 =====

// AdminSetForceChange PUT /api/admin/users/:id/must-change-password
func (e SysAdmin) AdminSetForceChange(c *gin.Context) {
	id, ok := e.parsePathID(c)
	if !ok {
		return
	}
	s := service.SysAdmin{}
	req := dto.SysAdminForceChangeReq{}
	if err := e.MakeContext(c).MakeOrm().Bind(&req, binding.JSON).MakeService(&s.Service).Errors; err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "请求参数无效")
		return
	}
	req.UserId = id
	req.UpdateBy = user.GetUserId(c)
	if err := s.AdminSetForceChange(&req); err != nil {
		if e.mapSysAdminErr(err) {
			return
		}
		e.Error(400, err, err.Error())
		return
	}
	e.OK(gin.H{"user_id": id, "must": req.Must}, "已更新")
}

// ===== 新增：当前管理员 profile（从 Token 获取 ID） =====

// AdminProfileGet GET /api/admin/profile
func (e SysAdmin) AdminProfileGet(c *gin.Context) {
	s := service.SysAdmin{}
	if err := e.MakeContext(c).MakeOrm().MakeService(&s.Service).Errors; err != nil {
		e.Error(500, err, err.Error())
		return
	}
	uid := user.GetUserId(c)
	if uid <= 0 {
		e.Error(401, errors.New("unauthorized"), "未登录")
		return
	}
	result, err := s.GetAdminDetail(&dto.SysAdminDetailReq{Id: uid})
	if err != nil {
		if e.mapSysAdminErr(err) {
			return
		}
		e.Error(500, err, "查询失败")
		return
	}
	e.OK(result, "查询成功")
}

// AdminProfileUpdate PUT /api/admin/profile
func (e SysAdmin) AdminProfileUpdate(c *gin.Context) {
	s := service.SysAdmin{}
	req := dto.SysAdminProfileUpdateReq{}
	if err := e.MakeContext(c).MakeOrm().Bind(&req, binding.JSON).MakeService(&s.Service).Errors; err != nil {
		e.Logger.Error(err)
		e.Error(400, err, "请求参数无效")
		return
	}
	uid := user.GetUserId(c)
	if uid <= 0 {
		e.Error(401, errors.New("unauthorized"), "未登录")
		return
	}

	// 先取旧值，用于“部分字段更新”回填缺省
	old, err := s.GetAdminDetail(&dto.SysAdminDetailReq{Id: uid})
	if err != nil {
		if e.mapSysAdminErr(err) {
			return
		}
		e.Error(500, err, "查询失败")
		return
	}

	// 权限护栏：role/status 只有超管能改
	isSuper := isSuperAdminFromToken(c)
	if !isSuper {
		if len(req.RoleIds) > 0 || strings.TrimSpace(req.Status) != "" {
			e.Error(403, errors.New("forbidden"), "仅超级管理员可修改角色/状态")
			return
		}
	}

	merged := dto.SysAdminUpdateReq{
		UserId:   uid,
		Nickname: strings.TrimSpace(req.Nickname),
		RealName: strings.TrimSpace(req.RealName),
		Email:    strings.TrimSpace(req.Email),
		Phone:    strings.TrimSpace(req.Phone),
		Avatar:   strings.TrimSpace(req.Avatar),
		Status:   strings.TrimSpace(req.Status),
		Remark:   strings.TrimSpace(req.Remark),
	}
	if merged.Nickname == "" {
		merged.Nickname = old.Nickname
	}
	if merged.RealName == "" {
		merged.RealName = old.RealName
	}
	if merged.Email == "" {
		merged.Email = old.Email
	}
	if merged.Phone == "" {
		merged.Phone = old.Phone
	}
	if merged.Avatar == "" {
		merged.Avatar = old.Avatar
	}
	if merged.Remark == "" {
		merged.Remark = old.Remark
	}
	// dept_id：nil 表示不改；否则设置为指定值（可置 0）
	if req.DeptId != nil {
		merged.DeptId = *req.DeptId
	} else {
		merged.DeptId = old.DeptId
	}
	// role_ids：不传则沿用旧值；传了且为超管则覆盖
	if len(req.RoleIds) > 0 {
		merged.RoleIds = req.RoleIds
	} else {
		merged.RoleIds = old.RoleIds
	}
	// status：不传则沿用旧值
	if merged.Status == "" {
		merged.Status = old.Status
	}

	merged.UpdateBy = user.GetUserId(c)
	merged.ActorIsCasbinBypass = isSuper
	result, err := s.AdminUpdate(&merged)
	if err != nil {
		if e.mapSysAdminErr(err) {
			return
		}
		e.Error(400, err, err.Error())
		return
	}
	e.OK(result, "更新成功")
}
