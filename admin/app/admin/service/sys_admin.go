package service

// SysAdmin 控制台管理员模块的服务层。
//
// 数据源：public.sys_admin（与 C 端 public.users 完全解耦）。早期实现曾经
// 通过 public.sys_user 视图间接访问本表，但视图在 INSERT/UPDATE 场景下语义
// 不稳定，且难以承载 dept_id/ must_change_password 等扩展字段，因此本轮
// 将所有 CRUD 迁移到直接操作 sys_admin。
//
// 本服务在 CRUD 之外新增：
//   - AdminBatchDelete       批量软删除（逐条校验，原因可追溯）
//   - AdminResetPassword     超管重置他人密码，默认要求对方下次登录再改
//   - AdminChangePasswordSelf 已登录管理员自助改密（配合首次登录强制改密）
//   - AdminSetSecurity       设置 IP 白名单 + 登录时间窗
//   - AdminSetForceChange    单独开关 must_change_password
//   - EnforceLoginRestrictions 登录链路调用，拒绝不合法 IP / 时间窗
//
// 对应 DB 迁移：scripts/db/migrations/082_sys_admin_security.sql

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/go-admin-team/go-admin-core/sdk/service"

	"go-admin/app/admin/models"
	"go-admin/app/admin/service/dto"
	common "go-admin/common/models"
	"go-admin/common/rbac"
)

// ===== 哨兵错误 =====

var (
	ErrSysAdminNotFound      = errors.New("sys admin not found")
	ErrSysAdminBuiltin       = errors.New("builtin super admin is not modifiable")
	ErrSysAdminSelfProtect   = errors.New("cannot operate self")
	ErrSysAdminPwdBadOld     = errors.New("old password mismatch")
	ErrSysAdminPwdPolicy     = errors.New("password policy violated")
	ErrSysAdminIPRejected    = errors.New("login ip not allowed")
	ErrSysAdminTimeRejected  = errors.New("login time window rejected")
	ErrSysAdminMustChangePwd = errors.New("password must be changed before continuing")
)

type SysAdmin struct {
	service.Service
}

