package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go-admin/app/admin/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 管理员账号注册模块专用错误（与 setup 冷启动错误区分，方便 API 层分类映射）
var (
	ErrAdminUsernameTaken = errors.New("admin username already exists in sys_admin")
	ErrAdminRoleNotFound  = errors.New("admin role not found in roles table")
	ErrAdminInvalidRole   = errors.New("invalid admin role slug")
)

// AdminRegisterInput 管理员账号注册入参
// 模块边界：仅供 /api/admin/account/register 使用；所有字段最终只写入 public.sys_admin。
type AdminRegisterInput struct {
	Username string
	Password string
	Nickname string
	RealName string
	Mobile   string
	Email    string
	// RoleSlug: super_admin / operator / finance（空值默认 super_admin）
	RoleSlug string
}

// RegisterAdminAccount 注册一条管理员记录到 public.sys_admin。
//
// 与用户体系的隔离保证：
//  1. 用户名唯一性仅在 public.sys_admin 命名空间内校验；C 端 public.users 中的同名账号不会造成冲突
//  2. 角色查询只读 public.roles；不写入 user_role_rel 或任何 C 端关联表
//  3. 不调用任何与 public.users 相关的 service，所有副作用限制在 sys_admin 单表
func RegisterAdminAccount(db *gorm.DB, in AdminRegisterInput) (*models.SysAdmin, error) {
	if db == nil {
		return nil, errors.New("database not ready")
	}
	if err := validateUsername(in.Username); err != nil {
		return nil, err
	}
	if err := validatePasswordPolicy(in.Password); err != nil {
		return nil, err
	}

	nick := strings.TrimSpace(in.Nickname)
	if nick == "" {
		nick = strings.TrimSpace(in.Username)
	}

	rs := strings.TrimSpace(strings.ToLower(in.RoleSlug))
	if rs == "" {
		rs = "super_admin"
	}
	if !isSetupRegisterableSlug(rs) {
		return nil, ErrAdminInvalidRole
	}

	sysAdminTable := setupSysAdminTable(db)
	rolesTable := setupRolesTable(db)

	var dup int64
	if err := db.Raw(fmt.Sprintf(`SELECT COUNT(1) FROM %s WHERE deleted_at IS NULL AND LOWER(TRIM(username)) = LOWER(TRIM(?))`, sysAdminTable), in.Username).Scan(&dup).Error; err != nil {
		return nil, err
	}
	if dup > 0 {
		return nil, ErrAdminUsernameTaken
	}

	var role struct {
		Id   int64
		Name string
	}
	if err := db.Raw(fmt.Sprintf(`SELECT id, COALESCE(TRIM(name), '') AS name FROM %s WHERE LOWER(TRIM(slug)) = ? LIMIT 1`, rolesTable), rs).Scan(&role).Error; err != nil {
		return nil, err
	}
	if role.Id == 0 {
		return nil, ErrAdminRoleNotFound
	}
	if strings.TrimSpace(role.Name) == "" {
		role.Name = rs
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	row := models.SysAdmin{
		Username:          strings.TrimSpace(in.Username),
		Password:          string(hash),
		NickName:          nick,
		RealName:          strings.TrimSpace(in.RealName),
		Mobile:            strings.TrimSpace(in.Mobile),
		Email:             strings.TrimSpace(in.Email),
		RoleId:            role.Id,
		RoleName:          truncateStr(role.Name, 50),
		RoleCode:          truncateStr(rs, 50),
		Status:            1,
		PasswordChangedAt: &now,
	}

	// public.sys_admin 的物理迁移（migration 081）未建 dept_id/post_id/remark/avatar/salt/update_by 这些 go-admin
	// 兼容字段；如果交给 GORM 默认 Create 会触发 42703 “字段不存在”。这里显式 Omit，保持只写实际存在的列。
	err = db.Transaction(func(tx *gorm.DB) error {
		return tx.Table(sysAdminTable).
			Omit("DeptId", "PostId", "Remark", "Avatar", "Salt", "UpdateBy").
			Create(&row).Error
	})
	if err != nil {
		return nil, err
	}
	return &row, nil
}
