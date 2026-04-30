package dto

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"go-admin/common/dto"
	common "go-admin/common/models"
)

// PlatformUserListReq 平台用户列表查询请求
type PlatformUserListReq struct {
	dto.Pagination    `search:"-"`
	UserId            int64  `form:"userId" search:"type:exact;column:id;table:users" comment:"用户 ID"`
	Username          string `form:"username" search:"type:contains;column:username;table:users" comment:"用户名"`
	RealName          string `form:"real_name" search:"type:contains;column:real_name;table:users" comment:"姓名"`
	Mobile            string `form:"mobile" search:"type:contains;column:mobile;table:users" comment:"手机号"`
	Nickname          string `form:"nickname" search:"type:contains;column:nickname;table:users" comment:"昵称"`
	Email             string `form:"email" search:"type:contains;column:email;table:users" comment:"邮箱"`
	Gender            *int32 `form:"gender" search:"type:exact;column:gender;table:users" comment:"性别"`
	Status            *int32 `form:"status" search:"type:exact;column:status;table:users" comment:"账号状态 0 禁用 1 正常"`
	MemberLevel       *int32 `form:"memberLevel" search:"-"` // 会员档位在 user_member，避免引用不存在的 users.member_level
	RealNameStatus    *int32 `form:"realNameStatus" search:"type:exact;column:real_name_status;table:users" comment:"实名状态"`
	RegisterTimeStart string `form:"registerTimeStart" search:"type:gte;column:created_at;table:users" comment:"注册时间开始"`
	RegisterTimeEnd   string `form:"registerTimeEnd" search:"type:lte;column:created_at;table:users" comment:"注册时间结束"`
	PlatformUserOrder
}

type PlatformUserOrder struct {
	UserIdOrder      string `search:"type:order;column:id;table:users" form:"userIdOrder"`
	NicknameOrder    string `search:"type:order;column:nickname;table:users" form:"nicknameOrder"`
	MobileOrder      string `search:"type:order;column:mobile;table:users" form:"mobileOrder"`
	StatusOrder      string `search:"type:order;column:status;table:users" form:"statusOrder"`
	// 会员档位不在 users 表；避免生成 ORDER BY users.member_level 导致 SQL 报错
	MemberLevelOrder string `search:"-" form:"memberLevelOrder"`
	CreatedAtOrder   string `search:"type:order;column:created_at;table:users" form:"createdAtOrder"`
}

func (m *PlatformUserListReq) GetNeedSearch() interface{} {
	return *m
}

// PlatformUserListItem 平台用户列表项
type PlatformUserListItem struct {
	UserId          int64     `json:"user_id"`           // 用户 ID
	Username        string    `json:"username"`          // 用户名
	RealName        string    `json:"real_name"`         // 姓名
	Mobile          string    `json:"mobile"`            // 手机号
	Email           string    `json:"email"`             // 邮箱
	Nickname        string    `json:"nickname"`          // 昵称
	Avatar          string    `json:"avatar"`            // 头像
	Gender          int32     `json:"gender"`            // 性别
	RoleNames       string    `json:"role_names"`        // 所属角色（逗号分隔）
	MemberLevel     int32     `json:"member_level"`      // 会员等级
	MemberLevelName string    `json:"member_level_name"` // 会员等级名称
	MemberExpireAt  int64     `json:"member_expire_at"`  // 会员过期时间戳
	Status          int32     `json:"status"`            // 账号状态 0 禁用 1 正常
	RealNameStatus  int32     `json:"real_name_status"`  // 实名状态
	BindDeviceCount int64     `json:"bind_device_count"` // 绑定设备数量
	RegisterTime    int64     `json:"register_time"`     // 注册时间戳
	LastLoginTime   int64     `json:"last_login_time"`   // 最后登录时间戳
	CreatedAt       time.Time `json:"created_at"`        // 创建时间
	UpdatedAt       time.Time `json:"updated_at"`        // 更新时间
}

// PlatformUserListResp 平台用户列表响应
type PlatformUserListResp struct {
	List     []PlatformUserListItem `json:"list"`     // 用户列表
	Total    int64                  `json:"total"`    // 总条数
	Page     int                    `json:"page"`     // 当前页码
	PageSize int                    `json:"pageSize"` // 每页条数
}

