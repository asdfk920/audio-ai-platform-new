package handler

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	log "github.com/go-admin-team/go-admin-core/logger"
	"github.com/go-admin-team/go-admin-core/sdk/pkg"
	"go-admin/app/admin/models"
	"go-admin/common/rbac"
	"gorm.io/gorm"
)

// #region agent log
func agentDebugLoginA35f07(hypothesisID, location, message string, data map[string]any) {
	const p = "c:/Users/Lenovo/Desktop/audio-ai-platform/debug-a35f07.log"
	line := map[string]any{
		"sessionId":    "a35f07",
		"runId":        "admin-login",
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

func adminLoginSysAdminTable(tx *gorm.DB) string {
	if tx != nil && tx.Dialector != nil && tx.Dialector.Name() == "postgres" {
		return "public.sys_admin"
	}
	return "sys_admin"
}

func adminLoginSysRoleTable(tx *gorm.DB) string {
	if tx != nil && tx.Dialector != nil && tx.Dialector.Name() == "postgres" {
		return "public.sys_role"
	}
	return "sys_role"
}

// #endregion

type Login struct {
	Username string `form:"UserName" json:"username" binding:"required"`
	Password string `form:"Password" json:"password" binding:"required"`
	Code     string `form:"Code" json:"code" binding:"required"`
	UUID     string `form:"UUID" json:"uuid" binding:"required"`
}

// GetUser 供 /api/v1/login（管理后台默认入口）：仅查 public.sys_admin，与 /api/admin/login 一致
func (u *Login) GetUser(tx *gorm.DB) (user SysUser, role SysRole, err error) {
	return AuthenticateSysAdmin(tx, u.Username, u.Password)
}

// AuthenticateSysAdmin 仅从 public.sys_admin 校验后台账号（/api/admin/login 与 GetUser 共用）
func AuthenticateSysAdmin(tx *gorm.DB, username, password string) (user SysUser, role SysRole, err error) {
	sysAdminTable := adminLoginSysAdminTable(tx)
	sysRoleTable := adminLoginSysRoleTable(tx)
	u := strings.TrimSpace(username)
	// #region agent log
	agentDebugLoginA35f07("H1", "handler/login.go:AuthenticateSysAdmin", "authenticate begin", map[string]any{
		"sysAdminTable": sysAdminTable,
		"sysRoleTable":  sysRoleTable,
		"username":      u,
	})
	// #endregion
	if u == "" {
		err = gorm.ErrRecordNotFound
		return
	}
	var adm models.SysAdmin
	err = tx.Table(sysAdminTable).
		Where("deleted_at IS NULL AND status = 1 AND LOWER(TRIM(username)) = LOWER(TRIM(?))", u).
		First(&adm).Error
	if err != nil {
		// #region agent log
		agentDebugLoginA35f07("H5", "handler/login.go:AuthenticateSysAdmin", "sys_admin First failed", map[string]any{
			"usernameTrimmed": u, "err": err.Error(), "table": sysAdminTable,
		})
		// #endregion
		log.Errorf("sys_admin lookup: %s", err.Error())
		return
	}
	_, err = pkg.CompareHashAndPassword(adm.Password, password)
	if err != nil {
		// #region agent log
		agentDebugLoginA35f07("H6", "handler/login.go:AuthenticateSysAdmin", "bcrypt CompareHashAndPassword failed", map[string]any{
			"adminId": adm.Id, "err": err.Error(),
		})
		// #endregion
		log.Errorf("sys_admin bcrypt: %s", err.Error())
		return
	}
	err = tx.Table(sysRoleTable).Where("role_id = ?", adm.RoleId).First(&role).Error
	if err != nil {
		// #region agent log
		agentDebugLoginA35f07("H7", "handler/login.go:AuthenticateSysAdmin", "sys_role First failed", map[string]any{
			"adminId": adm.Id, "roleId": adm.RoleId, "err": err.Error(), "table": sysRoleTable,
		})
		// #endregion
		log.Errorf("get role error, %s", err.Error())
		return
	}
	effKey := strings.TrimSpace(role.RoleKey)
	roleKeyEmpty := effKey == ""
	if effKey == "" {
		effKey = strings.TrimSpace(adm.RoleCode)
	}
	// #region agent log
	agentDebugLoginA35f07("H8", "handler/login.go:AuthenticateSysAdmin", "role keys for rbac", map[string]any{
		"roleId":            adm.RoleId,
		"roleKeyFromView":   strings.TrimSpace(role.RoleKey),
		"roleCodeFromAdmin": strings.TrimSpace(adm.RoleCode),
		"effKeyForRbac":     effKey,
		"roleKeyWasEmpty":   roleKeyEmpty,
		"rbacOk":            rbac.IsConsoleAdminRole(effKey),
	})
	// #endregion
	if !rbac.IsConsoleAdminRole(effKey) {
		log.Warnf("login rejected: admin %q has non-console role view=%q eff=%q", u, role.RoleKey, effKey)
		err = gorm.ErrRecordNotFound
		return
	}
	// #region agent log
	agentDebugLoginA35f07("H4", "handler/login.go:AuthenticateSysAdmin", "authenticate success", map[string]any{
		"adminId": adm.Id,
		"roleId":  adm.RoleId,
	})
	// #endregion
	user = SysUser{
		UserId:   int(adm.Id),
		Username: adm.Username,
		Password: adm.Password,
		NickName: adm.NickName,
		Phone:    adm.Mobile,
		Email:    adm.Email,
		RoleId:   int(adm.RoleId),
		Status:   "2",
		Avatar:   adm.Avatar,
	}
	return user, role, nil
}
