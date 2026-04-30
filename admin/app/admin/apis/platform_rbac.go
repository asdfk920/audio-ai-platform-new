package apis

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/api"
	jwtauth "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"

	"go-admin/app/admin/service"
)

// PlatformRbac 平台权限矩阵（roles.permissions 落库）
type PlatformRbac struct {
	api.Api
}

func roleKeyFromJWT(c *gin.Context) string {
	claims := jwtauth.ExtractClaims(c)
	if v, ok := claims[jwtauth.RoleKey].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := claims["rolekey"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// Matrix GET /api/v1/platform-rbac/matrix
func (e PlatformRbac) Matrix(c *gin.Context) {
	e.MakeContext(c).MakeOrm()
	if e.Orm == nil {
		e.Error(http.StatusInternalServerError, nil, "数据库未就绪")
		return
	}
	rk := roleKeyFromJWT(c)
	if rk == "" {
		_ = e.Orm.Raw(`
			SELECT r.role_key FROM sys_user u
			INNER JOIN sys_role r ON r.role_id = u.role_id
			WHERE u.user_id = ? LIMIT 1
		`, user.GetUserId(c)).Scan(&rk)
	}
	svc := service.PlatformRbac{}
	data, err := svc.MatrixForRole(e.Orm, rk)
	if err != nil {
		e.Error(500, err, "加载矩阵失败")
		return
	}
	e.OK(data, "ok")
}

// Roles GET /api/v1/platform-rbac/roles
func (e PlatformRbac) Roles(c *gin.Context) {
	e.MakeContext(c).MakeOrm()
	if e.Orm == nil {
		e.Error(http.StatusInternalServerError, nil, "数据库未就绪")
		return
	}
	svc := service.PlatformRbac{}
	list, err := svc.ListRoles(e.Orm)
	if err != nil {
		e.Error(500, err, err.Error())
		return
	}
	e.OK(gin.H{
		"modules": svc.ModulesMeta(),
		"list":    list,
	}, "ok")
}

type updateRoleReq struct {
	Modules map[string]string `json:"modules"`
}

// UpdateRole PUT /api/v1/platform-rbac/roles/:roleKey
func (e PlatformRbac) UpdateRole(c *gin.Context) {
	e.MakeContext(c).MakeOrm()
	if e.Orm == nil {
		e.Error(http.StatusInternalServerError, nil, "数据库未就绪")
		return
	}
	target := strings.TrimSpace(c.Param("roleKey"))
	var req updateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		e.Error(http.StatusBadRequest, err, "参数错误")
		return
	}
	if req.Modules == nil {
		e.Error(http.StatusBadRequest, errors.New("modules required"), "modules 不能为空")
		return
	}
	editor := roleKeyFromJWT(c)
	svc := service.PlatformRbac{}
	err := svc.UpdateRoleModules(e.Orm, editor, target, req.Modules)
	if err != nil {
		if errors.Is(err, service.ErrRbacForbidden) {
			e.Error(http.StatusForbidden, err, "无权修改权限配置")
			return
		}
		e.Error(500, err, err.Error())
		return
	}
	e.OK(gin.H{"saved": true, "message": "已保存"}, "ok")
}
