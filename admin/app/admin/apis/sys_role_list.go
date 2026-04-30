package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-admin-team/go-admin-core/sdk/api"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"
	_ "github.com/go-admin-team/go-admin-core/sdk/pkg/response"

	"go-admin/app/admin/models"
	"go-admin/app/admin/service"
	"go-admin/app/admin/service/dto"
)

type SysRole struct {
	api.Api
}

// RoleList 角色列表（完整版，包含统计信息）
// @Summary 角色列表（完整版）
// @Description 获取 JSON，包含权限数量、管理员数量等统计信息
// @Tags 角色/Role
// @Param keyword query string false "关键词（角色名称或编码）"
// @Param status query string false "状态（1:禁用 2:正常）"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param sort_by query string false "排序字段"
// @Param sort_order query string false "排序方式"
// @Success 200 {object} response.Response "{"code": 200, "data": [...]}"
// @Router /api/v1/role/list [get]
// @Security Bearer
func (e SysRole) RoleList(c *gin.Context) {
	s := service.SysRole{}
	req := dto.SysRoleGetPageReq{}
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

	list := make([]dto.SysRoleListItem, 0)
	var count int64

	err = s.GetRoleList(&req, &list, &count)
	if err != nil {
		e.Error(500, err, "查询失败")
		return
	}

	e.PageOK(list, int(count), req.GetPageIndex(), req.GetPageSize(), "查询成功")
}

// CreateRole 创建角色
// @Summary 创建角色
// @Description 创建新角色，配置角色名称、权限范围、角色描述等信息
// @Tags 角色/Role
// @Accept application/json
// @Param data body dto.SysRoleCreateReq true "角色信息"
// @Success 200 {object} response.Response "{"code": 200, "data": {...}}"
// @Router /api/v1/role/create [post]
// @Security Bearer
func (e SysRole) CreateRole(c *gin.Context) {
	s := service.SysRole{}
	req := dto.SysRoleCreateReq{}
	err := e.MakeContext(c).
		MakeOrm().
		Bind(&req, binding.JSON).
		MakeService(&s.Service).
		Errors
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, err.Error())
		return
	}

	// 设置创建人
	req.CreateBy = user.GetUserId(c)

	// 调用服务层创建角色
	result, err := s.CreateRole(&req)
	if err != nil {
		e.Error(500, err, "创建失败")
		return
	}

	e.OK(result, "创建成功")
}

// UpdateRole 修改角色
// @Summary 修改角色
// @Description 更新角色配置信息，包括角色名称、权限范围、角色描述、状态等
// @Tags 角色/Role
// @Accept application/json
// @Param data body dto.SysRoleUpdateRequest true "角色信息"
// @Success 200 {object} response.Response "{"code": 200, "data": {...}}"
// @Router /api/v1/role/update [put]
// @Security Bearer
func (e SysRole) UpdateRole(c *gin.Context) {
	s := service.SysRole{}
	req := dto.SysRoleUpdateRequest{}
	err := e.MakeContext(c).
		MakeOrm().
		Bind(&req, binding.JSON).
		MakeService(&s.Service).
		Errors
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, err.Error())
		return
	}

	// 设置更新人
	req.UpdateBy = user.GetUserId(c)

	// 调用服务层更新角色
	result, err := s.UpdateRole(&req)
	if err != nil {
		e.Error(500, err, "更新失败")
		return
	}

	e.OK(result, "更新成功")
}

// DeleteRole 删除角色
// @Summary 删除角色
// @Description 删除角色，包含关联检查、软删除处理等
// @Tags 角色/Role
// @Accept application/json
// @Param data body dto.SysRoleDeleteRequest true "删除信息"
// @Success 200 {object} response.Response "{"code": 200, "data": {...}}"
// @Router /api/v1/role/delete [post]
// @Security Bearer
func (e SysRole) DeleteRole(c *gin.Context) {
	s := service.SysRole{}
	req := dto.SysRoleDeleteRequest{}
	err := e.MakeContext(c).
		MakeOrm().
		Bind(&req, binding.JSON).
		MakeService(&s.Service).
		Errors
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, err.Error())
		return
	}

	// 设置删除人
	req.DeleteBy = user.GetUserId(c)

	// 调用服务层删除角色
	result, err := s.DeleteRole(&req)
	if err != nil {
		e.Error(500, err, "删除失败")
		return
	}

	e.OK(result, "删除成功")
}

// Update2Status 修改角色状态（与标准路由 /api/v1/role-status 对齐）
func (e SysRole) Update2Status(c *gin.Context) {
	s := service.SysRole{}
	req := dto.UpdateStatusReq{}
	err := e.MakeContext(c).
		MakeOrm().
		Bind(&req, binding.JSON, nil).
		MakeService(&s.Service).
		Errors
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, err.Error())
		return
	}
	req.SetUpdateBy(user.GetUserId(c))
	err = s.UpdateStatus(&req)
	if err != nil {
		e.Error(500, err, err.Error())
		return
	}
	e.OK(req.GetId(), "更新成功")
}

// Update2DataScope 更新角色数据权限（与标准路由 /api/v1/roledatascope 对齐）
func (e SysRole) Update2DataScope(c *gin.Context) {
	s := service.SysRole{}
	req := dto.RoleDataScopeReq{}
	err := e.MakeContext(c).
		MakeOrm().
		Bind(&req, binding.JSON, nil).
		MakeService(&s.Service).
		Errors
	if err != nil {
		e.Logger.Error(err)
		e.Error(500, err, err.Error())
		return
	}
	data := &models.SysRole{
		RoleId:    req.RoleId,
		DataScope: req.DataScope,
		DeptIds:   req.DeptIds,
	}
	data.UpdateBy = user.GetUserId(c)
	err = s.UpdateDataScope(&req).Error
	if err != nil {
		e.Error(500, err, err.Error())
		return
	}
	e.OK(nil, "操作成功")
}
