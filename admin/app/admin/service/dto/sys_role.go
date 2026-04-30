package dto

import (
	"go-admin/app/admin/models"
	common "go-admin/common/models"

	"go-admin/common/dto"
)

type SysRoleGetPageReq struct {
	dto.Pagination `search:"-"`

	RoleId    int    `form:"roleId" search:"type:exact;column:role_id;table:sys_role" comment:"角色编码"`      // 角色编码
	RoleName  string `form:"roleName" search:"type:exact;column:role_name;table:sys_role" comment:"角色名称"`  // 角色名称
	Keyword   string `form:"keyword" search:"type:contains;column:role_name;table:sys_role" comment:"关键词"` // 关键词
	Status    string `form:"status" search:"type:exact;column:status;table:sys_role" comment:"状态"`         // 状态
	RoleKey   string `form:"roleKey" search:"type:exact;column:role_key;table:sys_role" comment:"角色代码"`    // 角色代码
	RoleSort  int    `form:"roleSort" search:"type:exact;column:role_sort;table:sys_role" comment:"角色排序"`  // 角色排序
	Flag      string `form:"flag" search:"type:exact;column:flag;table:sys_role" comment:"标记"`             // 标记
	Remark    string `form:"remark" search:"type:exact;column:remark;table:sys_role" comment:"备注"`         // 备注
	Admin     bool   `form:"admin" search:"type:exact;column:admin;table:sys_role" comment:"是否管理员"`
	DataScope string `form:"dataScope" search:"type:exact;column:data_scope;table:sys_role" comment:"是否管理员"`
	SortBy    string `form:"sort_by" search:"-" comment:"排序字段"`
	SortOrder string `form:"sort_order" search:"-" comment:"排序方式"`
}

type SysRoleOrder struct {
	RoleIdOrder    string `search:"type:order;column:role_id;table:sys_role" form:"roleIdOrder"`
	RoleNameOrder  string `search:"type:order;column:role_name;table:sys_role" form:"roleNameOrder"`
	RoleSortOrder  string `search:"type:order;column:role_sort;table:sys_role" form:"usernameOrder"`
	StatusOrder    string `search:"type:order;column:status;table:sys_role" form:"statusOrder"`
	CreatedAtOrder string `search:"type:order;column:created_at;table:sys_role" form:"createdAtOrder"`
}

func (m *SysRoleGetPageReq) GetNeedSearch() interface{} {
	return *m
}

type SysRoleInsertReq struct {
	RoleId    int              `uri:"id" comment:"角色编码"`        // 角色编码
	RoleName  string           `form:"roleName" comment:"角色名称"` // 角色名称
	Status    string           `form:"status" comment:"状态"`     // 状态 1禁用 2正常
	RoleKey   string           `form:"roleKey" comment:"角色代码"`  // 角色代码
	RoleSort  int              `form:"roleSort" comment:"角色排序"` // 角色排序
	Flag      string           `form:"flag" comment:"标记"`       // 标记
	Remark    string           `form:"remark" comment:"备注"`     // 备注
	Admin     bool             `form:"admin" comment:"是否管理员"`
	DataScope string           `form:"dataScope"`
	SysMenu   []models.SysMenu `form:"sysMenu"`
	MenuIds   []int            `form:"menuIds"`
	SysDept   []models.SysDept `form:"sysDept"`
	DeptIds   []int            `form:"deptIds"`
	common.ControlBy
}

func (s *SysRoleInsertReq) Generate(model *models.SysRole) {
	if s.RoleId != 0 {
		model.RoleId = s.RoleId
	}
	model.RoleName = s.RoleName
	model.Status = s.Status
	model.RoleKey = s.RoleKey
	model.RoleSort = s.RoleSort
	model.Flag = s.Flag
	model.Remark = s.Remark
	model.Admin = s.Admin
	model.DataScope = s.DataScope
	model.SysMenu = &s.SysMenu
	model.SysDept = s.SysDept
}

func (s *SysRoleInsertReq) GetId() interface{} {
	return s.RoleId
}