// #region agent log
func agentDebugNDJSONA35f07(runID, hypothesisId, location, message string, data map[string]any) {
	type payload struct {
		SessionID    string         `json:"sessionId"`
		RunID        string         `json:"runId"`
		HypothesisID string         `json:"hypothesisId"`
		Location     string         `json:"location"`
		Message      string         `json:"message"`
		Data         map[string]any `json:"data,omitempty"`
		Timestamp    int64          `json:"timestamp"`
	}
	root := os.Getenv("WORKSPACE_ROOT")
	if root == "" {
		// 兼容 Cursor 工作区根目录；失败则退回相对路径
		root = `C:\Users\Lenovo\Desktop\audio-ai-platform`
	}
	p := filepath.Join(root, "debug-a35f07.log")
	b, err := json.Marshal(payload{
		SessionID:    "a35f07",
		RunID:        runID,
		HypothesisID: hypothesisId,
		Location:     location,
		Message:      message,
		Data:         data,
		Timestamp:    time.Now().UnixMilli(),
	})
	if err != nil {
		return
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
	_ = f.Close()
}

// #endregion

// ===== 状态/角色/部门辅助 =====

// go-admin 前端状态约定："2" 正常 / "1" 禁用；sys_admin.status：1 启用 / 0 禁用
func goAdminStatusToDB(s string) int16 {
	if s == "2" {
		return 1
	}
	return 0
}

func dbStatusToGoAdmin(v int16) string {
	if v == 1 {
		return "2"
	}
	return "1"
}

func adminStatusText(s string) string {
	if s == "2" {
		return "正常"
	}
	return "禁用"
}

func (e *SysAdmin) loadRole(roleID int64) (models.SysRole, error) {
	var role models.SysRole
	err := e.Orm.Where("role_id = ?", roleID).First(&role).Error
	return role, err
}

func roleKey(r models.SysRole) string {
	if k := strings.TrimSpace(r.RoleKey); k != "" {
		return k
	}
	return strings.TrimSpace(r.RoleName)
}

// deptNameByID 安全读取部门名；0 或查无返回空串。
func (e *SysAdmin) deptNameByID(deptID int) string {
	if deptID <= 0 {
		return ""
	}
	var d models.SysDept
	if err := e.Orm.Select("dept_id, dept_name").First(&d, deptID).Error; err != nil {
		return ""
	}
	return d.DeptName
}

// desensitizePhone 手机号脱敏：xxx****xxxx
func desensitizePhone(phone string) string {
	if len(phone) != 11 {
		return phone
	}
	return phone[:3] + "****" + phone[7:]
}

// ===== 列表 / 详情 =====

// GetAdminList 分页查询 sys_admin；Count 与 Find 共用同一份 where。
func (e *SysAdmin) GetAdminList(c *dto.SysAdminGetPageReq, list *[]dto.SysAdminListItem, count *int64) error {
	db := e.Orm.Model(&models.SysAdmin{}).Where("deleted_at IS NULL")

	if kw := strings.TrimSpace(c.Keyword); kw != "" {
		like := "%" + kw + "%"
		db = db.Where("username LIKE ? OR nick_name LIKE ? OR real_name LIKE ?", like, like, like)
	}
	if c.RoleId > 0 {
		db = db.Where("role_id = ?", int64(c.RoleId))
	}
	if c.Status != "" {
		db = db.Where("status = ?", goAdminStatusToDB(c.Status))
	}
	if c.LastLoginFrom != "" {
		db = db.Where("last_login_at >= ?", c.LastLoginFrom)
	}
	if c.LastLoginTo != "" {
		db = db.Where("last_login_at <= ?", c.LastLoginTo+" 23:59:59")
	}

	if err := db.Count(count).Error; err != nil {
		e.Log.Errorf("sys_admin count error: %s", err)
		return err
	}

	sortBy := c.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := strings.ToLower(c.SortOrder)
	if sortOrder != "asc" {
		sortOrder = "desc"
	}
	db = db.Order(sortBy + " " + sortOrder)

	page := c.GetPageIndex()
	pageSize := c.GetPageSize()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	db = db.Offset((page - 1) * pageSize).Limit(pageSize)

	var admins []models.SysAdmin
	if err := db.Find(&admins).Error; err != nil {
		e.Log.Errorf("sys_admin find error: %s", err)
		return err
	}

	*list = make([]dto.SysAdminListItem, 0, len(admins))
	for _, adm := range admins {
		st := dbStatusToGoAdmin(adm.Status)
		item := dto.SysAdminListItem{
			AdminId:            int(adm.Id),
			UserId:             int(adm.Id),
			Username:           adm.Username,
			Nickname:           adm.NickName,
			RealName:           adm.RealName,
			Email:              adm.Email,
			PhoneRaw:           adm.Mobile,
			Phone:              desensitizePhone(adm.Mobile),
			Avatar:             adm.Avatar,
			DeptId:             adm.DeptId,
			DeptName:           e.deptNameByID(adm.DeptId),
			Status:             st,
			StatusText:         adminStatusText(st),
			LoginCount:         adm.LoginCount,
			MustChangePassword: adm.MustChangePassword,
			CreatedAt:          adm.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:          adm.UpdatedAt.Format("2006-01-02 15:04:05"),
			IsSuper:            rbac.IsCasbinBypassRole(strings.TrimSpace(adm.RoleCode)),
		}
		if adm.LastLoginAt != nil && !adm.LastLoginAt.IsZero() {
			item.LastLoginTime = adm.LastLoginAt.Format("2006-01-02 15:04:05")
		}
		item.LastLoginIp = adm.LastLoginIP

		if adm.CreatedBy != nil && *adm.CreatedBy > 0 {
			var creator models.SysAdmin
			if err := e.Orm.Select("id, username").First(&creator, *adm.CreatedBy).Error; err == nil {
				item.CreatedBy = creator.Username
			}
		}
		if role, err := e.loadRole(adm.RoleId); err == nil {
			item.RoleList = append(item.RoleList, dto.SysAdminRoleItem{
				RoleId:   role.RoleId,
				RoleName: role.RoleName,
				RoleCode: roleKey(role),
			})
		}

		*list = append(*list, item)
	}
	return nil
}

// GetAdminDetail 取详情；包含 dept / 安全策略 / 强制改密标志。
func (e *SysAdmin) GetAdminDetail(req *dto.SysAdminDetailReq) (*dto.SysAdminDetail, error) {
	var adm models.SysAdmin
	if err := e.Orm.First(&adm, req.Id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSysAdminNotFound
		}
		return nil, err
	}
	if adm.DeletedAt.Valid {
		return nil, ErrSysAdminNotFound
	}

	st := dbStatusToGoAdmin(adm.Status)
	detail := &dto.SysAdminDetail{
		AdminId:            int(adm.Id),
		UserId:             int(adm.Id),
		Username:           adm.Username,
		Nickname:           adm.NickName,
		RealName:           adm.RealName,
		Email:              adm.Email,
		Phone:              adm.Mobile,
		Avatar:             adm.Avatar,
		DeptId:             adm.DeptId,
		DeptName:           e.deptNameByID(adm.DeptId),
		Status:             st,
		StatusText:         adminStatusText(st),
		LoginCount:         adm.LoginCount,
		Remark:             adm.Remark,
		AllowedIps:         adm.AllowedIPs,
		MustChangePassword: adm.MustChangePassword,
		IsSuper:            rbac.IsCasbinBypassRole(strings.TrimSpace(adm.RoleCode)),
	}
	if adm.AllowedLoginStart != nil {
		detail.AllowedLoginStart = *adm.AllowedLoginStart
	}
	if adm.AllowedLoginEnd != nil {
		detail.AllowedLoginEnd = *adm.AllowedLoginEnd
	}
	if adm.LastLoginAt != nil && !adm.LastLoginAt.IsZero() {
		detail.LastLoginTime = adm.LastLoginAt.Format("2006-01-02 15:04:05")
	}
	detail.LastLoginIp = adm.LastLoginIP
	if adm.LastPasswordChangedAt != nil && !adm.LastPasswordChangedAt.IsZero() {
		detail.LastPasswordChangedAt = adm.LastPasswordChangedAt.Format("2006-01-02 15:04:05")
	}
	if adm.CreatedBy != nil && *adm.CreatedBy > 0 {
		var creator models.SysAdmin
		if err := e.Orm.Select("id, username").First(&creator, *adm.CreatedBy).Error; err == nil {
			detail.CreatedBy = creator.Username
		}
	}
	if adm.UpdateBy != nil && *adm.UpdateBy > 0 {
		var updater models.SysAdmin
		if err := e.Orm.Select("id, username").First(&updater, *adm.UpdateBy).Error; err == nil {
			detail.UpdatedBy = updater.Username
		}
	}
	detail.CreatedAt = adm.CreatedAt.Format("2006-01-02 15:04:05")
	detail.UpdatedAt = adm.UpdatedAt.Format("2006-01-02 15:04:05")

	if role, err := e.loadRole(adm.RoleId); err == nil {
		detail.RoleList = append(detail.RoleList, dto.SysAdminRoleItem{
			RoleId:   role.RoleId,
			RoleName: role.RoleName,
			RoleCode: roleKey(role),
		})
		detail.RoleIds = append(detail.RoleIds, role.RoleId)
	}
	return detail, nil
}

