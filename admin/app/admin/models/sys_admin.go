package models

import (
	"time"

	"go-admin/common/models"
)

// SysAdmin 物理表 public.sys_admin：控制台管理员唯一数据源（登录走 sys_user 视图时该视图应映射本表；冷启动注册与后台「创建管理员」均写入本表）
type SysAdmin struct {
	Id                int64          `gorm:"primaryKey;autoIncrement;comment:主键" json:"id"`
	Username          string         `gorm:"size:64;not null;comment:用户名" json:"username"`
	Password          string         `gorm:"size:255;not null;comment:密码" json:"-"`
	RealName          string         `gorm:"size:64;column:real_name;comment:真实姓名" json:"realName"`
	NickName          string         `gorm:"size:64;column:nick_name;comment:昵称" json:"nickName"`
	Mobile            string         `gorm:"size:20;comment:手机" json:"mobile"`
	Email             string         `gorm:"size:100;comment:邮箱" json:"email"`
	RoleId            int64          `gorm:"not null;index;column:role_id;comment:角色ID" json:"roleId"`
	RoleName          string         `gorm:"size:50;column:role_name;comment:角色名称" json:"roleName"`
	RoleCode          string         `gorm:"size:50;column:role_code;comment:角色标识" json:"roleCode"`
	Status            int16          `gorm:"default:1;index;comment:状态0禁用1启用" json:"status"`
	LoginCount        int            `gorm:"default:0;column:login_count;comment:登录次数" json:"loginCount"`
	LastLoginAt       *time.Time     `gorm:"column:last_login_at;comment:最后登录时间" json:"lastLoginAt"`
	LastLoginIP       string         `gorm:"size:50;column:last_login_ip;comment:最后登录IP" json:"lastLoginIp"`
	PasswordExpiredAt *time.Time     `gorm:"column:password_expired_at;comment:密码过期时间" json:"passwordExpiredAt"`
	PasswordChangedAt *time.Time     `gorm:"column:password_changed_at;comment:密码修改时间" json:"passwordChangedAt"`
	CreatedBy         *int64         `gorm:"column:created_by;comment:创建人" json:"createdBy"`
	DeptId            int            `gorm:"column:dept_id;default:0;comment:部门(go-admin兼容)" json:"deptId"`
	PostId            int            `gorm:"column:post_id;default:0;comment:岗位(go-admin兼容)" json:"postId"`
	Remark            string         `gorm:"size:255;column:remark;comment:备注" json:"remark"`
	Avatar            string         `gorm:"size:255;column:avatar;comment:头像" json:"avatar"`
	Salt              string         `gorm:"size:255;column:salt;comment:盐" json:"-"`
	UpdateBy          *int64         `gorm:"column:update_by;comment:更新人" json:"updateBy"`

	// 迁移 082 扩展：安全策略 / 强制改密 —— 均可为空，留空表示不限制/无要求
	AllowedIPs             string     `gorm:"column:allowed_ips;type:text;comment:登录 IP 白名单(逗号/CIDR)" json:"allowedIps"`
	AllowedLoginStart      *string    `gorm:"column:allowed_login_start;comment:允许登录时间窗起点(HH:MM:SS)" json:"allowedLoginStart"`
	AllowedLoginEnd        *string    `gorm:"column:allowed_login_end;comment:允许登录时间窗终点(HH:MM:SS)" json:"allowedLoginEnd"`
	MustChangePassword     bool       `gorm:"column:must_change_password;default:false;comment:是否要求下次登录后强制改密" json:"mustChangePassword"`
	LastPasswordChangedAt  *time.Time `gorm:"column:last_password_changed_at;comment:最近一次改密时间" json:"lastPasswordChangedAt"`
	models.ModelTime
}

func (*SysAdmin) TableName() string {
	return "sys_admin"
}

func (e *SysAdmin) Generate() models.ActiveRecord {
	o := *e
	return &o
}

func (e *SysAdmin) GetId() interface{} {
	return e.Id
}

func (e *SysAdmin) SetCreateBy(createBy int) {
	if createBy == 0 {
		e.CreatedBy = nil
		return
	}
	v := int64(createBy)
	e.CreatedBy = &v
}

func (e *SysAdmin) SetUpdateBy(updateBy int) {
	if updateBy == 0 {
		e.UpdateBy = nil
		return
	}
	v := int64(updateBy)
	e.UpdateBy = &v
}
