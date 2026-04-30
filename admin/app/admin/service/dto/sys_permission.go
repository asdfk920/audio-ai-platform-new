package dto

import (
	"go-admin/common/dto"
	common "go-admin/common/models"
)

// SysPermissionGetPageReq 权限列表查询请求
type SysPermissionGetPageReq struct {
	dto.Pagination `search:"-"`

	Module    string `form:"module" search:"type:exact;column:module;table:sys_menu" comment:"权限所属模块"`     // 权限所属模块
	Type      string `form:"type" search:"type:exact;column:type;table:sys_menu" comment:"权限类型"`           // 权限类型（1:菜单 2:操作 3:数据）
	Keyword   string `form:"keyword" search:"type:contains;column:menu_name;table:sys_menu" comment:"关键词"` // 关键词
	Status    string `form:"status" search:"type:exact;column:status;table:sys_menu" comment:"权限状态"`       // 权限状态
	SortBy    string `form:"sort_by" search:"-" comment:"排序字段"`                                            // 排序字段
	SortOrder string `form:"sort_order" search:"-" comment:"排序方式"`                                         // 排序方式
}

func (s *SysPermissionGetPageReq) GetPageIndex() int {
	if s.PageIndex <= 0 {
		return 1
	}
	return s.PageIndex
}

func (s *SysPermissionGetPageReq) GetPageSize() int {
	if s.PageSize <= 0 {
		return 20
	}
	return s.PageSize
}

func (s *SysPermissionGetPageReq) GetNeedSearch() interface{} {
	return *s
}

// SysPermissionTreeReq 权限树查询请求
type SysPermissionTreeReq struct {
	Module string `form:"module" comment:"权限所属模块"` // 权限所属模块
	Status string `form:"status" comment:"权限状态"`   // 权限状态
}

// SysPermissionListItem 权限列表项
type SysPermissionListItem struct {
	PermissionId   int    `json:"permission_id"`   // 权限 ID
	MenuId         int    `json:"menu_id"`         // 菜单 ID
	PermissionName string `json:"permission_name"` // 权限名称
	PermissionCode string `json:"permission_code"` // 权限标识
	Description    string `json:"description"`     // 权限描述
	Type           string `json:"type"`            // 权限类型
	TypeText       string `json:"type_text"`       // 权限类型文本
	Module         string `json:"module"`          // 所属模块
	ModuleText     string `json:"module_text"`     // 所属模块文本
	Status         string `json:"status"`          // 权限状态
	StatusText     string `json:"status_text"`     // 权限状态文本
	RoleCount      int    `json:"role_count"`      // 被多少个角色引用
	CreatedAt      string `json:"created_at"`      // 创建时间
}

// SysPermissionTreeItem 权限树节点
type SysPermissionTreeItem struct {
	Id             int                      `json:"id"`                        // 节点 ID
	Label          string                   `json:"label"`                     // 节点标签
	Type           string                   `json:"type"`                      // 节点类型（module: 模块 permission:权限）
	Children       []*SysPermissionTreeItem `json:"children,omitempty"`        // 子节点
	PermissionCode string                   `json:"permission_code,omitempty"` // 权限标识（仅权限节点）
	Module         string                   `json:"module,omitempty"`          // 所属模块（仅权限节点）
	Status         string                   `json:"status,omitempty"`          // 状态（仅权限节点）
}

// SysPermissionModuleItem 模块分组项
type SysPermissionModuleItem struct {
	ModuleName     string                  `json:"module_name"`     // 模块名称
	ModuleCode     string                  `json:"module_code"`     // 模块标识
	PermissionList []SysPermissionListItem `json:"permission_list"` // 权限项列表
}

// SysPermissionDetail 权限详情
type SysPermissionDetail struct {
	PermissionId   int    `json:"permission_id"`   // 权限 ID
	MenuId         int    `json:"menu_id"`         // 菜单 ID
	PermissionName string `json:"permission_name"` // 权限名称
	PermissionCode string `json:"permission_code"` // 权限标识
	Description    string `json:"description"`     // 权限描述
	Type           string `json:"type"`            // 权限类型
	TypeText       string `json:"type_text"`       // 权限类型文本
	Module         string `json:"module"`          // 所属模块
	ModuleText     string `json:"module_text"`     // 所属模块文本
	Status         string `json:"status"`          // 权限状态
	StatusText     string `json:"status_text"`     // 权限状态文本
	ParentId       int    `json:"parent_id"`       // 父级 ID
	Sort           int    `json:"sort"`            // 排序
	CreatedAt      string `json:"created_at"`      // 创建时间
	UpdatedAt      string `json:"updated_at"`      // 更新时间
}

// SysPermissionInsertReq 插入权限请求
type SysPermissionInsertReq struct {
	MenuName   string `json:"menuName" binding:"required"` // 菜单名称
	Permission string `json:"permission"`                  // 权限标识
	Type       string `json:"type"`                        // 类型（1:菜单 2:操作 3:数据）
	Module     string `json:"module"`                      // 所属模块
	ParentId   int    `json:"parentId"`                    // 父级 ID
	Sort       int    `json:"sort"`                        // 排序
	Status     string `json:"status"`                      // 状态
	Remark     string `json:"remark"`                      // 备注
	common.ControlBy
}

func (s *SysPermissionInsertReq) GetId() interface{} {
	return 0
}

// SysPermissionUpdateReq 更新权限请求
type SysPermissionUpdateReq struct {
	MenuId     int    `json:"menuId" binding:"required"`   // 菜单 ID
	MenuName   string `json:"menuName" binding:"required"` // 菜单名称
	Permission string `json:"permission"`                  // 权限标识
	Type       string `json:"type"`                        // 类型
	Module     string `json:"module"`                      // 所属模块
	ParentId   int    `json:"parentId"`                    // 父级 ID
	Sort       int    `json:"sort"`                        // 排序
	Status     string `json:"status"`                      // 状态
	Remark     string `json:"remark"`                      // 备注
	common.ControlBy
}

func (s *SysPermissionUpdateReq) GetId() interface{} {
	return s.MenuId
}