// ===== 创建 =====

// AdminCreate 新增管理员；默认创建后要求首次登录改密，可通过 req.MustChangePassword 显式关闭。
func (e *SysAdmin) AdminCreate(req *dto.SysAdminCreateReq) (*dto.SysAdminDetail, error) {
	// #region agent log
	agentDebugNDJSONA35f07("admin-create", "H1", "service/sys_admin.go:AdminCreate", "enter", map[string]any{
		"usernameLen": len(strings.TrimSpace(req.Username)),
		"nicknameLen": len(strings.TrimSpace(req.Nickname)),
		"roleIdsLen":  len(req.RoleIds),
		"deptId":      req.DeptId,
		"hasEmail":    strings.TrimSpace(req.Email) != "",
		"hasPhone":    strings.TrimSpace(req.Phone) != "",
	})
	// #endregion

	if err := sysAdminUsernamePolicy(req.Username); err != nil {
		return nil, err
	}
	if err := sysAdminPasswordPolicy(req.Password); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Nickname) == "" {
		return nil, errors.New("昵称不能为空")
	}
	if len(req.RoleIds) == 0 {
		return nil, errors.New("至少选择一个角色")
	}
	if err := validateOptionalEmail(req.Email); err != nil {
		return nil, err
	}
	if err := validateOptionalPhone(req.Phone); err != nil {
		return nil, err
	}

	// 唯一性校验
	var cnt int64
	if err := e.Orm.Model(&models.SysAdmin{}).
		Where("deleted_at IS NULL AND LOWER(TRIM(username)) = LOWER(TRIM(?))", req.Username).
		Count(&cnt).Error; err != nil {
		return nil, errors.New("用户名校验失败")
	}
	if cnt > 0 {
		return nil, errors.New("该用户名已被使用请更换")
	}
	if req.Phone != "" {
		if err := e.Orm.Model(&models.SysAdmin{}).
			Where("deleted_at IS NULL AND mobile = ?", req.Phone).
			Count(&cnt).Error; err != nil {
			return nil, errors.New("手机号校验失败")
		}
		if cnt > 0 {
			return nil, errors.New("该手机号已被其他账户使用")
		}
	}
	if req.Email != "" {
		if err := e.Orm.Model(&models.SysAdmin{}).
			Where("deleted_at IS NULL AND LOWER(email) = LOWER(?)", req.Email).
			Count(&cnt).Error; err != nil {
			return nil, errors.New("邮箱校验失败")
		}
		if cnt > 0 {
			return nil, errors.New("该邮箱已被其他账户使用")
		}
	}

	// Licence 占位：控制台账号最多 100 个
	var adminCount int64
	if err := e.Orm.Model(&models.SysAdmin{}).Where("deleted_at IS NULL").Count(&adminCount).Error; err != nil {
		return nil, errors.New("管理员数量校验失败")
	}
	if adminCount >= 100 {
		return nil, errors.New("管理员数量已达授权上限请联系商务续费")
	}

	// 角色有效性校验
	roles := make([]models.SysRole, 0, len(req.RoleIds))
	if err := e.Orm.Where("role_id IN ?", req.RoleIds).Find(&roles).Error; err != nil {
		return nil, errors.New("角色信息校验失败")
	}
	if len(roles) != len(req.RoleIds) {
		return nil, errors.New("所选角色不存在或已被禁用")
	}
	for _, r := range roles {
		if r.Status == "1" {
			return nil, errors.New("所选角色不存在或已被禁用")
		}
	}
	primary := roles[0]
	if len(req.RoleIds) > 1 {
		e.Log.Warnf("控制台管理员仅落库单一角色，已使用 role_id=%d，忽略其余", primary.RoleId)
	}

	// #region agent log
	agentDebugNDJSONA35f07("admin-create", "H2", "service/sys_admin.go:AdminCreate", "primary_role", map[string]any{
		"roleId":   primary.RoleId,
		"roleKey":  roleKey(primary),
		"roleName": primary.RoleName,
		"status":   primary.Status,
	})
	// #endregion

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("密码加密失败")
	}

	st := req.Status
	if st == "" {
		st = "2"
	}
	mustChange := true
	if req.MustChangePassword != nil {
		mustChange = *req.MustChangePassword
	}

	adm := models.SysAdmin{
		Username:           req.Username,
		Password:           string(hashed),
		NickName:           req.Nickname,
		RealName:           req.RealName,
		Email:              req.Email,
		Mobile:             req.Phone,
		Avatar:             req.Avatar,
		DeptId:             req.DeptId,
		Remark:             req.Remark,
		RoleId:             int64(primary.RoleId),
		RoleName:           primary.RoleName,
		RoleCode:           roleKey(primary),
		Status:             goAdminStatusToDB(st),
		MustChangePassword: mustChange,
	}
	now := time.Now()
	adm.LastPasswordChangedAt = &now
	adm.PasswordChangedAt = &now
	adm.SetCreateBy(req.CreateBy)

	if err := e.Orm.Create(&adm).Error; err != nil {
		e.Log.Errorf("insert sys_admin failed: %v", err)
		// #region agent log
		agentDebugNDJSONA35f07("admin-create", "H3", "service/sys_admin.go:AdminCreate", "insert_failed", map[string]any{
			"errType": fmt.Sprintf("%T", err),
			"errMsg":  err.Error(),
		})
		// #endregion
		return nil, errors.New("创建管理员失败")
	}
	// #region agent log
	agentDebugNDJSONA35f07("admin-create", "H4", "service/sys_admin.go:AdminCreate", "insert_ok", map[string]any{
		"id": adm.Id,
	})
	// #endregion
	return e.GetAdminDetail(&dto.SysAdminDetailReq{Id: int(adm.Id)})
}

