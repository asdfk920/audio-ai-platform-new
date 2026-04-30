package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	log "github.com/go-admin-team/go-admin-core/logger"
	"github.com/go-admin-team/go-admin-core/sdk/service"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"go-admin/app/admin/user/service/dto"
	"go-admin/common/actions"
	"go-admin/common/rbac"
	cDto "go-admin/common/dto"
)

type PlatformUserListService struct {
	service.Service
}

// GetPage 获取平台用户列表
func (s *PlatformUserListService) GetPage(req *dto.PlatformUserListReq, p *actions.DataPermission, list *[]dto.PlatformUserListItem, count *int64) error {
	if s.Orm == nil {
		return fmt.Errorf("orm nil")
	}

	// 基于 public.users 表进行查询；不能直接 Find(dto.PlatformUserListItem)，否则 GORM 会将 struct 推断为表名导致报错。
	base := s.Orm.
		Table("users").
		Scopes(
			cDto.MakeCondition(req.GetNeedSearch()),
		).
		Where("deleted_at IS NULL").
		// 仅过滤仍挂在 C 端 users 上的控制台角色（历史数据）；新管理员只存在于 sys_admin，不会出现在 users 中
		Where(`
NOT EXISTS (
  SELECT 1 FROM user_role_rel ur
  INNER JOIN roles r ON r.id = ur.role_id
  WHERE ur.user_id = users.id
    AND LOWER(TRIM(COALESCE(NULLIF(TRIM(r.slug), ''), r.name))) IN ?
)`, rbac.ConsoleAdminSlugs)

	if req.MemberLevel != nil {
		base = base.Where(
			"EXISTS (SELECT 1 FROM user_member um WHERE um.user_id = users.id AND um.level = ?)",
			*req.MemberLevel,
		)
	}

	// 先 count（不带 limit/offset）
	if err := base.Count(count).Error; err != nil {
		log.Errorf("db count error: %s", err)
		return err
	}

	// 再查分页数据
	err := base.
		Select(strings.Join([]string{
			"users.id as user_id",
			"coalesce(users.username,'') as username",
			"coalesce(users.real_name,'') as real_name",
			"coalesce(users.mobile,'') as mobile",
			"coalesce(users.email,'') as email",
			"coalesce(users.nickname,'') as nickname",
			"coalesce(users.avatar,'') as avatar",
			"coalesce(users.gender,0) as gender",
			"coalesce((SELECT um.level FROM user_member um WHERE um.user_id = users.id LIMIT 1),0) as member_level",
			"0 as member_expire_at",
			"coalesce(users.status,1) as status",
			"coalesce(users.real_name_status,0) as real_name_status",
			"users.created_at",
			"users.updated_at",
		}, ",")).
		Scopes(
			cDto.Paginate(req.GetPageSize(), req.GetPageIndex()),
		).
		Scan(list).Error
	if err != nil {
		log.Errorf("db list error: %s", err)
		return err
	}

	s.fillListRoleNames(list)

	// 补充额外信息：绑定设备数、会员等级名称等
	for i := range *list {
		item := &(*list)[i]

		// 查询绑定设备数量
		deviceCount, err := s.getBindDeviceCount(item.UserId)
		if err != nil {
			log.Errorf("查询用户 %d 绑定设备数失败：%v", item.UserId, err)
			// 不影响主流程，继续处理
		}
		item.BindDeviceCount = deviceCount

		// 转换时间戳
		item.RegisterTime = item.CreatedAt.Unix()

		// 会员等级名称映射
		item.MemberLevelName = s.getMemberLevelName(item.MemberLevel)
	}

	return nil
}

// fillListRoleNames 批量填充角色名称（逗号分隔）
func (s *PlatformUserListService) fillListRoleNames(list *[]dto.PlatformUserListItem) {
	if s.Orm == nil || list == nil || len(*list) == 0 {
		return
	}
	ids := make([]int64, 0, len(*list))
	for i := range *list {
		ids = append(ids, (*list)[i].UserId)
	}
	if len(ids) == 0 {
		return
	}
	var rows []struct {
		UserID int64  `gorm:"column:user_id"`
		Name   string `gorm:"column:name"`
	}
	err := s.Orm.Table("user_role_rel ur").
		Select("ur.user_id, r.name").
		Joins("JOIN roles r ON r.id = ur.role_id").
		Where("ur.user_id IN ?", ids).
		Scan(&rows).Error
	if err != nil {
		log.Errorf("fillListRoleNames: %v", err)
		return
	}
	m := make(map[int64][]string)
	for _, r := range rows {
		m[r.UserID] = append(m[r.UserID], r.Name)
	}
	for i := range *list {
		id := (*list)[i].UserId
		if names, ok := m[id]; ok {
			(*list)[i].RoleNames = strings.Join(names, ", ")
		}
	}
}

func (s *PlatformUserListService) userHasRoleName(userId int64, roleName string) (bool, error) {
	var n int64
	err := s.Orm.Table("user_role_rel ur").
		Joins("INNER JOIN roles r ON r.id = ur.role_id").
		Where("ur.user_id = ? AND r.name = ?", userId, roleName).
		Count(&n).Error
	return n > 0, err
}