type SysRoleUpdateReq struct {
	RoleId    int              `uri:"id" comment:"角色编码"`        // 角色编码
	RoleName  string           `form:"roleName" comment:"角色名称"` // 角色名称
	Status    string           `form:"status" comment:"状态"`     // 状态
	RoleKey   string           `form:"roleKey" comment:"角色代码"`  // 角色代码
	RoleSort  int              `form:"roleSort" comment:"角色排序"` // 角色排序
	Flag      string           `form:"flag" comment:"标记"`       // 标记
	Remark    string           `form:"remark" comment:"备注"`     // 备注
	Admin     bool             `form:"admin" comment:"是否管理员"`
	DataScope string           `form:"dataScope"`
	SysMenu   []models.SysMenu `form:"sysMenu"`
	MenuIds   []int            `form:"menuIds"`
	SysDept   []models.SysDept `form:"sysDept"`
	DeptIds   []int            `form:"deptIds"`
	common.ControlBy
}

func (s *SysRoleUpdateReq) Generate(model *models.SysRole) {
	if s.RoleId != 0 {
		model.RoleId = s.RoleId
	}
	model.RoleName = s.RoleName
	model.Status = s.Status
	model.RoleKey = s.RoleKey
	model.RoleSort = s.RoleSort
	model.Flag = s.Flag
	model.Remark = s.Remark
	model.Admin = s.Admin
	model.DataScope = s.DataScope
	model.SysMenu = &s.SysMenu
	model.SysDept = s.SysDept
}

func (s *SysRoleUpdateReq) GetId() interface{} {
	return s.RoleId
}

type UpdateStatusReq struct {
	RoleId int    `form:"roleId" comment:"角色编码"` // 角色编码
	Status string `form:"status" comment:"状态"`   // 状态
	common.ControlBy
}

func (s *UpdateStatusReq) Generate(model *models.SysRole) {
	if s.RoleId != 0 {
		model.RoleId = s.RoleId
	}
	model.Status = s.Status
}

func (s *UpdateStatusReq) GetId() interface{} {
	return s.RoleId
}

type SysRoleByName struct {
	RoleName string `form:"role"` // 角色编码
}

type SysRoleGetReq struct {
	Id int `uri:"id"`
}

func (s *SysRoleGetReq) GetId() interface{} {
	return s.Id
}

type SysRoleDeleteReq struct {
	Ids []int `json:"ids"`
}

func (s *SysRoleDeleteReq) GetId() interface{} {
	return s.Ids
}

// SysRoleDeleteRequest 删除角色请求（单个删除）
type SysRoleDeleteRequest struct {
	RoleId  int    `json:"role_id" binding:"required" comment:"角色 ID"` // 角色 ID（必填）
	Confirm bool   `json:"confirm" comment:"确认标识"`                     // 确认标识（选填）
	Reason  string `json:"reason" comment:"删除原因"`                      // 删除原因（选填）
	common.ControlBy
}

func (s *SysRoleDeleteRequest) GetId() interface{} {
	return s.RoleId
}

// SysRoleDeleteResponse 删除角色响应
type SysRoleDeleteResponse struct {
	RoleId         int         `json:"role_id"`         // 角色 ID
	RoleName       string      `json:"role_name"`       // 角色名称
	RoleCode       string      `json:"role_code"`       // 角色编码
	BackupInfo     *BackupInfo `json:"backup_info"`     // 备份信息
	AffectedAdmins int         `json:"affected_admins"` // 受影响的管理员数量
	Message        string      `json:"message"`         // 操作提示
}

// BackupInfo 备份信息
type BackupInfo struct {
	BackupTime   string     `json:"backup_time"`   // 备份时间
	BackupData   RoleBackup `json:"backup_data"`   // 备份数据
	RetentionDay int        `json:"retention_day"` // 保留期限（天）
}

// RoleBackup 角色备份数据
type RoleBackup struct {
	RoleId         int      `json:"role_id"`         // 角色 ID
	RoleName       string   `json:"role_name"`       // 角色名称
	RoleCode       string   `json:"role_code"`       // 角色编码
	Description    string   `json:"description"`     // 角色描述
	PermissionList []string `json:"permission_list"` // 权限标识列表
	Status         string   `json:"status"`          // 状态
	DeletedAt      string   `json:"deleted_at"`      // 删除时间
	DeletedBy      string   `json:"deleted_by"`      // 删除人
	Reason         string   `json:"reason"`          // 删除原因
}

// RoleDataScopeReq 角色数据权限修改
type RoleDataScopeReq struct {
	RoleId    int    `json:"roleId" binding:"required"`
	DataScope string `json:"dataScope" binding:"required"`
	DeptIds   []int  `json:"deptIds"`
}

func (s *RoleDataScopeReq) Generate(model *models.SysRole) {
	if s.RoleId != 0 {
		model.RoleId = s.RoleId
	}
	model.DataScope = s.DataScope
	model.DeptIds = s.DeptIds
}