// ===== 更新 =====

// AdminUpdate 更新非敏感信息；不通过本接口改密。
func (e *SysAdmin) AdminUpdate(req *dto.SysAdminUpdateReq) (*dto.SysAdminDetail, error) {
	if req.UserId <= 0 {
		return nil, errors.New("用户 ID 无效")
	}
	if strings.TrimSpace(req.Nickname) == "" {
		return nil, errors.New("昵称不能为空")
	}
	if len(req.RoleIds) == 0 {
		return nil, errors.New("至少选择一个角色")
	}
	if err := validateOptionalEmail(req.Email); err != nil {
		return nil, err
	}
	if err := validateOptionalPhone(req.Phone); err != nil {
		return nil, err
	}

	var adm models.SysAdmin
	if err := e.Orm.First(&adm, req.UserId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSysAdminNotFound
		}
		return nil, errors.New("管理员不存在")
	}
	if adm.DeletedAt.Valid {
		return nil, ErrSysAdminNotFound
	}
	targetBypass := rbac.IsCasbinBypassRole(strings.TrimSpace(adm.RoleCode))
	selfUpdate := req.UpdateBy > 0 && req.UpdateBy == int(adm.Id)
	actorBypass := req.ActorIsCasbinBypass
	// #region agent log
	agentDebugNDJSONA35f07("admin-update", "H6", "service/sys_admin.go:AdminUpdate", "builtin_gate", map[string]any{
		"targetId":      int(adm.Id),
		"targetBypass":  targetBypass,
		"selfUpdate":    selfUpdate,
		"actorBypass":   actorBypass,
		"updateBy":      req.UpdateBy,
		"reqUserId":     req.UserId,
		"reqStatus":     strings.TrimSpace(req.Status),
		"roleIdsLenIn":  len(req.RoleIds),
		"lockedRoleId":  int(adm.RoleId),
	})
	// #endregion
	if targetBypass {
		if !selfUpdate && !actorBypass {
			return nil, ErrSysAdminBuiltin
		}
		// 内置 Casbin 放行角色：仅当操作者不是超管时，禁止改角色；超管可修改其他超管的角色。
		if !actorBypass {
			if adm.RoleId <= 0 {
				return nil, errors.New("内置管理员角色配置异常")
			}
			req.RoleIds = []int{int(adm.RoleId)}
		}
		// 禁用仅允许命中"不能禁用自己的账户"分支。
		if strings.TrimSpace(req.Status) == "1" {
			if selfUpdate {
				return nil, errors.New("不能禁用自己的账户")
			}
			return nil, ErrSysAdminBuiltin
		}
	}

	if req.Phone != "" && req.Phone != adm.Mobile {
		var cnt int64
		if err := e.Orm.Model(&models.SysAdmin{}).
			Where("deleted_at IS NULL AND mobile = ? AND id <> ?", req.Phone, adm.Id).
			Count(&cnt).Error; err != nil {
			return nil, errors.New("手机号校验失败")
		}
		if cnt > 0 {
			return nil, errors.New("该手机号已被其他管理员使用")
		}
	}
	if req.Email != "" && !strings.EqualFold(req.Email, adm.Email) {
		var cnt int64
		if err := e.Orm.Model(&models.SysAdmin{}).
			Where("deleted_at IS NULL AND LOWER(email) = LOWER(?) AND id <> ?", req.Email, adm.Id).
			Count(&cnt).Error; err != nil {
			return nil, errors.New("邮箱校验失败")
		}
		if cnt > 0 {
			return nil, errors.New("该邮箱已被其他管理员使用")
		}
	}

	roles := make([]models.SysRole, 0, len(req.RoleIds))
	if err := e.Orm.Where("role_id IN ?", req.RoleIds).Find(&roles).Error; err != nil {
		return nil, errors.New("角色信息校验失败")
	}
	if len(roles) != len(req.RoleIds) {
		return nil, errors.New("所选角色不存在或已被禁用")
	}
	for _, r := range roles {
		if r.Status == "1" {
			return nil, errors.New("所选角色不存在或已被禁用")
		}
	}
	primary := roles[0]

	if req.UpdateBy == req.UserId && req.Status == "1" {
		return nil, errors.New("不能禁用自己的账户")
	}

	adm.NickName = req.Nickname
	adm.RealName = req.RealName
	adm.Email = req.Email
	adm.Mobile = req.Phone
	adm.Avatar = req.Avatar
	adm.DeptId = req.DeptId
	adm.Remark = req.Remark
	adm.RoleId = int64(primary.RoleId)
	adm.RoleName = primary.RoleName
	adm.RoleCode = roleKey(primary)
	if req.Status != "" {
		adm.Status = goAdminStatusToDB(req.Status)
	}
	adm.SetUpdateBy(req.UpdateBy)

	if err := e.Orm.Save(&adm).Error; err != nil {
		e.Log.Errorf("update sys_admin failed: %v", err)
		return nil, errors.New("更新管理员失败")
	}
	return e.GetAdminDetail(&dto.SysAdminDetailReq{Id: int(adm.Id)})
}

