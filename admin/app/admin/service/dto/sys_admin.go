package dto

import (
	"go-admin/common/dto"
	common "go-admin/common/models"
)

// SysAdminGetPageReq 管理员列表查询请求
type SysAdminGetPageReq struct {
	dto.Pagination `search:"-"`

	Keyword       string `form:"keyword" search:"type:contains;column:username;table:sys_user" comment:"关键词"`              // 关键词
	RoleId        int    `form:"role_id" search:"type:exact;column:role_id;table:sys_user_role" comment:"角色 ID"`           // 角色 ID
	Status        string `form:"status" search:"type:exact;column:status;table:sys_user" comment:"状态"`                     // 状态
	LastLoginFrom string `form:"last_login_from" search:"type:gte;column:last_login_time;table:sys_user" comment:"最后登录起始"` // 最后登录起始
	LastLoginTo   string `form:"last_login_to" search:"type:lte;column:last_login_time;table:sys_user" comment:"最后登录结束"`   // 最后登录结束
	SortBy        string `form:"sort_by" search:"-" comment:"排序字段"`                                                        // 排序字段
	SortOrder     string `form:"sort_order" search:"-" comment:"排序方式"`                                                     // 排序方式
}

func (s *SysAdminGetPageReq) GetPageIndex() int {
	if s.PageIndex <= 0 {
		return 1
	}
	return s.PageIndex
}

func (s *SysAdminGetPageReq) GetPageSize() int {
	if s.PageSize <= 0 {
		return 20
	}
	return s.PageSize
}

func (s *SysAdminGetPageReq) GetNeedSearch() interface{} {
	return *s
}

// SysAdminDetailReq 管理员详情请求
type SysAdminDetailReq struct {
	Id int `uri:"id" swaggerignore:"true" comment:"管理员 ID"` // 管理员 ID
}

func (s *SysAdminDetailReq) GetId() interface{} {
	return s.Id
}

// SysAdminListItem 管理员列表项
type SysAdminListItem struct {
	AdminId            int                `json:"admin_id"`               // 管理员 ID
	UserId             int                `json:"user_id"`                // 用户 ID
	Username           string             `json:"username"`               // 用户名
	Nickname           string             `json:"nickname"`               // 昵称姓名
	RealName           string             `json:"real_name"`              // 真实姓名
	Email              string             `json:"email"`                  // 邮箱
	Phone              string             `json:"phone"`                  // 手机号（脱敏）
	PhoneRaw           string             `json:"-"`                      // 原始手机号
	Avatar             string             `json:"avatar"`                 // 头像 URL
	DeptId             int                `json:"dept_id"`                // 部门 ID（0 表示未分配）
	DeptName           string             `json:"dept_name"`              // 部门名称
	RoleList           []SysAdminRoleItem `json:"role_list"`              // 关联角色列表
	Status             string             `json:"status"`                 // 管理员状态（1:禁用 2:正常）
	StatusText         string             `json:"status_text"`            // 状态文本
	LastLoginTime      string             `json:"last_login_time"`        // 最后登录时间
	LastLoginIp        string             `json:"last_login_ip"`          // 最后登录 IP
	LoginCount         int                `json:"login_count"`            // 登录次数
	MustChangePassword bool               `json:"must_change_password"`   // 下次登录是否强制改密
	CreatedAt          string             `json:"created_at"`             // 创建时间
	CreatedBy          string             `json:"created_by"`             // 创建人
	UpdatedAt          string             `json:"updated_at"`             // 更新时间
	IsSuper            bool               `json:"is_super"`               // 是否超级管理员
}

// SysAdminRoleItem 管理员角色项
type SysAdminRoleItem struct {
	RoleId   int    `json:"role_id"`   // 角色 ID
	RoleName string `json:"role_name"` // 角色名称
	RoleCode string `json:"role_code"` // 角色编码
}