type DeptIdList struct {
	DeptId int `json:"DeptId"`
}

// SysRoleListItem 角色列表项（包含统计信息）
type SysRoleListItem struct {
	RoleId          int      `json:"role_id"`          // 角色 ID
	RoleName        string   `json:"role_name"`        // 角色名称
	RoleCode        string   `json:"role_code"`        // 角色编码
	Description     string   `json:"description"`      // 角色描述
	Status          string   `json:"status"`           // 状态
	StatusText      string   `json:"status_text"`      // 状态文本
	PermissionCount int      `json:"permission_count"` // 权限数量
	PermissionList  []string `json:"permission_list"`  // 权限标识列表
	AdminCount      int      `json:"admin_count"`      // 关联管理员数量
	CreatedAt       string   `json:"created_at"`       // 创建时间
	CreatedBy       string   `json:"created_by"`       // 创建人
	UpdatedAt       string   `json:"updated_at"`       // 更新时间
	UpdatedBy       string   `json:"updated_by"`       // 更新人
}

// SysRoleListResponse 角色列表响应
type SysRoleListResponse struct {
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	List     []SysRoleListItem `json:"list"`
}

// SysRoleCreateReq 创建角色请求
type SysRoleCreateReq struct {
	RoleName       string   `json:"role_name" binding:"required,max=50" comment:"角色名称"`        // 角色名称（必填，不超过 50 字符）
	RoleCode       string   `json:"role_code" binding:"max=30" comment:"角色编码"`                 // 角色编码（选填，不超过 30 字符）
	Description    string   `json:"description" binding:"max=200" comment:"角色描述"`              // 角色描述（选填，不超过 200 字符）
	PermissionList []string `json:"permission_list" binding:"required,min=1" comment:"权限标识列表"` // 权限标识列表（必填，至少一个权限）
	Status         string   `json:"status" binding:"oneof=1 2" comment:"角色状态"`                 // 角色状态（选填，1:禁用 2:正常，默认 2）
	SourceRoleId   int      `json:"source_role_id" comment:"源角色 ID"`                           // 源角色 ID（选填，用于权限继承）
	common.ControlBy
}

func (s *SysRoleCreateReq) GetId() interface{} {
	return 0
}

// SysRoleCreateResponse 创建角色响应
type SysRoleCreateResponse struct {
	RoleId         int      `json:"role_id"`         // 角色 ID
	RoleName       string   `json:"role_name"`       // 角色名称
	RoleCode       string   `json:"role_code"`       // 角色编码
	PermissionList []string `json:"permission_list"` // 权限标识列表
	CreatedAt      string   `json:"created_at"`      // 创建时间
	Message        string   `json:"message"`         // 操作提示
}

// SysRoleUpdateRequest 更新角色请求（JSON）
type SysRoleUpdateRequest struct {
	RoleId         int      `json:"role_id" binding:"required" comment:"角色 ID"`                // 角色 ID（必填）
	RoleName       string   `json:"role_name" binding:"required,max=50" comment:"角色名称"`        // 角色名称（必填，不超过 50 字符）
	RoleCode       string   `json:"role_code" binding:"required,max=30" comment:"角色编码"`        // 角色编码（必填，不超过 30 字符）
	Description    string   `json:"description" binding:"max=200" comment:"角色描述"`              // 角色描述（选填，不超过 200 字符）
	PermissionList []string `json:"permission_list" binding:"required,min=1" comment:"权限标识列表"` // 权限标识列表（必填，至少一个权限）
	Status         string   `json:"status" binding:"oneof=1 2" comment:"角色状态"`                 // 角色状态（选填，1:禁用 2:正常）
	common.ControlBy
}

func (s *SysRoleUpdateRequest) GetId() interface{} {
	return s.RoleId
}

// SysRoleUpdateResponse 更新角色响应
type SysRoleUpdateResponse struct {
	RoleId         int      `json:"role_id"`         // 角色 ID
	RoleName       string   `json:"role_name"`       // 角色名称
	RoleCode       string   `json:"role_code"`       // 角色编码
	UpdatedFields  []string `json:"updated_fields"`  // 修改的字段列表
	UpdatedAt      string   `json:"updated_at"`      // 更新时间
	AffectedAdmins int      `json:"affected_admins"` // 受影响的管理员数量
	Message        string   `json:"message"`         // 操作提示
}