// ===== 删除 =====

func (e *SysAdmin) AdminDelete(req *dto.SysAdminDeleteReq) (*dto.SysAdminDeleteResponse, error) {
	if req.UserId <= 0 {
		return nil, errors.New("用户 ID 无效")
	}
	if req.Confirm != nil && !*req.Confirm {
		return nil, errors.New("请确认删除操作")
	}

	var adm models.SysAdmin
	if err := e.Orm.First(&adm, req.UserId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSysAdminNotFound
		}
		return nil, errors.New("管理员记录不存在")
	}
	if adm.DeletedAt.Valid {
		return nil, errors.New("管理员已被删除")
	}
	if rbac.IsCasbinBypassRole(strings.TrimSpace(adm.RoleCode)) {
		return nil, ErrSysAdminBuiltin
	}
	if req.DeleteBy == req.UserId {
		return nil, ErrSysAdminSelfProtect
	}

	if err := e.Orm.Delete(&adm).Error; err != nil {
		return nil, errors.New("删除管理员失败")
	}
	deletedAt := time.Now().Format("2006-01-02 15:04:05")
	if adm.DeletedAt.Valid && !adm.DeletedAt.Time.IsZero() {
		deletedAt = adm.DeletedAt.Time.Format("2006-01-02 15:04:05")
	}
	return &dto.SysAdminDeleteResponse{
		Success:   true,
		AdminId:   int(adm.Id),
		Username:  adm.Username,
		DeletedAt: deletedAt,
		Message:   "删除成功",
	}, nil
}