// userIDExcludedFromPlatform 仍挂控制台角色的 C 端 users 行（历史）不参与平台用户运营能力；新管理员仅 sys_admin，与 users 无关联
func (s *PlatformUserListService) userIDExcludedFromPlatform(userID int64) (bool, error) {
	if s.Orm == nil {
		return false, fmt.Errorf("orm nil")
	}
	var n int64
	err := s.Orm.Table("user_role_rel ur").
		Joins("INNER JOIN roles r ON r.id = ur.role_id").
		Where("ur.user_id = ?", userID).
		Where("LOWER(TRIM(COALESCE(NULLIF(TRIM(r.slug), ''), r.name))) IN ?", rbac.ConsoleAdminSlugs).
		Count(&n).Error
	return n > 0, err
}

// getBindDeviceCount 获取用户绑定设备数量
func (s *PlatformUserListService) getBindDeviceCount(userId int64) (int64, error) {
	var count int64

	// 查询 user_device_bind 表中该用户的绑定记录数
	err := s.Orm.Table("user_device_bind").
		Where("user_id = ? AND status = 1", userId).
		Count(&count).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 记录不存在，返回 0
			return 0, nil
		}
		return 0, err
	}

	return count, nil
}

// getMemberLevelName 获取会员等级名称
func (s *PlatformUserListService) getMemberLevelName(level int32) string {
	levelNames := map[int32]string{
		0: "普通会员",
		1: "VIP 会员",
		2: "SVIP 会员",
		3: "终身会员",
	}

	if name, ok := levelNames[level]; ok {
		return name
	}
	return "普通会员"
}

// Get 获取用户详情（多维度关联查询）
func (s *PlatformUserListService) Get(userId int64) (*dto.PlatformUserInfoResp, error) {
	var row struct {
		ID             int64      `gorm:"column:id"`
		Username       string     `gorm:"column:username"`
		RealName       string     `gorm:"column:real_name"`
		Mobile         string     `gorm:"column:mobile"`
		Email          string     `gorm:"column:email"`
		Nickname       string     `gorm:"column:nickname"`
		Avatar         string     `gorm:"column:avatar"`
		Gender         *int32     `gorm:"column:gender"`
		Birthday       *time.Time `gorm:"column:birthday"`
		Status         int32      `gorm:"column:status"`
		RealNameStatus int32      `gorm:"column:real_name_status"`
		LastLoginAt    *time.Time `gorm:"column:last_login_at"`
		LastLoginIP    string     `gorm:"column:last_login_ip"`
		CreatedAt      time.Time  `gorm:"column:created_at"`
		UpdatedAt      time.Time  `gorm:"column:updated_at"`
	}

	err := s.Orm.Table("users").
		Where("id = ? AND deleted_at IS NULL", userId).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, fmt.Errorf("查询用户基础信息失败：%w", err)
	}

	if skip, err := s.userIDExcludedFromPlatform(row.ID); err != nil {
		return nil, err
	} else if skip {
		return nil, errors.New("用户不存在")
	}

	user := dto.PlatformUserInfoResp{
		UserId:         row.ID,
		Username:       row.Username,
		RealName:       row.RealName,
		Mobile:         s.maskMobile(row.Mobile),
		Email:          row.Email,
		Nickname:       row.Nickname,
		Avatar:         row.Avatar,
		Status:         row.Status,
		RealNameStatus: row.RealNameStatus,
		LastLoginIP:    row.LastLoginIP,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
	if row.Gender != nil {
		user.Gender = *row.Gender
	}
	if row.Birthday != nil {
		user.Birthday = row.Birthday.Format("2006-01-02")
	}
	if row.LastLoginAt != nil {
		user.LastLoginTime = row.LastLoginAt.Unix()
	}
	user.RegisterTime = row.CreatedAt.Unix()

	s.fillPlatformRoles(&user)

	// 会员信息
	s.fillMemberInfo(&user)

	// 设备信息
	s.fillDeviceInfo(&user, userId)

	// 实名认证信息
	s.fillRealNameInfo(&user, userId)

	// 会话信息
	s.fillSessionInfo(&user, userId)

	return &user, nil
}

