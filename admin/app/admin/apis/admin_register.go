package apis

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/api"
	"github.com/go-admin-team/go-admin-core/sdk/config"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/captcha"

	"go-admin/app/admin/service"
)

// AdminAccount 管理员账号注册模块
//
// 与 C 端用户注册的边界：
//   - 数据：仅写入 public.sys_admin；不读写 public.users / user_role_rel
//   - 用户名：独立命名空间；同名账号可在 users 与 sys_admin 各自存在，不互相校验
//   - 角色：只从 public.roles 读 slug→id 映射，不参与 C 端角色关系维护
type AdminAccount struct {
	api.Api
}

type adminAccountRegisterReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"`
	RealName string `json:"realName"`
	Mobile   string `json:"mobile"`
	Email    string `json:"email"`
	// RoleSlug super_admin | operator | finance，默认 super_admin
	RoleSlug string `json:"roleSlug"`
	Code     string `json:"code" binding:"required"`
	UUID     string `json:"uuid" binding:"required"`
}

// Register 管理员账号注册
// @Summary 管理员账号注册
// @Description 仅写入 public.sys_admin；与 C 端 users 注册完全独立
// @Tags 管理员/Admin
// @Accept application/json
// @Param data body adminAccountRegisterReq true "管理员注册信息"
// @Success 200 {object} response.Response
// @Router /api/admin/account/register [post]
func (e AdminAccount) Register(c *gin.Context) {
	e.MakeContext(c).MakeOrm()
	if e.Orm == nil {
		e.Error(http.StatusInternalServerError, nil, "数据库未就绪")
		return
	}
	var req adminAccountRegisterReq
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

	row, err := service.RegisterAdminAccount(e.Orm, service.AdminRegisterInput{
		Username: req.Username,
		Password: req.Password,
		Nickname: req.Nickname,
		RealName: req.RealName,
		Mobile:   req.Mobile,
		Email:    req.Email,
		RoleSlug: req.RoleSlug,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWeakPassword):
			e.Error(http.StatusBadRequest, err, "密码不符合策略：至少8位且含大小写字母、数字与特殊字符")
		case errors.Is(err, service.ErrInvalidUsername):
			e.Error(http.StatusBadRequest, err, "用户名格式无效：3–64位字母数字，须以字母开头")
		case errors.Is(err, service.ErrAdminUsernameTaken):
			e.Error(http.StatusConflict, err, "管理员用户名已存在")
		case errors.Is(err, service.ErrAdminInvalidRole):
			e.Error(http.StatusBadRequest, err, "无效的管理员身份，请选择超级管理员、运营管理员或财务管理员")
		case errors.Is(err, service.ErrAdminRoleNotFound):
			e.Error(http.StatusInternalServerError, err, "所选角色在数据库中不存在，请先执行数据库迁移（含 roles 表）")
		default:
			e.Logger.Error(err)
			e.Error(500, err, err.Error())
		}
		return
	}

	e.OK(gin.H{
		"id":       row.Id,
		"username": row.Username,
		"nickName": row.NickName,
		"roleId":   row.RoleId,
		"roleCode": row.RoleCode,
	}, "注册成功，请登录")
}