// AdminBatchDelete 批量软删除。复用 AdminDelete 以保持一致的合规校验；
// 每条失败单独记录到 Fails 列表，整体接口仍返回 200，上层 HTTP handler 不报 500。
func (e *SysAdmin) AdminBatchDelete(req *dto.SysAdminBatchDeleteReq) (*dto.SysAdminBatchDeleteResp, error) {
	resp := &dto.SysAdminBatchDeleteResp{Total: len(req.UserIds)}
	for _, id := range req.UserIds {
		confirm := true
		_, err := e.AdminDelete(&dto.SysAdminDeleteReq{
			UserId:  id,
			Confirm: &confirm,
			Reason:  req.Reason,
			ControlBy: common.ControlBy{
				CreateBy: req.CreateBy,
				UpdateBy: req.UpdateBy,
				DeleteBy: req.UpdateBy,
			},
		})
		if err != nil {
			resp.Failed++
			resp.Fails = append(resp.Fails, fmt.Sprintf("%d:%s", id, err.Error()))
			continue
		}
		resp.Success++
	}
	return resp, nil
}

// ===== 状态 =====

func (e *SysAdmin) AdminStatus(req *dto.SysAdminStatusReq) (*dto.SysAdminDetail, error) {
	if req.UserId <= 0 {
		return nil, errors.New("用户 ID 无效")
	}
	if req.Status != "1" && req.Status != "2" {
		return nil, errors.New("状态值无效")
	}
	if req.UpdateBy == req.UserId && req.Status == "1" {
		return nil, errors.New("不能禁用自己的账户")
	}

	var adm models.SysAdmin
	if err := e.Orm.First(&adm, req.UserId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSysAdminNotFound
		}
		return nil, errors.New("管理员不存在")
	}
	if adm.DeletedAt.Valid {
		return nil, ErrSysAdminNotFound
	}
	if rbac.IsCasbinBypassRole(strings.TrimSpace(adm.RoleCode)) && req.Status == "1" {
		return nil, ErrSysAdminBuiltin
	}

	adm.Status = goAdminStatusToDB(req.Status)
	adm.SetUpdateBy(req.UpdateBy)
	if err := e.Orm.Save(&adm).Error; err != nil {
		return nil, errors.New("更新状态失败")
	}
	return e.GetAdminDetail(&dto.SysAdminDetailReq{Id: int(adm.Id)})
}

// ===== 密码管理 =====

// AdminResetPassword 超管重置他人密码；默认置 must_change_password = true。
func (e *SysAdmin) AdminResetPassword(req *dto.SysAdminResetPasswordReq) error {
	if req.UserId <= 0 {
		return ErrSysAdminNotFound
	}
	if err := sysAdminPasswordPolicy(req.NewPassword); err != nil {
		return err
	}
	var adm models.SysAdmin
	if err := e.Orm.First(&adm, req.UserId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSysAdminNotFound
		}
		return err
	}
	if adm.DeletedAt.Valid {
		return ErrSysAdminNotFound
	}
	// 允许超管给自己重置（走 change-password 语义）；禁止其他操作者重置内置超管。
	if rbac.IsCasbinBypassRole(strings.TrimSpace(adm.RoleCode)) && int(adm.Id) != req.UpdateBy {
		return ErrSysAdminBuiltin
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密码加密失败")
	}
	now := time.Now()
	must := true
	if req.RequireChangeOnLogin != nil {
		must = *req.RequireChangeOnLogin
	}
	// 超管重置自己密码时，不再要求自己下次登录改密
	if int(adm.Id) == req.UpdateBy {
		must = false
	}

	updates := map[string]interface{}{
		"password":                 string(hashed),
		"must_change_password":     must,
		"last_password_changed_at": now,
		"password_changed_at":      now,
		"updated_at":               now,
	}
	if req.UpdateBy > 0 {
		updates["update_by"] = int64(req.UpdateBy)
	}
	if err := e.Orm.Model(&models.SysAdmin{}).Where("id = ?", adm.Id).Updates(updates).Error; err != nil {
		e.Log.Errorf("reset password failed: %v", err)
		return errors.New("重置密码失败")
	}
	return nil
}