func (s *PlatformUserListService) fillPlatformRoles(user *dto.PlatformUserInfoResp) {
	var rows []struct {
		ID   int64  `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	err := s.Orm.Table("user_role_rel ur").
		Select("r.id, r.name").
		Joins("JOIN roles r ON r.id = ur.role_id").
		Where("ur.user_id = ?", user.UserId).
		Scan(&rows).Error
	if err != nil {
		log.Errorf("fillPlatformRoles: %v", err)
		return
	}
	for _, r := range rows {
		user.Roles = append(user.Roles, dto.PlatformUserRoleItem{
			ID: r.ID, RoleKey: r.Name, Name: r.Name,
		})
	}
}

// fillMemberInfo 填充会员信息（与内容服务 EffectiveVipLevel 规则对齐：后台展示的「当前权益档位」与列表/鉴权一致）
func (s *PlatformUserListService) fillMemberInfo(user *dto.PlatformUserInfoResp) {
	var member struct {
		Level        int32      `gorm:"column:level"`
		ExpireAt     *time.Time `gorm:"column:expire_at"`
		ExpiredAt    *time.Time `gorm:"column:expired_at"`
		Status       int32      `gorm:"column:status"`
		IsPermanent  int16      `gorm:"column:is_permanent"`
		CreatedAt    time.Time  `gorm:"column:created_at"`
	}

	err := s.Orm.Table("user_member").
		Where("user_id = ?", user.UserId).
		First(&member).Error

	if err == nil {
		var exp *time.Time
		if member.ExpireAt != nil {
			exp = member.ExpireAt
		} else if member.ExpiredAt != nil {
			exp = member.ExpiredAt
		}
		if exp != nil {
			user.MemberExpireAt = exp.Unix()
		}
		user.MemberCreatedAt = member.CreatedAt.Unix()

		effLevel, memStat := platformEffectiveMemberLevel(member.Level, member.Status, member.IsPermanent, exp)
		user.MemberLevel = effLevel
		user.MemberStatus = memStat
	}
	user.MemberLevelName = s.getMemberLevelName(user.MemberLevel)
}

// platformEffectiveMemberLevel 与 services/content/internal/repo/dao.EffectiveVipLevel 语义一致。
// 返回：对外有效等级；会员状态 0 正常 1 过期 2 冻结（与 dto 注释一致）。
func platformEffectiveMemberLevel(level int32, status int32, isPermanent int16, expire *time.Time) (effLevel int32, memberStatus int32) {
	if status != 1 {
		return 0, 2
	}
	if isPermanent == 1 {
		return level, 0
	}
	if expire == nil {
		return 0, 1
	}
	if !expire.After(time.Now()) {
		return 0, 1
	}
	return level, 0
}

// fillDeviceInfo 填充设备信息
func (s *PlatformUserListService) fillDeviceInfo(user *dto.PlatformUserInfoResp, userId int64) {
	// 查询绑定设备总数
	totalCount, _ := s.getBindDeviceCount(userId)
	user.BindDeviceCount = totalCount

	// 查询在线设备数量
	onlineCount, _ := s.getOnlineDeviceCount(userId)
	user.OnlineDeviceCount = onlineCount

	// 查询设备列表（最近绑定的 10 个设备）
	var devices []struct {
		DeviceSn   string    `gorm:"column:device_sn"`
		DeviceName string    `gorm:"column:device_name"`
		Model      string    `gorm:"column:model"`
		OnlineSt   int16     `gorm:"column:online_status"`
		BindTime   time.Time `gorm:"column:bind_time"`
	}
	err := s.Orm.Table("user_device_bind as udb").
		Select("d.sn as device_sn, d.name as device_name, d.model, d.online_status, udb.bound_at as bind_time").
		Joins("LEFT JOIN device d ON udb.sn = d.sn").
		Where("udb.user_id = ? AND udb.status = 1", userId).
		Order("udb.bound_at DESC").
		Limit(10).
		Scan(&devices).Error

	if err == nil {
		for _, dev := range devices {
			user.DeviceList = append(user.DeviceList, dto.UserDeviceItem{
				DeviceSn:   dev.DeviceSn,
				DeviceName: dev.DeviceName,
				Model:      dev.Model,
				Online:     dev.OnlineSt == 1,
				BindTime:   dev.BindTime.Unix(),
			})
		}
	}
}

// fillRealNameInfo 填充实名认证信息
func (s *PlatformUserListService) fillRealNameInfo(user *dto.PlatformUserInfoResp, userId int64) {
	var realName struct {
		Status      int32     `gorm:"column:status"`
		RealName    string    `gorm:"column:real_name"`
		IDCard      string    `gorm:"column:id_card"`
		SubmitTime  time.Time `gorm:"column:submit_time"`
		AuditTime   time.Time `gorm:"column:audit_time"`
		Auditor     string    `gorm:"column:auditor"`
		AuditRemark string    `gorm:"column:audit_remark"`
	}

	err := s.Orm.Table("user_real_name").
		Where("user_id = ?", userId).
		First(&realName).Error

	if err == nil {
		user.RealNameInfo = &dto.RealNameInfo{
			Status:      realName.Status,
			RealName:    s.maskRealName(realName.RealName),
			IDCard:      s.maskIDCard(realName.IDCard),
			SubmitTime:  realName.SubmitTime.Unix(),
			AuditTime:   realName.AuditTime.Unix(),
			Auditor:     realName.Auditor,
			AuditRemark: realName.AuditRemark,
		}
	}
}

// fillSessionInfo 填充会话信息
func (s *PlatformUserListService) fillSessionInfo(user *dto.PlatformUserInfoResp, userId int64) {
	// 查询当前有效会话数
	var sessionCount int64
	err := s.Orm.Table("user_session").
		Where("user_id = ? AND status = 1", userId).
		Count(&sessionCount).Error

	if err == nil {
		user.ActiveSessionCount = sessionCount
	}

	// 查询最近登录记录（最近 5 条）
	var loginLogs []struct {
		LoginTime time.Time `gorm:"column:login_time"`
		Device    string    `gorm:"column:device"`
		IP        string    `gorm:"column:login_ip"`
		Location  string    `gorm:"column:location"`
	}

	err = s.Orm.Table("sys_login_log").
		Where("username = ?", user.Mobile).
		Order("login_time DESC").
		Limit(5).
		Scan(&loginLogs).Error

	if err == nil {
		for _, log := range loginLogs {
			login := dto.RecentLogin{
				LoginTime: log.LoginTime.Unix(),
				Device:    log.Device,
				IP:        log.IP,
				Location:  log.Location,
			}
			user.RecentLogins = append(user.RecentLogins, login)
		}
	}
}

// getOnlineDeviceCount 获取用户在线设备数量
func (s *PlatformUserListService) getOnlineDeviceCount(userId int64) (int64, error) {
	var count int64

	err := s.Orm.Table("user_device_bind as udb").
		Joins("LEFT JOIN device d ON udb.sn = d.sn").
		Where("udb.user_id = ? AND udb.status = 1 AND d.online_status = 1", userId).
		Count(&count).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}

	return count, nil
}

// maskMobile 手机号脱敏
func (s *PlatformUserListService) maskMobile(mobile string) string {
	if len(mobile) < 11 {
		return mobile
	}
	return mobile[:3] + " " + mobile[3:7] + " " + mobile[7:]
}

// maskRealName 姓名脱敏
func (s *PlatformUserListService) maskRealName(name string) string {
	if len(name) <= 2 {
		return name[:1] + "*"
	}
	return name[:1] + "*" + name[len(name)-1:]
}

// maskIDCard 身份证号脱敏
func (s *PlatformUserListService) maskIDCard(idCard string) string {
	if len(idCard) < 14 {
		return idCard
	}
	return idCard[:6] + "******" + idCard[len(idCard)-4:]
}

// 平台用户手动新增/修改相关错误；API 层依据类型映射到中文提示，确保不暴露数据库细节。
var (
	ErrPlatformUserForbidden       = errors.New("only super_admin can add platform users")
	ErrPlatformUserInvalidUsername = errors.New("invalid username")
	ErrPlatformUserInvalidPassword = errors.New("invalid password")
	ErrPlatformUserDuplicate       = errors.New("username/mobile/email already exists")
	ErrPlatformUserRoleNotFound    = errors.New("normal user role (slug=user) not configured in roles table")
	ErrPlatformUserNotFound        = errors.New("platform user not found")
)

// 平台用户用户名规则：字母开头，长度 3~64，仅字母数字与下划线。
var platformUsernameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{2,63}$`)

// IsCurrentAdminSuperAdmin 检查 JWT 中当前登录管理员（adminID 对应 sys_admin.id）
// 是否为「超级管理员」（role_code = 'super_admin' 或 roles.slug = 'super_admin'）。
//
// 返回值：
//   - true：允许继续执行后台敏感操作（例如手动新增平台用户）
//   - false：当前管理员不存在 / 已被禁用 / 不是超级管理员
func (s *PlatformUserListService) IsCurrentAdminSuperAdmin(adminID int64) (bool, error) {
	if s.Orm == nil {
		return false, fmt.Errorf("orm nil")
	}
	if adminID <= 0 {
		return false, nil
	}
	sysAdminTable := "sys_admin"
	rolesTable := "roles"
	if s.Orm.Dialector != nil && s.Orm.Dialector.Name() == "postgres" {
		sysAdminTable = "public.sys_admin"
		rolesTable = "public.roles"
	}
	var row struct {
		Status   int32
		RoleCode string
		RoleSlug string
	}
	sql := fmt.Sprintf(`
SELECT
    a.status AS status,
    COALESCE(a.role_code, '') AS role_code,
    COALESCE(r.slug, '') AS role_slug
FROM %s a
LEFT JOIN %s r ON r.id = a.role_id
WHERE a.id = ? AND a.deleted_at IS NULL
LIMIT 1`, sysAdminTable, rolesTable)
	if err := s.Orm.Raw(sql, adminID).Scan(&row).Error; err != nil {
		return false, err
	}
	if row.Status != 1 {
		return false, nil
	}
	rc := strings.ToLower(strings.TrimSpace(row.RoleCode))
	rs := strings.ToLower(strings.TrimSpace(row.RoleSlug))
	return rc == "super_admin" || rs == "super_admin", nil
}

// Create 后台手动新增平台用户
//
// 前置条件（由 API 层保证）：caller 必须是 sys_admin 中的超级管理员。
// 本方法实现三个业务约束：
//  1. 角色只允许「普通用户」(roles.slug = 'user')，忽略入参中其他 role_ids；
//  2. 新增后数据仅落 public.users + public.user_role_rel，绝不触碰 sys_admin；
//  3. 唯一性校验只在 public.users 命名空间内，与 sys_admin 冲突的用户名可以共存。
func (s *PlatformUserListService) Create(req *dto.PlatformUserCreateReq) error {
	if s.Orm == nil {
		return fmt.Errorf("orm nil")
	}
	if req == nil {
		return fmt.Errorf("empty request")
	}

	username := strings.TrimSpace(req.Username)
	password := req.Password
	mobile := strings.TrimSpace(req.Mobile)
	email := strings.TrimSpace(req.Email)
	realName := strings.TrimSpace(req.RealName)
	nickname := strings.TrimSpace(req.Nickname)
	avatar := strings.TrimSpace(req.Avatar)

	if !platformUsernameRe.MatchString(username) {
		return ErrPlatformUserInvalidUsername
	}
	if len(password) < 6 {
		return ErrPlatformUserInvalidPassword
	}
	if mobile == "" && email == "" {
		return fmt.Errorf("手机号和邮箱不能同时为空")
	}

	// 唯一性仅校验 users；不与 sys_admin 关联
	var dup int64
	dupQ := s.Orm.Table("users").Where("deleted_at IS NULL").
		Where("LOWER(TRIM(username)) = LOWER(TRIM(?))", username)
	if err := dupQ.Count(&dup).Error; err != nil {
		log.Errorf("platform-user username duplicate check error: %v", err)
		return err
	}
	if dup == 0 && mobile != "" {
		if err := s.Orm.Table("users").Where("deleted_at IS NULL").
			Where("mobile = ?", mobile).Count(&dup).Error; err != nil {
			log.Errorf("platform-user mobile duplicate check error: %v", err)
			return err
		}
	}
	if dup == 0 && email != "" {
		if err := s.Orm.Table("users").Where("deleted_at IS NULL").
			Where("LOWER(email) = LOWER(?)", email).Count(&dup).Error; err != nil {
			log.Errorf("platform-user email duplicate check error: %v", err)
			return err
		}
	}
	if dup > 0 {
		return ErrPlatformUserDuplicate
	}

	// 查询「普通用户」角色（slug = 'user'）
	rolesTable := "roles"
	if s.Orm.Dialector != nil && s.Orm.Dialector.Name() == "postgres" {
		rolesTable = "public.roles"
	}
	var roleID int64
	if err := s.Orm.Raw(fmt.Sprintf(`SELECT id FROM %s WHERE LOWER(TRIM(slug)) = 'user' LIMIT 1`, rolesTable)).Scan(&roleID).Error; err != nil {
		log.Errorf("lookup normal user role failed: %v", err)
		return err
	}
	if roleID == 0 {
		return ErrPlatformUserRoleNotFound
	}

	// 密码哈希：与 C 端登录保持一致（bcrypt(salt + password) + password_algo = bcrypt_concat）
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return err
	}
	salt := hex.EncodeToString(saltBytes)
	hashed, err := bcrypt.GenerateFromPassword([]byte(salt+password), 10)
	if err != nil {
		return err
	}

	// status 默认 1，gender 默认 0
	status := int32(1)
	if req.Status != nil {
		status = *req.Status
	}
	var gender *int32
	if req.Gender != nil {
		gender = req.Gender
	}
	var birthday *string
	if req.Birthday != nil && strings.TrimSpace(*req.Birthday) != "" {
		v := strings.TrimSpace(*req.Birthday)
		birthday = &v
	}

	now := time.Now()
	return s.Orm.Transaction(func(tx *gorm.DB) error {
		row := map[string]interface{}{
			"username":            username,
			"password":            string(hashed),
			"salt":                salt,
			"password_algo":       "bcrypt_concat",
			"password_changed_at": now,
			"real_name":           realName,
			"nickname":            nickname,
			"mobile":              nullableString(mobile),
			"email":               nullableString(email),
			"avatar":              avatar,
			"status":              status,
			"user_type":           1,
			"register_channel":    "admin",
			"created_at":          now,
			"updated_at":          now,
		}
		if gender != nil {
			row["gender"] = *gender
		}
		if birthday != nil {
			row["birthday"] = *birthday
		}

		if err := tx.Table("users").Create(&row).Error; err != nil {
			log.Errorf("insert users failed: %v", err)
			return err
		}
		// users.id 为 BIGSERIAL；插入后按用户名回查以拿到自增 id（避免依赖 GORM 对 map 的 id 回写行为差异）
		var newID int64
		if err := tx.Table("users").
			Select("id").
			Where("LOWER(TRIM(username)) = LOWER(TRIM(?)) AND deleted_at IS NULL", username).
			Scan(&newID).Error; err != nil {
			return err
		}
		if newID == 0 {
			return fmt.Errorf("insert users succeeded but id lookup failed")
		}

		// 强制只绑定「普通用户」角色，忽略 req.RoleIds 里的其他值
		relSQL := `INSERT INTO user_role_rel (user_id, role_id, created_at) VALUES (?, ?, ?) ON CONFLICT (user_id, role_id) DO NOTHING`
		if s.Orm.Dialector != nil && s.Orm.Dialector.Name() != "postgres" {
			// 非 PG 走不带 ON CONFLICT 的兜底
			relSQL = `INSERT INTO user_role_rel (user_id, role_id, created_at) VALUES (?, ?, ?)`
		}
		if err := tx.Exec(relSQL, newID, roleID, now).Error; err != nil {
			log.Errorf("insert user_role_rel failed: %v", err)
			return err
		}
		return nil
	})
}