// PlatformUserCreateReq 创建平台用户请求
//
// 约束：
//  1. 入口是后台「平台用户」模块，仅允许超级管理员调用；
//  2. 新增成功后数据只写入 public.users 与 public.user_role_rel，不会涉及 sys_admin；
//  3. role_ids 只接受「普通用户」(roles.slug = 'user') 的角色；传入其他角色将被忽略并强制落为 user。
type PlatformUserCreateReq struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	RealName string  `json:"real_name"`
	Nickname string  `json:"nickname"`
	Mobile   string  `json:"mobile"`
	Email    string  `json:"email"`
	Avatar   string  `json:"avatar"`
	Gender   *int32  `json:"gender"`
	Birthday *string `json:"birthday"`
	Status   *int32  `json:"status"`
	RoleIds  []int64 `json:"role_ids"`
	common.ControlBy
}

func (s *PlatformUserCreateReq) GetId() interface{} {
	return 0
}

// PlatformUserUpdateReq 更新平台用户请求
type PlatformUserUpdateReq struct {
	UserId   int64   `uri:"userId" validate:"required"`
	Nickname string  `json:"nickname" validate:"omitempty,max=100"`
	Avatar   string  `json:"avatar" validate:"omitempty,max=500"`
	Status   *int32  `json:"status" validate:"omitempty,oneof=0 1"`
	RealName string  `json:"real_name" validate:"omitempty,max=64"`
	Mobile   string  `json:"mobile" validate:"omitempty,mobile"`
	Email    string  `json:"email" validate:"omitempty,email"`
	Gender   *int32  `json:"gender" validate:"omitempty,oneof=0 1 2"`
	Birthday *string `json:"birthday" validate:"omitempty"`
	common.ControlBy
}

func (s *PlatformUserUpdateReq) GetId() interface{} {
	return s.UserId
}

// PlatformUserGetInfoReq 获取用户详情请求
type PlatformUserGetInfoReq struct {
	UserId int64 `uri:"userId" validate:"required"`
}

func (s *PlatformUserGetInfoReq) GetId() interface{} {
	return s.UserId
}

// PlatformUserRoleItem 平台用户角色（与前端编辑 role_ids 对齐）
type PlatformUserRoleItem struct {
	ID      int64  `json:"id"`
	RoleKey string `json:"roleKey"`
	Name    string `json:"name"`
}

// PlatformUserInfoResp 用户详情响应
type PlatformUserInfoResp struct {
	// 基础信息
	UserId         int64     `json:"user_id"` // 用户 ID（物理主键 id）
	Username       string    `json:"username"`
	RealName       string    `json:"real_name"`
	Mobile         string    `json:"mobile"` // 手机号（脱敏）
	Email          string    `json:"email"`  // 邮箱
	Nickname       string    `json:"nickname"` // 昵称
	Avatar         string    `json:"avatar"`     // 头像
	Gender         int32     `json:"gender"`
	Birthday       string    `json:"birthday"` // yyyy-MM-dd，由服务层从 DB 填充
	Status         int32     `json:"status"` // 账号状态 0 禁用 1 正常
	RealNameStatus int32     `json:"real_name_status"` // 实名状态
	Roles          []PlatformUserRoleItem `json:"roles"`
	RegisterTime   int64     `json:"register_time"`    // 注册时间戳
	LastLoginTime  int64     `json:"last_login_time"`  // 最后登录时间戳
	LastLoginIP    string    `json:"last_login_ip"`    // 最后登录 IP
	CreatedAt      time.Time `json:"created_at"`       // 创建时间
	UpdatedAt      time.Time `json:"updated_at"`       // 更新时间

	// 会员信息
	MemberLevel     int32  `json:"member_level"`      // 会员等级
	MemberLevelName string `json:"member_level_name"` // 会员等级名称
	MemberExpireAt  int64  `json:"member_expire_at"`  // 会员过期时间戳
	MemberStatus    int32  `json:"member_status"`     // 会员状态 0-正常 1-过期 2-冻结
	MemberCreatedAt int64  `json:"member_created_at"` // 会员开通时间戳

	// 设备信息
	BindDeviceCount   int64            `json:"bind_device_count"`   // 绑定设备总数
	OnlineDeviceCount int64            `json:"online_device_count"` // 在线设备数
	DeviceList        []UserDeviceItem `json:"device_list"`         // 设备列表

	// 实名信息
	RealNameInfo *RealNameInfo `json:"real_name_info,omitempty"` // 实名认证信息

	// 会话信息
	ActiveSessionCount int64         `json:"active_session_count"` // 当前有效会话数
	RecentLogins       []RecentLogin `json:"recent_logins"`        // 最近登录记录
}