// SysAdminDetail 管理员详情
type SysAdminDetail struct {
	AdminId               int                `json:"admin_id"`                  // 管理员 ID
	UserId                int                `json:"user_id"`                   // 用户 ID
	Username              string             `json:"username"`                  // 用户名
	Nickname              string             `json:"nickname"`                  // 昵称姓名
	RealName              string             `json:"real_name"`                 // 真实姓名
	Email                 string             `json:"email"`                     // 邮箱
	Phone                 string             `json:"phone"`                     // 手机号
	Avatar                string             `json:"avatar"`                    // 头像 URL
	DeptId                int                `json:"dept_id"`                   // 部门 ID
	DeptName              string             `json:"dept_name"`                 // 部门名称
	RoleList              []SysAdminRoleItem `json:"role_list"`                 // 关联角色列表
	RoleIds               []int              `json:"role_ids"`                  // 角色 ID 列表
	Status                string             `json:"status"`                    // 管理员状态
	StatusText            string             `json:"status_text"`               // 状态文本
	LastLoginTime         string             `json:"last_login_time"`           // 最后登录时间
	LastLoginIp           string             `json:"last_login_ip"`             // 最后登录 IP
	LoginCount            int                `json:"login_count"`               // 登录次数
	Remark                string             `json:"remark"`                    // 备注
	AllowedIps            string             `json:"allowed_ips"`               // IP 白名单（逗号分隔/CIDR）
	AllowedLoginStart     string             `json:"allowed_login_start"`       // 允许登录时间窗起点 HH:MM
	AllowedLoginEnd       string             `json:"allowed_login_end"`         // 允许登录时间窗终点 HH:MM
	MustChangePassword    bool               `json:"must_change_password"`      // 下次登录后是否强制改密
	LastPasswordChangedAt string             `json:"last_password_changed_at"`  // 最近一次改密时间
	CreatedAt             string             `json:"created_at"`                // 创建时间
	CreatedBy             string             `json:"created_by"`                // 创建人
	UpdatedAt             string             `json:"updated_at"`                // 更新时间
	UpdatedBy             string             `json:"updated_by"`                // 更新人
	IsSuper               bool               `json:"is_super"`                  // 是否超级管理员
}

// SysAdminCreateReq 创建管理员请求
type SysAdminCreateReq struct {
	Username string `json:"username" binding:"required,min=6,max=20" comment:"用户名"` // 用户名（必填）
	Password string `json:"password" binding:"required,min=8,max=20" comment:"密码"`  // 密码（必填）
	Nickname string `json:"nickname" binding:"required,max=50" comment:"昵称"`        // 昵称（必填）
	RealName string `json:"real_name" binding:"omitempty,max=64" comment:"真实姓名"`    // 真实姓名（选填）
	Email    string `json:"email" binding:"omitempty,email,max=100" comment:"邮箱"`   // 邮箱（选填）
	Phone    string `json:"phone" binding:"omitempty,len=11" comment:"手机号"`         // 手机号（选填）
	Avatar   string `json:"avatar" binding:"omitempty,max=255" comment:"头像 URL"`    // 头像（选填）
	DeptId   int    `json:"dept_id" binding:"omitempty,min=0" comment:"部门 ID"`      // 部门 ID（选填）
	RoleIds  []int  `json:"role_ids" binding:"required,min=1" comment:"角色 ID 列表"`   // 角色 ID 列表（必填）
	Status   string `json:"status" binding:"omitempty,oneof=1 2" comment:"状态"`      // 状态（选填，1:禁用 2:正常）
	Remark   string `json:"remark" binding:"omitempty,max=255" comment:"备注"`        // 备注（选填）
	// 创建后强制首次登录改密（默认 true，保障密码只有本人知道）
	MustChangePassword *bool `json:"must_change_password" comment:"是否在首次登录后强制改密"`
	common.ControlBy
}

func (s *SysAdminCreateReq) GetId() interface{} {
	return 0
}

// SysAdminUpdateReq 更新管理员请求
type SysAdminUpdateReq struct {
	// UserId 不再要求前端在 JSON body 传递；优先由路由 /:id 注入，避免篡改。
	UserId   int    `json:"user_id" uri:"id" comment:"用户 ID"`  // 用户 ID（URI 或 Body）
	Nickname string `json:"nickname" binding:"required,max=50" comment:"昵称"`      // 昵称
	RealName string `json:"real_name" binding:"omitempty,max=64" comment:"真实姓名"`  // 真实姓名
	Email    string `json:"email" binding:"omitempty,email,max=100" comment:"邮箱"` // 邮箱
	Phone    string `json:"phone" binding:"omitempty,len=11" comment:"手机号"`       // 手机号
	Avatar   string `json:"avatar" binding:"omitempty,max=255" comment:"头像 URL"`  // 头像
	DeptId   int    `json:"dept_id" binding:"omitempty,min=0" comment:"部门 ID"`    // 部门 ID
	RoleIds  []int  `json:"role_ids" binding:"required,min=1" comment:"角色 ID 列表"` // 角色 ID 列表
	Status   string `json:"status" binding:"omitempty,oneof=1 2" comment:"状态"`    // 状态
	Remark   string `json:"remark" binding:"omitempty,max=255" comment:"备注"`      // 备注
	// ActorIsCasbinBypass 由 handler 从 JWT 推导写入，不接受客户端 JSON（避免伪造提权）。
	ActorIsCasbinBypass bool `json:"-" swaggerignore:"true"`
	common.ControlBy
}