// nullableString 空字符串返回 nil（让 INSERT 走 DEFAULT/NULL，避免违反 UNIQUE 约束时空串被当作真实值冲突）。
func nullableString(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// GetUserInfo 获取用户详细信息
func (s *PlatformUserListService) GetUserInfo(userId int64) (*dto.UserInfo, error) {
	// 1. 查询用户基本信息
	var user struct {
		Username    string
		Nickname    string
		Avatar      string
		Email       string
		RoleKey     string
		Permissions string
	}

	err := s.Orm.Table("users u").
		Select("u.username, u.nickname, u.avatar, u.email, r.name as role_key, r.permissions::text").
		Joins("LEFT JOIN user_role_rel ur ON u.id = ur.user_id").
		Joins("LEFT JOIN roles r ON ur.role_id = r.id").
		Where("u.id = ? AND u.deleted_at IS NULL", userId).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 返回默认用户信息，避免前端报错
			return &dto.UserInfo{
				Roles:       []string{"user"},
				Name:        "user",
				Avatar:      "",
				Intro:       "",
				Permissions: []string{},
			}, nil
		}
		log.Errorf("查询用户信息失败：%v", err)
		return nil, fmt.Errorf("查询用户信息失败：%w", err)
	}

	// 2. 解析权限列表（逗号分隔）
	var permissions []string
	if user.Permissions != "" {
		// 简单分割，实际可能需要更复杂的解析逻辑
		permissions = []string{user.Permissions}
	} else {
		permissions = []string{}
	}

	// 3. 封装返回结果
	roles := []string{"user"} // 默认角色
	if user.RoleKey != "" {
		roles = []string{user.RoleKey}
	}

	info := &dto.UserInfo{
		Roles:       roles,
		Name:        user.Username,
		Avatar:      user.Avatar,
		Intro:       user.Nickname, // 使用昵称作为简介
		Permissions: permissions,
	}

	return info, nil
}

