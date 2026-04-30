package service

import (
	"encoding/json"
	"errors"
	"strings"

	"gorm.io/gorm"
)

type PlatformRbac struct{}

type roleRow struct {
	ID          int64           `gorm:"column:id"`
	Name        string          `gorm:"column:name"`
	Slug        string          `gorm:"column:slug"`
	Description string          `gorm:"column:description"`
	Permissions json.RawMessage `gorm:"column:permissions;type:jsonb"`
}

func (PlatformRbac) ListRoles(db *gorm.DB) ([]map[string]interface{}, error) {
	var rows []roleRow
	err := db.Table("roles").
		Select("id", "name", "slug", "description", "permissions").
		Where("COALESCE(TRIM(slug), '') <> ''").
		Order("id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, r := range rows {
		rk := strings.TrimSpace(r.Slug)
		if rk == "" {
			rk = strings.TrimSpace(r.Name)
		}
		var perm map[string]interface{}
		if len(r.Permissions) > 0 {
			_ = json.Unmarshal(r.Permissions, &perm)
		}
		if perm == nil {
			perm = map[string]interface{}{}
		}
		out = append(out, map[string]interface{}{
			"roleKey":     rk,
			"roleName":    r.Name,
			"description": r.Description,
			"permissions": perm,
		})
	}
	return out, nil
}

func defaultModulesMeta() []map[string]interface{} {
	return []map[string]interface{}{
		{"key": "user_mgmt", "title": "用户管理"},
		{"key": "member_mgmt", "title": "会员管理"},
		{"key": "device_mgmt", "title": "设备管理"},
		{"key": "content_mgmt", "title": "内容管理"},
		{"key": "ota", "title": "OTA"},
		{"key": "stats", "title": "统计"},
		{"key": "audit_log", "title": "审计"},
		{"key": "sys_config", "title": "系统配置"},
	}
}

func (PlatformRbac) ModulesMeta() []map[string]interface{} {
	return defaultModulesMeta()
}

func (PlatformRbac) MatrixForRole(db *gorm.DB, roleKey string) (map[string]interface{}, error) {
	rk := strings.TrimSpace(roleKey)
	if rk == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var row roleRow
	err := db.Table("roles").
		Select("name", "slug", "description", "permissions").
		Where("LOWER(TRIM(slug)) = LOWER(TRIM(?))", rk).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	var perm map[string]interface{}
	if len(row.Permissions) > 0 {
		_ = json.Unmarshal(row.Permissions, &perm)
	}
	if perm == nil {
		perm = map[string]interface{}{}
	}
	mods := modulesFromPermissions(perm)
	return map[string]interface{}{
		"roleKey": strings.TrimSpace(row.Slug),
		"role": map[string]interface{}{
			"name":        row.Name,
			"description": row.Description,
			"modules":     mods,
		},
		"modules": mods,
	}, nil
}

func modulesFromPermissions(perm map[string]interface{}) map[string]interface{} {
	raw, ok := perm["modules"]
	if !ok || raw == nil {
		return map[string]interface{}{}
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	return m
}

// ErrRbacForbidden 当前角色无权修改矩阵
var ErrRbacForbidden = errors.New("role not allowed to edit rbac")

func (PlatformRbac) UpdateRoleModules(db *gorm.DB, editorRoleKey string, targetRoleKey string, modules map[string]string) error {
	ed := strings.ToLower(strings.TrimSpace(editorRoleKey))
	if ed != "super_admin" && ed != "admin" {
		return ErrRbacForbidden
	}
	tk := strings.TrimSpace(targetRoleKey)
	if tk == "" {
		return errors.New("empty role")
	}
	if strings.EqualFold(tk, "super_admin") && ed != "super_admin" {
		return ErrRbacForbidden
	}

	var row roleRow
	err := db.Table("roles").Select("id", "permissions").Where("LOWER(TRIM(slug)) = LOWER(TRIM(?))", tk).First(&row).Error
	if err != nil {
		return err
	}
	var perm map[string]interface{}
	if len(row.Permissions) > 0 {
		_ = json.Unmarshal(row.Permissions, &perm)
	}
	if perm == nil {
		perm = map[string]interface{}{}
	}
	modMap := make(map[string]interface{}, len(modules))
	for k, v := range modules {
		modMap[k] = v
	}
	perm["modules"] = modMap
	b, err := json.Marshal(perm)
	if err != nil {
		return err
	}
	return db.Table("roles").Where("id = ?", row.ID).Update("permissions", string(b)).Error
}

// CanAccessModule 返回 true 表示该角色对某模块不是 none
func CanAccessModule(db *gorm.DB, roleKey string, moduleKey string) bool {
	rk := strings.TrimSpace(roleKey)
	if rk == "" {
		return false
	}
	var raw json.RawMessage
	err := db.Table("roles").Select("permissions").Where("LOWER(TRIM(slug)) = LOWER(TRIM(?))", rk).Limit(1).Scan(&raw).Error
	if err != nil || len(raw) == 0 {
		return false
	}
	var perm map[string]interface{}
	if err := json.Unmarshal(raw, &perm); err != nil {
		return false
	}
	mods, ok := perm["modules"].(map[string]interface{})
	if !ok {
		return false
	}
	v, ok := mods[moduleKey]
	if !ok {
		return false
	}
	s, _ := v.(string)
	return strings.ToLower(strings.TrimSpace(s)) != "none"
}