// AdminChangePasswordSelf 当前管理员自助改密；必须校验旧密码；成功后清 must_change_password。
func (e *SysAdmin) AdminChangePasswordSelf(adminID int, oldPwd, newPwd string) error {
	if adminID <= 0 {
		return ErrSysAdminNotFound
	}
	if err := sysAdminPasswordPolicy(newPwd); err != nil {
		return err
	}
	if oldPwd == newPwd {
		return errors.New("新密码不能与旧密码相同")
	}
	var adm models.SysAdmin
	if err := e.Orm.First(&adm, adminID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSysAdminNotFound
		}
		return err
	}
	if adm.DeletedAt.Valid {
		return ErrSysAdminNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(adm.Password), []byte(oldPwd)); err != nil {
		return ErrSysAdminPwdBadOld
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密码加密失败")
	}
	now := time.Now()
	if err := e.Orm.Model(&models.SysAdmin{}).Where("id = ?", adm.Id).Updates(map[string]interface{}{
		"password":                 string(hashed),
		"must_change_password":     false,
		"last_password_changed_at": now,
		"password_changed_at":      now,
		"updated_at":               now,
		"update_by":                int64(adminID),
	}).Error; err != nil {
		return errors.New("修改密码失败")
	}
	return nil
}

// ===== 安全策略 =====

// AdminSetSecurity 设置 IP 白名单 + 登录时间窗；空串/空值表示「不限」。
func (e *SysAdmin) AdminSetSecurity(req *dto.SysAdminSecurityReq) error {
	if req.UserId <= 0 {
		return ErrSysAdminNotFound
	}
	ips := normalizeAllowedIPs(req.AllowedIps)
	start, end, err := validateLoginWindow(req.AllowedLoginStart, req.AllowedLoginEnd)
	if err != nil {
		return err
	}
	var adm models.SysAdmin
	if err := e.Orm.First(&adm, req.UserId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSysAdminNotFound
		}
		return err
	}
	if adm.DeletedAt.Valid {
		return ErrSysAdminNotFound
	}
	updates := map[string]interface{}{
		"allowed_ips":         ips,
		"allowed_login_start": start,
		"allowed_login_end":   end,
		"updated_at":          time.Now(),
	}
	if req.UpdateBy > 0 {
		updates["update_by"] = int64(req.UpdateBy)
	}
	return e.Orm.Model(&models.SysAdmin{}).Where("id = ?", adm.Id).Updates(updates).Error
}

// AdminSetForceChange 单独开关某管理员下次登录是否必须改密。
func (e *SysAdmin) AdminSetForceChange(req *dto.SysAdminForceChangeReq) error {
	if req.UserId <= 0 {
		return ErrSysAdminNotFound
	}
	var adm models.SysAdmin
	if err := e.Orm.First(&adm, req.UserId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrSysAdminNotFound
		}
		return err
	}
	if adm.DeletedAt.Valid {
		return ErrSysAdminNotFound
	}
	updates := map[string]interface{}{
		"must_change_password": req.Must,
		"updated_at":           time.Now(),
	}
	if req.UpdateBy > 0 {
		updates["update_by"] = int64(req.UpdateBy)
	}
	return e.Orm.Model(&models.SysAdmin{}).Where("id = ?", adm.Id).Updates(updates).Error
}

// EnforceLoginRestrictions 供登录链路调用：在认证通过后额外校验 IP / 时间窗。
// must_change_password 不在这里拒绝，由调用方透出给前端决定交互（允许登录但强制进入改密页）。
func (e *SysAdmin) EnforceLoginRestrictions(adm *models.SysAdmin, clientIP string, when time.Time) error {
	if adm == nil {
		return ErrSysAdminNotFound
	}
	if !ipAllowed(adm.AllowedIPs, clientIP) {
		return ErrSysAdminIPRejected
	}
	var start, end string
	if adm.AllowedLoginStart != nil {
		start = *adm.AllowedLoginStart
	}
	if adm.AllowedLoginEnd != nil {
		end = *adm.AllowedLoginEnd
	}
	if !timeWindowAllowed(start, end, when) {
		return ErrSysAdminTimeRejected
	}
	return nil
}