// Update 更新平台用户（走 users 表；与 sys_admin 完全解耦）
//
// 业务约束：
//  1. 目标 user_id 必须在 public.users 且未软删除；
//  2. 若修改 mobile / email，会校验不与其他 users 冲突（自身原值可以保留）；
//  3. 字段级更新：只有显式传入的字段才会写回；生日空字符串被视为「清空」；
//  4. 不修改用户名、密码、角色（角色通过独立接口 /roles 写 user_role_rel）。
func (s *PlatformUserListService) Update(req *dto.PlatformUserUpdateReq) error {
	if s.Orm == nil {
		return fmt.Errorf("orm nil")
	}
	if req == nil || req.UserId <= 0 {
		return ErrPlatformUserNotFound
	}

	var exist struct {
		ID       int64
		Username string
	}
	err := s.Orm.Table("users").
		Select("id, username").
		Where("id = ? AND deleted_at IS NULL", req.UserId).
		First(&exist).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPlatformUserNotFound
		}
		log.Errorf("查询用户失败：%v", err)
		return fmt.Errorf("查询用户失败：%w", err)
	}
	if skip, e2 := s.userIDExcludedFromPlatform(exist.ID); e2 != nil {
		return e2
	} else if skip {
		return ErrPlatformUserNotFound
	}

	// 手机/邮箱变更时先做唯一性校验（排除自身）
	mobile := strings.TrimSpace(req.Mobile)
	email := strings.TrimSpace(req.Email)
	if mobile != "" {
		var cnt int64
		if err := s.Orm.Table("users").
			Where("deleted_at IS NULL AND id <> ? AND mobile = ?", req.UserId, mobile).
			Count(&cnt).Error; err != nil {
			return fmt.Errorf("手机号唯一性校验失败：%w", err)
		}
		if cnt > 0 {
			return ErrPlatformUserDuplicate
		}
	}
	if email != "" {
		var cnt int64
		if err := s.Orm.Table("users").
			Where("deleted_at IS NULL AND id <> ? AND LOWER(email) = LOWER(?)", req.UserId, email).
			Count(&cnt).Error; err != nil {
			return fmt.Errorf("邮箱唯一性校验失败：%w", err)
		}
		if cnt > 0 {
			return ErrPlatformUserDuplicate
		}
	}

	updateData := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if req.Nickname != "" {
		updateData["nickname"] = req.Nickname
	}
	if req.Avatar != "" {
		updateData["avatar"] = req.Avatar
	}
	if req.Status != nil {
		updateData["status"] = *req.Status
	}
	if req.RealName != "" {
		updateData["real_name"] = req.RealName
	}
	if mobile != "" {
		updateData["mobile"] = mobile
	}
	if email != "" {
		updateData["email"] = email
	}
	if req.Gender != nil {
		updateData["gender"] = *req.Gender
	}
	// 生日：nil -> 保持原值；空字符串 -> 清空；非空 -> 更新
	if req.Birthday != nil {
		bs := strings.TrimSpace(*req.Birthday)
		if bs == "" {
			updateData["birthday"] = nil
		} else {
			updateData["birthday"] = bs
		}
	}

	if err := s.Orm.Transaction(func(tx *gorm.DB) error {
		return tx.Table("users").
			Where("id = ? AND deleted_at IS NULL", req.UserId).
			Updates(updateData).Error
	}); err != nil {
		log.Errorf("更新用户信息失败：%v", err)
		return fmt.Errorf("更新用户信息失败：%w", err)
	}

	log.Infof("成功更新用户 %d 信息", req.UserId)
	return nil
}

