package rbac

import "strings"

// ConsoleAdminSlugs 允许登录后台管理端的角色标识（与 public.roles.slug / sys_role.role_key 一致）
var ConsoleAdminSlugs = []string{"super_admin", "admin", "operator", "finance"}

var consoleAdminSet map[string]struct{}

func init() {
	consoleAdminSet = make(map[string]struct{}, len(ConsoleAdminSlugs))
	for _, s := range ConsoleAdminSlugs {
		consoleAdminSet[strings.ToLower(s)] = struct{}{}
	}
}

// casbinBypassSet 与 go-admin 历史行为一致：仅 `admin` 曾跳过 Casbin；冷启动超级管理员使用 slug `super_admin`，须同等对待否则 casbin_rule 无对应 p 策略时会 403。
var casbinBypassSet = map[string]struct{}{
	"admin":       {},
	"super_admin": {},
}

// IsCasbinBypassRole 为 true 时不走 Casbin Enforce（全接口放行）。operator/finance 仍走 Casbin 或 CasbinExclude。
func IsCasbinBypassRole(roleKey string) bool {
	k := strings.ToLower(strings.TrimSpace(roleKey))
	if k == "" {
		return false
	}
	_, ok := casbinBypassSet[k]
	return ok
}

// IsConsoleAdminRole 返回 true 表示允许使用 /api/v1/login 进入后台
func IsConsoleAdminRole(roleKey string) bool {
	k := strings.ToLower(strings.TrimSpace(roleKey))
	if k == "" {
		return false
	}
	_, ok := consoleAdminSet[k]
	return ok
}
