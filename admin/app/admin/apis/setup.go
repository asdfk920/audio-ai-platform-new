package apis

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/api"
	"github.com/go-admin-team/go-admin-core/sdk/config"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/captcha"

	"go-admin/app/admin/service"
)

type Setup struct {
	api.Api
}

// #region agent log
func agentDebugSetupAPI(hypothesisID, location, message string, data map[string]any) {
	const p = "c:/Users/Lenovo/Desktop/audio-ai-platform/debug-a35f07.log"
	line := map[string]any{
		"sessionId":    "a35f07",
		"runId":        "setup-status",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	b, err := json.Marshal(line)
	if err != nil {
		return
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(b, '\n'))
}

// #endregion

// GetStatus GET /api/admin/setup/status
func (e Setup) GetStatus(c *gin.Context) {
	// #region agent log
	agentDebugSetupAPI("H9", "apis/setup.go:GetStatus", "setup status request arrived", map[string]any{
		"path": c.Request.URL.Path,
		"host": c.Request.Host,
	})
	// #endregion
	e.MakeContext(c).MakeOrm()
	if e.Orm == nil {
		// #region agent log
		agentDebugSetupAPI("H10", "apis/setup.go:GetStatus", "orm is nil in setup status", map[string]any{"path": c.Request.URL.Path})
		// #endregion
		e.Error(http.StatusInternalServerError, nil, "数据库未就绪")
		return
	}
	needs, err := service.NeedsSetup(e.Orm)
	if err != nil {
		// #region agent log
		agentDebugSetupAPI("H11", "apis/setup.go:GetStatus", "NeedsSetup returned error", map[string]any{"err": err.Error()})
		// #endregion
		e.Error(500, err, err.Error())
		return
	}
	setupRoles := []gin.H{
		{"slug": "super_admin", "label": "超级管理员"},
		{"slug": "operator", "label": "运营管理员"},
		{"slug": "finance", "label": "财务管理员"},
	}
	// #region agent log
	agentDebugSetupAPI("H12", "apis/setup.go:GetStatus", "setup status response", map[string]any{"needsSetup": needs})
	// #endregion
	e.OK(gin.H{"needsSetup": needs, "setupRoles": setupRoles}, "ok")
}

type setupRegisterReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"`
	// RoleSlug super_admin | operator | finance，默认 super_admin
	RoleSlug string `json:"roleSlug"`
	Code     string `json:"code" binding:"required"`
	UUID     string `json:"uuid" binding:"required"`
}

// PostRegister POST /api/admin/setup/register
// 冷启动首管理员：仅写入 public.sys_admin，不写入 public.users，不建立与 C 端用户表的关联。
func (e Setup) PostRegister(c *gin.Context) {
	e.MakeContext(c).MakeOrm()
	if e.Orm == nil {
		e.Error(http.StatusInternalServerError, nil, "数据库未就绪")
		return
	}
	var req setupRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(http.StatusBadRequest, err, "参数错误")
		return
	}
	if config.ApplicationConfig.Mode != "dev" {
		if !captcha.Verify(req.UUID, req.Code, true) {
			e.Error(http.StatusBadRequest, errors.New("captcha"), "验证码错误")
			return
		}
	}
	err := service.RegisterFirstAdmin(e.Orm, req.Username, req.Password, req.Nickname, req.RoleSlug)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSetupNotNeeded):
			e.Error(http.StatusForbidden, err, "已完成初始化，请使用登录")
		case errors.Is(err, service.ErrWeakPassword):
			e.Error(http.StatusBadRequest, err, "密码不符合策略：至少8位且含大小写字母、数字与特殊字符")
		case errors.Is(err, service.ErrInvalidUsername):
			e.Error(http.StatusBadRequest, err, "用户名格式无效：3–64位字母数字，须以字母开头")
		case errors.Is(err, service.ErrUsernameTaken):
			e.Error(http.StatusConflict, err, "用户名已存在")
		case errors.Is(err, service.ErrInvalidSetupRole):
			e.Error(http.StatusBadRequest, err, "无效的管理员身份，请选择超级管理员、运营管理员或财务管理员")
		case errors.Is(err, service.ErrSuperAdminRole) || errors.Is(err, service.ErrSetupRoleNotFound):
			e.Error(http.StatusInternalServerError, err, "所选角色在数据库中不存在，请先执行数据库迁移（含 roles 表）")
		default:
			e.Logger.Error(err)
			e.Error(500, err, err.Error())
		}
		return
	}
	e.OK(gin.H{"ok": true}, "注册成功，请登录")
}
