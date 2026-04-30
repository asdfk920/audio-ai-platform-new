package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-admin-team/go-admin-core/sdk/api"
	_ "github.com/go-admin-team/go-admin-core/sdk/pkg/response"

	"go-admin/app/admin/service"
	"go-admin/app/admin/service/dto"
)

type SysPermission struct {
	api.Api
}

// PermissionList 权限列表（按模块分组）
// @Summary 权限列表（按模块分组）
// @Description 获取系统所有权限标识，按模块和类型分组展示
// @Tags 权限/Permission
// @Param module query string false "权限所属模块（user/device/content/order/system）"
// @Param type query string false "权限类型（1:菜单权限 2:操作权限 3:数据权限）"
// @Param keyword query string false "关键词搜索权限名称或标识"
// @Param status query string false "权限状态（1:禁用 2:启用）"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response "{"code": 200, "data": {...}}"
// @Router /api/v1/permission/list [get]
// @Security Bearer
func (e SysPermission) PermissionList(c *gin.Context) {
	s := service.SysPermission{}
	req := dto.SysPermissionGetPageReq{}
	err := e.MakeContext(c).
		MakeOrm().
		Bind(&req, binding.Form).
		MakeService(&s.Service).
		Errors
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, err.Error())
		return
	}

	list := make([]dto.SysPermissionListItem, 0)
	var count int64

	err = s.GetPermissionList(&req, &list, &count)
	if err != nil {
		e.Error(500, err, "查询失败")
		return
	}

	e.PageOK(list, int(count), req.GetPageIndex(), req.GetPageSize(), "查询成功")
}

// PermissionTree 权限树形结构
// @Summary 权限树形结构
// @Description 获取树形结构的权限列表，模块为父节点，权限为子节点
// @Tags 权限/Permission
// @Param module query string false "权限所属模块"
// @Param status query string false "权限状态"
// @Success 200 {object} response.Response "{"code": 200, "data": [...]}"
// @Router /api/v1/permission/tree [get]
// @Security Bearer
func (e SysPermission) PermissionTree(c *gin.Context) {
	s := service.SysPermission{}
	req := dto.SysPermissionTreeReq{}
	err := e.MakeContext(c).
		MakeOrm().
		Bind(&req, binding.Form).
		MakeService(&s.Service).
		Errors
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, err.Error())
		return
	}

	tree, err := s.GetPermissionTree(&req)
	if err != nil {
		e.Error(500, err, "查询失败")
		return
	}

	e.OK(tree, "查询成功")
}