// Delete 删除用户
func (s *PlatformUserListService) Delete(userId int64) error {
	// TODO: 实现删除用户逻辑
	return errors.New("暂未实现")
}

// SetUserRoles 覆盖式设置平台用户的角色集合（users + user_role_rel，与 sys_admin 完全解耦）。
//
// 业务约束：
//  1. 目标 users 行必须存在且未软删除；
//  2. 禁止给已是「超级管理员」的 users 行改角色（避免越权）；
//  3. roleIds 允许为空（即撤销全部角色），但如果非空，所有 id 必须在 public.roles 中真实存在；
//  4. 以「先全删后全插」的事务完成覆盖；user_role_rel 的 (user_id, role_id) 已有唯一约束。
func (s *PlatformUserListService) SetUserRoles(userId int64, roleIds []int64) error {
	if s.Orm == nil {
		return fmt.Errorf("orm nil")
	}
	if userId <= 0 {
		return ErrPlatformUserNotFound
	}

	var exist struct {
		ID int64
	}
	if err := s.Orm.Table("users").
		Select("id").
		Where("id = ? AND deleted_at IS NULL", userId).
		First(&exist).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPlatformUserNotFound
		}
		return fmt.Errorf("查询用户失败：%w", err)
	}
	if skip, e2 := s.userIDExcludedFromPlatform(exist.ID); e2 != nil {
		return e2
	} else if skip {
		return ErrPlatformUserNotFound
	}

	// 禁止修改已是超级管理员的用户（防止越权降权或误操作）
	if isSuper, err := s.userHasRoleName(userId, "super_admin"); err != nil {
		return fmt.Errorf("查询用户角色失败：%w", err)
	} else if isSuper {
		return fmt.Errorf("禁止修改超级管理员角色")
	}

	// 去重、剔除非正数
	seen := make(map[int64]struct{}, len(roleIds))
	cleaned := make([]int64, 0, len(roleIds))
	for _, id := range roleIds {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		cleaned = append(cleaned, id)
	}

	if len(cleaned) > 0 {
		var cnt int64
		if err := s.Orm.Table("roles").
			Where("id IN ?", cleaned).
			Count(&cnt).Error; err != nil {
			return fmt.Errorf("校验角色失败：%w", err)
		}
		if int(cnt) != len(cleaned) {
			return ErrPlatformUserRoleNotFound
		}
	}

	return s.Orm.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM user_role_rel WHERE user_id = ?`, userId).Error; err != nil {
			return fmt.Errorf("清除旧角色失败：%w", err)
		}
		if len(cleaned) == 0 {
			return nil
		}
		now := time.Now()
		for _, rid := range cleaned {
			if err := tx.Exec(
				`INSERT INTO user_role_rel (user_id, role_id, created_at) VALUES (?, ?, ?) ON CONFLICT (user_id, role_id) DO NOTHING`,
				userId, rid, now,
			).Error; err != nil {
				return fmt.Errorf("写入角色关系失败：%w", err)
			}
		}
		return nil
	})
}

// UpdateUserStatus 禁用/启用用户
func (s *PlatformUserListService) UpdateUserStatus(req *dto.UpdateUserStatusReq, adminId int64, clientIP string) (*dto.UpdateUserStatusResp, error) {
	if s.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}

	// 1. 校验目标用户是否存在
	var user struct {
		ID       int64
		Username string
		Status   int32
	}

	err := s.Orm.Table("users").
		Select("id, username, status").
		Where("id = ? AND deleted_at IS NULL", req.UserId).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		log.Errorf("查询用户失败：%v", err)
		return nil, fmt.Errorf("查询用户失败：%w", err)
	}
	if skip, e2 := s.userIDExcludedFromPlatform(user.ID); e2 != nil {
		return nil, e2
	} else if skip {
		return nil, fmt.Errorf("用户不存在")
	}

	// 2. 禁止操作超级管理员账号（roles.name = super_admin）
	isSuper, err := s.userHasRoleName(req.UserId, "super_admin")
	if err != nil {
		log.Errorf("查询用户角色失败：%v", err)
		return nil, fmt.Errorf("查询用户角色失败：%w", err)
	}
	if isSuper {
		return nil, fmt.Errorf("禁止操作超级管理员账号")
	}

	// 3. 禁止操作自身账号
	if user.ID == adminId {
		return nil, fmt.Errorf("禁止操作自身账号")
	}

	// 4. 幂等性校验：若目标状态与当前状态一致，直接返回成功
	if user.Status == req.Status {
		statusText := "已启用"
		if req.Status == 0 {
			statusText = "已禁用"
		}
		return &dto.UpdateUserStatusResp{
			UserId:     user.ID,
			Status:     req.Status,
			StatusText: statusText,
			UpdateTime: time.Now(),
		}, nil
	}

	// 5. 更新状态
	tx := s.Orm.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err = tx.Table("users").
		Where("id = ?", req.UserId).
		Updates(map[string]interface{}{
			"status":     req.Status,
			"updated_at": time.Now(),
		}).Error
	if err != nil {
		tx.Rollback()
		log.Errorf("更新用户状态失败：%v", err)
		return nil, fmt.Errorf("更新用户状态失败：%w", err)
	}

	if req.Status == 0 {
		log.Infof("用户 %d 已被禁用（TODO: 失效 Redis Token）", req.UserId)
	}

	log.Infof("管理员 %d 操作用户 %d 状态: %d -> %d, 原因: %s", adminId, req.UserId, user.Status, req.Status, req.Reason)

	if err := tx.Commit().Error; err != nil {
		log.Errorf("提交事务失败：%v", err)
		return nil, fmt.Errorf("提交事务失败：%w", err)
	}

	statusText := "已启用"
	if req.Status == 0 {
		statusText = "已禁用"
	}
	return &dto.UpdateUserStatusResp{
		UserId:     user.ID,
		Status:     req.Status,
		StatusText: statusText,
		UpdateTime: time.Now(),
	}, nil
}

// UpdateUserVipLevel 修改用户会员等级（写入 public.user_member，而非 users 表上不存在的 vip_level 列）
func (s *PlatformUserListService) UpdateUserVipLevel(req *dto.UpdateUserVipLevelReq, adminId int64, clientIP string) (*dto.UpdateUserVipLevelResp, error) {
	if s.Orm == nil {
		return nil, fmt.Errorf("orm nil")
	}

	var user struct {
		ID       int64
		Username string
		Status   int32
	}

	err := s.Orm.Table("users").
		Select("id, username, status").
		Where("id = ? AND deleted_at IS NULL", req.UserId).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		log.Errorf("查询用户失败：%v", err)
		return nil, fmt.Errorf("查询用户失败：%w", err)
	}
	if skip, e2 := s.userIDExcludedFromPlatform(user.ID); e2 != nil {
		return nil, e2
	} else if skip {
		return nil, fmt.Errorf("用户不存在")
	}

	isSuper, err := s.userHasRoleName(req.UserId, "super_admin")
	if err != nil {
		log.Errorf("查询用户角色失败：%v", err)
		return nil, fmt.Errorf("查询用户角色失败：%w", err)
	}
	if isSuper {
		return nil, fmt.Errorf("禁止操作超级管理员账号")
	}

	if user.ID == adminId {
		return nil, fmt.Errorf("禁止操作自身账号")
	}

	if user.Status == 0 {
		return nil, fmt.Errorf("用户已禁用")
	}

	var curMember struct {
		Level int32 `gorm:"column:level"`
	}
	currentLevel := int32(0)
	if takeErr := s.Orm.Table("user_member").Select("level").Where("user_id = ?", req.UserId).Take(&curMember).Error; takeErr == nil {
		currentLevel = curMember.Level
	} else if takeErr != nil && !errors.Is(takeErr, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("查询会员信息失败：%w", takeErr)
	}

	expireAt := time.Date(9999, 12, 31, 23, 59, 59, 0, time.Local)
	if strings.TrimSpace(req.VipExpireTime) != "" {
		raw := strings.TrimSpace(req.VipExpireTime)
		if t, e := time.ParseInLocation("2006-01-02 15:04:05", raw, time.Local); e == nil {
			expireAt = t
		} else if t, e := time.ParseInLocation("2006-01-02", raw, time.Local); e == nil {
			expireAt = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
	}

	if currentLevel == req.VipLevel {
		vipName := getVipLevelName(req.VipLevel)
		return &dto.UpdateUserVipLevelResp{
			UserId:        user.ID,
			VipLevel:      req.VipLevel,
			VipName:       vipName,
			VipExpireTime: req.VipExpireTime,
			UpdateTime:    time.Now(),
		}, nil
	}

	tx := s.Orm.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var exists struct {
		UserID int64 `gorm:"column:user_id"`
	}
	takeRowErr := tx.Table("user_member").Select("user_id").Where("user_id = ?", req.UserId).Take(&exists).Error
	if takeRowErr != nil && !errors.Is(takeRowErr, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return nil, fmt.Errorf("查询会员信息失败：%w", takeRowErr)
	}
	hasRow := takeRowErr == nil

	now := time.Now()
	up := map[string]interface{}{
		"level":         req.VipLevel,
		"expire_at":     expireAt,
		"expired_at":    expireAt,
		"updated_at":    now,
		"status":        int32(1),
		"grant_by":      adminId,
		"register_type": "admin",
	}

	if !hasRow {
		up["user_id"] = req.UserId
		up["created_at"] = now
		err = tx.Table("user_member").Create(up).Error
	} else {
		err = tx.Table("user_member").Where("user_id = ?", req.UserId).Updates(up).Error
	}

	if err != nil {
		tx.Rollback()
		log.Errorf("更新用户会员等级失败：%v", err)
		return nil, fmt.Errorf("更新用户会员等级失败：%w", err)
	}

	log.Infof("管理员 %d 修改用户 %d 会员等级：%d -> %d, 到期：%v, 原因：%s",
		adminId, req.UserId, currentLevel, req.VipLevel, expireAt, req.Reason)

	if err := tx.Commit().Error; err != nil {
		log.Errorf("提交事务失败：%v", err)
		return nil, fmt.Errorf("提交事务失败：%w", err)
	}

	vipName := getVipLevelName(req.VipLevel)

	return &dto.UpdateUserVipLevelResp{
		UserId:        user.ID,
		VipLevel:      req.VipLevel,
		VipName:       vipName,
		VipExpireTime: req.VipExpireTime,
		UpdateTime:    time.Now(),
	}, nil
}

// getVipLevelName 获取会员等级名称
func getVipLevelName(level int32) string {
	switch level {
	case 0:
		return "普通用户"
	case 1:
		return "银卡会员"
	case 2:
		return "金卡会员"
	case 3:
		return "至尊会员"
	default:
		return "普通用户"
	}
}