// ===== 参数校验辅助 =====

func sysAdminUsernamePolicy(s string) error {
	if s == "" {
		return errors.New("用户名不能为空")
	}
	if l := len(s); l < 6 || l > 20 {
		return errors.New("用户名长度必须在 6 到 20 位之间")
	}
	for _, ch := range s {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
			return errors.New("用户名只能包含英文字母和数字")
		}
	}
	return nil
}

func validateOptionalEmail(s string) error {
	if s == "" {
		return nil
	}
	atCount := 0
	valid := true
	for i, ch := range s {
		if ch == '@' {
			atCount++
			if i == 0 || i == len(s)-1 {
				valid = false
				break
			}
		}
	}
	if atCount != 1 || !valid {
		return errors.New("邮箱格式不正确")
	}
	return nil
}

func validateOptionalPhone(s string) error {
	if s == "" {
		return nil
	}
	if len(s) != 11 {
		return errors.New("手机号格式不正确")
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return errors.New("手机号格式不正确")
		}
	}
	return nil
}

// sysAdminPasswordPolicy 至少 8 位、含大小写与数字；与 setup.go 中 validatePasswordPolicy 规则略有差异（这里不要求特殊字符，与前端提示保持一致）。
func sysAdminPasswordPolicy(pwd string) error {
	if l := len(pwd); l < 8 || l > 20 {
		return ErrSysAdminPwdPolicy
	}
	var hasU, hasL, hasD bool
	for _, ch := range pwd {
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasU = true
		case ch >= 'a' && ch <= 'z':
			hasL = true
		case ch >= '0' && ch <= '9':
			hasD = true
		}
	}
	if !(hasU && hasL && hasD) {
		return ErrSysAdminPwdPolicy
	}
	return nil
}

// normalizeAllowedIPs 去空 / 去重 / 保持原始顺序。
func normalizeAllowedIPs(raw string) string {
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return strings.Join(out, ",")
}

// validateLoginWindow 接受空串（代表不限），否则必须解析为 HH:MM 或 HH:MM:SS；
// 不校验 start < end，以便支持跨零点窗口（例如 22:00 - 06:00）。
func validateLoginWindow(start, end string) (*string, *string, error) {
	parse := func(s string) (*string, error) {
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, nil
		}
		var t time.Time
		var err error
		for _, layout := range []string{"15:04", "15:04:05"} {
			t, err = time.Parse(layout, s)
			if err == nil {
				break
			}
		}
		if err != nil {
			return nil, fmt.Errorf("时间格式必须为 HH:MM：%q", s)
		}
		v := t.Format("15:04:05")
		return &v, nil
	}
	s, err := parse(start)
	if err != nil {
		return nil, nil, err
	}
	en, err := parse(end)
	if err != nil {
		return nil, nil, err
	}
	return s, en, nil
}

// ipAllowed 空白名单 = 允许所有；单个 token 支持 IPv4 / IPv6 / CIDR。
func ipAllowed(list, ip string) bool {
	list = strings.TrimSpace(list)
	if list == "" {
		return true
	}
	target := net.ParseIP(strings.TrimSpace(ip))
	if target == nil {
		return false
	}
	for _, token := range strings.Split(list, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if strings.Contains(token, "/") {
			if _, cidr, err := net.ParseCIDR(token); err == nil && cidr.Contains(target) {
				return true
			}
			continue
		}
		if net.ParseIP(token).Equal(target) {
			return true
		}
	}
	return false
}

// timeWindowAllowed 任一端为空 = 允许所有；支持跨零点窗口。
func timeWindowAllowed(start, end string, when time.Time) bool {
	if strings.TrimSpace(start) == "" || strings.TrimSpace(end) == "" {
		return true
	}
	ts, err := time.Parse("15:04:05", start)
	if err != nil {
		return true
	}
	te, err := time.Parse("15:04:05", end)
	if err != nil {
		return true
	}
	w := when.In(time.Local)
	cur := time.Date(0, 1, 1, w.Hour(), w.Minute(), w.Second(), 0, time.UTC)
	if ts.Equal(te) {
		return cur.Equal(ts)
	}
	if ts.Before(te) {
		return !cur.Before(ts) && !cur.After(te)
	}
	return !cur.Before(ts) || !cur.After(te)
}
