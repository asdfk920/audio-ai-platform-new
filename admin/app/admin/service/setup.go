package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"go-admin/app/admin/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrSetupNotNeeded     = errors.New("setup not needed")
	ErrWeakPassword       = errors.New("password does not meet policy")
	ErrInvalidUsername    = errors.New("invalid username")
	ErrUsernameTaken      = errors.New("username already exists")
	ErrSuperAdminRole     = errors.New("super_admin role not found") // 兼容旧错误名
	ErrSetupRoleNotFound  = errors.New("selected role not found in database")
	ErrInvalidSetupRole   = errors.New("invalid setup role slug")
)

// SetupRegisterableSlugs 冷启动注册时允许选择的角色（与业务约定一致）
var SetupRegisterableSlugs = []string{"super_admin", "operator", "finance"}

func isSetupRegisterableSlug(slug string) bool {
	s := strings.TrimSpace(strings.ToLower(slug))
	for _, x := range SetupRegisterableSlugs {
		if s == x {
			return true
		}
	}
	return false
}

// #region agent log
func agentDebugSetupNDJSON(hypothesisID, location, message string, data map[string]any) {
	const p = "c:/Users/Lenovo/Desktop/audio-ai-platform/debug-a35f07.log"
	line := map[string]any{
		"sessionId":    "a35f07",
		"runId":        "setup-status",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	b, err := json.Marshal(line)
	if err != nil {
		return
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(b, '\n'))
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func setupSysAdminTable(db *gorm.DB) string {
	if db != nil && db.Dialector != nil && db.Dialector.Name() == "postgres" {
		return "public.sys_admin"
	}
	return "sys_admin"
}

func setupRolesTable(db *gorm.DB) string {
	if db != nil && db.Dialector != nil && db.Dialector.Name() == "postgres" {
		return "public.roles"
	}
	return "roles"
}

// #endregion

// NeedsSetup true 表示库中尚无可登录后台的管理员账号（仅看物理表 sys_admin，与 C 端 users 分离）
func NeedsSetup(db *gorm.DB) (bool, error) {
	sysAdminTable := setupSysAdminTable(db)
	var n int64
	err := db.Table(sysAdminTable).Where("deleted_at IS NULL").Count(&n).Error
	var rawN, totalN int64
	var rawErr, totalErr error
	if err == nil {
		rawErr = db.Raw(fmt.Sprintf(`SELECT COUNT(1) FROM %s WHERE deleted_at IS NULL`, sysAdminTable)).Scan(&rawN).Error
		totalErr = db.Raw(fmt.Sprintf(`SELECT COUNT(1) FROM %s`, sysAdminTable)).Scan(&totalN).Error
	}
	// #region agent log
	// H2 额外诊断：确认后端实际连的是哪个 DB/schema，以及 public.sys_admin 究竟是表还是视图
	var dbName, curSchema, searchPath, relKind, relNamespace string
	var pgCheckErr1, pgCheckErr2 error
	if db != nil && db.Dialector != nil && db.Dialector.Name() == "postgres" {
		pgCheckErr1 = db.Raw(`SELECT current_database() || '|' || current_schema() || '|' || current_setting('search_path')`).Scan(&dbName).Error
		if idx := strings.Index(dbName, "|"); idx >= 0 {
			curSchema = dbName[idx+1:]
			dbName = dbName[:idx]
		}
		if idx := strings.Index(curSchema, "|"); idx >= 0 {
			searchPath = curSchema[idx+1:]
			curSchema = curSchema[:idx]
		}
		var rk struct {
			Kind string
			Ns   string
		}
		pgCheckErr2 = db.Raw(`SELECT c.relkind::text AS kind, n.nspname AS ns FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relname='sys_admin' ORDER BY (n.nspname='public') DESC LIMIT 1`).Scan(&rk).Error
		relKind = rk.Kind
		relNamespace = rk.Ns
	}
	agentDebugSetupNDJSON("H1H2H3", "service/setup.go:NeedsSetup", "sys_admin counts + DB identity (H1 restart/H2 wrong DB/H3 view)", map[string]any{
		"gormN":             n,
		"gormErr":           errString(err),
		"rawNullDeletedN":   rawN,
		"rawErr":            errString(rawErr),
		"totalRowsN":        totalN,
		"totalErr":          errString(totalErr),
		"needsSetupByGorm":  n == 0 && err == nil,
		"table":             sysAdminTable,
		"currentDatabase":   dbName,
		"currentSchema":     curSchema,
		"searchPath":        searchPath,
		"sysAdminRelKind":   relKind,
		"sysAdminNamespace": relNamespace,
		"pgIdentityErr":     errString(pgCheckErr1),
		"pgRelKindErr":      errString(pgCheckErr2),
	})
	// #endregion
	if err != nil {
		// 如果表不存在或查询失败，说明系统尚未初始化，需要注册
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			return true, nil
		}
		return false, err
	}
	return n == 0, nil
}

var usernameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]{2,63}$`)

func validateUsername(u string) error {
	u = strings.TrimSpace(u)
	if !usernameRe.MatchString(u) {
		return ErrInvalidUsername
	}
	return nil
}

func validatePasswordPolicy(pwd string) error {
	if len(pwd) < 8 {
		return ErrWeakPassword
	}
	var upper, lower, digit, special bool
	for _, r := range pwd {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			special = true
		}
	}
	if !upper || !lower || !digit || !special {
		return ErrWeakPassword
	}
	return nil
}

// RegisterFirstAdmin 冷启动注册首个管理员（仅当 NeedsSetup 为 true）；roleSlug 为 super_admin / operator / finance。
// 实现上仅向 public.sys_admin 插入一行，不读写 public.users、user_role_rel，也不调用 C 端用户服务。
func RegisterFirstAdmin(db *gorm.DB, username, password, nickname, roleSlug string) error {
	sysAdminTable := setupSysAdminTable(db)
	rolesTable := setupRolesTable(db)
	ok, err := NeedsSetup(db)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSetupNotNeeded
	}
	if err = validateUsername(username); err != nil {
		return err
	}
	if err = validatePasswordPolicy(password); err != nil {
		return err
	}
	nick := strings.TrimSpace(nickname)
	if nick == "" {
		nick = username
	}

	// 控制台与 C 端 users 命名空间独立：仅校验 sys_admin 用户名
	var adminDup int64
	if err = db.Raw(fmt.Sprintf(`SELECT COUNT(1) FROM %s WHERE deleted_at IS NULL AND LOWER(TRIM(username)) = LOWER(TRIM(?))`, sysAdminTable), username).Scan(&adminDup).Error; err != nil {
		return err
	}
	if adminDup > 0 {
		return ErrUsernameTaken
	}

	rs := strings.TrimSpace(strings.ToLower(roleSlug))
	if rs == "" {
		rs = "super_admin"
	}
	if !isSetupRegisterableSlug(rs) {
		return ErrInvalidSetupRole
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	var roleID int64
	if err = db.Raw(fmt.Sprintf(`SELECT id FROM %s WHERE LOWER(TRIM(slug)) = ? LIMIT 1`, rolesTable), rs).Scan(&roleID).Error; err != nil {
		return err
	}
	if roleID == 0 {
		if rs == "super_admin" {
			return ErrSuperAdminRole
		}
		return ErrSetupRoleNotFound
	}

	return db.Transaction(func(tx *gorm.DB) error {
		var roleName string
		if err := tx.Raw(fmt.Sprintf(`SELECT COALESCE(TRIM(name), '') FROM %s WHERE id = ?`, rolesTable), roleID).Scan(&roleName).Error; err != nil {
			return err
		}
		if strings.TrimSpace(roleName) == "" {
			roleName = rs
		}
		now := time.Now()
		row := models.SysAdmin{
			Username:          username,
			Password:          string(hash),
			NickName:          nick,
			RoleId:            roleID,
			RoleName:          truncateStr(roleName, 50),
			RoleCode:          truncateStr(rs, 50),
			Status:            1,
			PasswordChangedAt: &now,
		}
		// 物理表未建 dept_id/post_id/remark/avatar/salt/update_by（见 admin_register.go 同款说明），显式 Omit
		if err := tx.Table(sysAdminTable).
			Omit("DeptId", "PostId", "Remark", "Avatar", "Salt", "UpdateBy").
			Create(&row).Error; err != nil {
			return err
		}
		// #region agent log
		agentDebugSetupNDJSON("H0", "service/setup.go:RegisterFirstAdmin", "cold-start register persisted to sys_admin only", map[string]any{
			"sysAdminId": row.Id, "touchesUsersTable": false,
		})
		// #endregion
		return nil
	})
}

func truncateStr(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return strings.TrimSpace(s)
	}
	return string(r[:max])
}

// RegisterFirstSuperAdmin 兼容旧调用：固定超级管理员角色
func RegisterFirstSuperAdmin(db *gorm.DB, username, password, nickname string) error {
	return RegisterFirstAdmin(db, username, password, nickname, "super_admin")
}