func (s *SysAdminUpdateReq) GetId() interface{} {
	return s.UserId
}

// SysAdminDeleteReq 删除管理员请求
type SysAdminDeleteReq struct {
	UserId  int    `json:"user_id" binding:"required" comment:"用户 ID"` // 用户 ID（必填）
	Confirm *bool  `json:"confirm" comment:"确认标识"`                     // 确认标识（选填）
	Reason  string `json:"reason" comment:"删除原因"`                      // 删除原因（选填）
	common.ControlBy
}

func (s *SysAdminDeleteReq) GetId() interface{} {
	return s.UserId
}

// SysAdminDeleteResponse 删除管理员响应
type SysAdminDeleteResponse struct {
	Success   bool   `json:"success"`    // 是否成功
	AdminId   int    `json:"admin_id"`   // 管理员 ID
	Username  string `json:"username"`   // 用户名
	DeletedAt string `json:"deleted_at"` // 删除时间
	Message   string `json:"message"`    // 操作提示
}

// SysAdminStatusReq 更新管理员状态请求
type SysAdminStatusReq struct {
	UserId int    `json:"user_id" uri:"id" comment:"用户 ID"` // 用户 ID（URI 或 Body）
	Status string `json:"status" binding:"required,oneof=1 2" comment:"状态"`    // 状态（1:禁用 2:正常）
	common.ControlBy
}

func (s *SysAdminStatusReq) GetId() interface{} {
	return s.UserId
}

// ===== 批量删除 =====

// SysAdminBatchDeleteReq 批量删除请求
type SysAdminBatchDeleteReq struct {
	UserIds []int  `json:"user_ids" binding:"required,min=1,max=100" comment:"管理员 ID 数组"` // 必填
	Reason  string `json:"reason" comment:"删除原因"`
	common.ControlBy
}

// SysAdminBatchDeleteResp 批量删除响应
type SysAdminBatchDeleteResp struct {
	Total   int      `json:"total"`          // 请求总数
	Success int      `json:"success"`        // 成功数
	Failed  int      `json:"failed"`         // 失败数
	Fails   []string `json:"fails,omitempty"` // 失败明细 "id:message"
}

// ===== 重置 / 修改密码 =====

// SysAdminResetPasswordReq 超管重置其他管理员密码请求
type SysAdminResetPasswordReq struct {
	UserId      int    `json:"user_id" uri:"id" comment:"管理员 ID"`      // URI
	NewPassword string `json:"new_password" binding:"required,min=8,max=20" comment:"新密码"` // 必填
	// 是否要求其下次登录后再次强制改密（默认 true，防止泄露新密码）
	RequireChangeOnLogin *bool `json:"require_change_on_login" comment:"是否下次登录后强制改密"`
	common.ControlBy
}

// SysAdminChangePasswordReq 自助修改密码请求（已登录 admin 调用）
type SysAdminChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required,min=1,max=100" comment:"旧密码"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=20" comment:"新密码"`
}

// ===== 安全策略 =====

// SysAdminSecurityReq IP 白名单 + 登录时间窗
type SysAdminSecurityReq struct {
	UserId            int    `json:"user_id" uri:"id" comment:"管理员 ID"` // URI
	AllowedIps        string `json:"allowed_ips" comment:"IP/CIDR 白名单，逗号分隔；留空表示不限"`
	AllowedLoginStart string `json:"allowed_login_start" comment:"允许登录起点 HH:MM，留空表示不限"`
	AllowedLoginEnd   string `json:"allowed_login_end" comment:"允许登录终点 HH:MM，留空表示不限"`
	common.ControlBy
}

// SysAdminForceChangeReq 设置某管理员下次登录必须改密
type SysAdminForceChangeReq struct {
	UserId int  `json:"user_id" uri:"id" comment:"管理员 ID"`
	Must   bool `json:"must" comment:"是否强制"`
	common.ControlBy
}