// UserDeviceItem 用户设备信息
type UserDeviceItem struct {
	DeviceSn   string `json:"device_sn"`   // 设备 SN
	DeviceName string `json:"device_name"` // 设备名称
	Model      string `json:"model"`       // 设备型号
	Online     bool   `json:"online"`      // 是否在线
	BindTime   int64  `json:"bind_time"`   // 绑定时间戳
}

// RealNameInfo 实名认证信息
type RealNameInfo struct {
	Status      int32  `json:"status"`       // 实名状态 0-未提交 1-审核中 2-已通过 3-已驳回
	RealName    string `json:"real_name"`    // 实名姓名（脱敏）
	IDCard      string `json:"id_card"`      // 身份证号（脱敏）
	SubmitTime  int64  `json:"submit_time"`  // 提交时间戳
	AuditTime   int64  `json:"audit_time"`   // 审核时间戳
	Auditor     string `json:"auditor"`      // 审核人
	AuditRemark string `json:"audit_remark"` // 审核意见
}

// UserInfo 用户信息（用于 getinfo 接口）
type UserInfo struct {
	Roles       []string `json:"roles"`        // 角色列表
	Name        string   `json:"name"`         // 用户名
	Avatar      string   `json:"avatar"`       // 头像
	Intro       string   `json:"introduction"` // 简介
	Permissions []string `json:"permissions"`  // 权限列表
}

// RecentLogin 最近登录记录
type RecentLogin struct {
	LoginTime int64  `json:"login_time"` // 登录时间戳
	Device    string `json:"device"`     // 登录设备
	IP        string `json:"ip"`         // 登录 IP
	Location  string `json:"location"`   // 登录地点
}

// UpdateUserStatusReq 更新用户状态请求
type UpdateUserStatusReq struct {
	UserId int64  `json:"userId"`   // 目标用户 ID
	Status int32  `json:"status"`   // 0=禁用 1=启用
	Reason string `json:"reason"`   // 操作原因
}

// UnmarshalJSON 兼容 userId/user_id、数字与字符串，避免前端大整数或序列化差异导致 ShouldBindJSON 失败
func (r *UpdateUserStatusReq) UnmarshalJSON(data []byte) error {
	var raw struct {
		UserId  interface{} `json:"userId"`
		UserID2 interface{} `json:"user_id"`
		Status  interface{} `json:"status"`
		Reason  string      `json:"reason"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	uid := flexInt64(raw.UserId)
	if uid == 0 {
		uid = flexInt64(raw.UserID2)
	}
	r.UserId = uid
	r.Status = int32(flexInt(raw.Status))
	r.Reason = raw.Reason
	return nil
}

func flexInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return int64(x)
	case json.Number:
		i, err := x.Int64()
		if err != nil {
			f, _ := x.Float64()
			return int64(f)
		}
		return i
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		if err != nil {
			return 0
		}
		return i
	default:
		return 0
	}
}

func flexInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return int(x)
	case json.Number:
		i, err := x.Int64()
		if err != nil {
			f, _ := x.Float64()
			return int(f)
		}
		return int(i)
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(x))
		if err != nil {
			return 0
		}
		return i
	default:
		return 0
	}
}

func (s *UpdateUserStatusReq) GetId() interface{} {
	return s.UserId
}

// UpdateUserStatusResp 更新用户状态响应
type UpdateUserStatusResp struct {
	UserId     int64     `json:"user_id"`     // 用户 ID
	Status     int32     `json:"status"`      // 用户状态
	StatusText string    `json:"status_text"` // 状态文本
	UpdateTime time.Time `json:"update_time"` // 更新时间
}

// UpdateUserVipLevelReq 修改用户会员等级请求
type UpdateUserVipLevelReq struct {
	UserId        int64  `json:"user_id" validate:"required"`                 // 用户 ID
	VipLevel      int32  `json:"vip_level" validate:"required,oneof=0 1 2 3"` // 目标会员等级
	VipExpireTime string `json:"vip_expire_time" validate:"omitempty"`        // 到期时间
	Reason        string `json:"reason" validate:"omitempty,max=500"`         // 修改原因
}

func (s *UpdateUserVipLevelReq) GetId() interface{} {
	return s.UserId
}

// UpdateUserVipLevelResp 修改用户会员等级响应
type UpdateUserVipLevelResp struct {
	UserId        int64     `json:"user_id"`         // 用户 ID
	VipLevel      int32     `json:"vip_level"`       // 会员等级
	VipName       string    `json:"vip_name"`        // 等级名称
	VipExpireTime string    `json:"vip_expire_time"` // 到期时间
	UpdateTime    time.Time `json:"update_time"`     // 修改时间
}
