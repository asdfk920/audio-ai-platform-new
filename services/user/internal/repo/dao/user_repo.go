package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/passwd"
	avatarutil "github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/avatar"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/userconst"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/auth/verifycode"
	"github.com/zeromicro/go-zero/core/logx"
)

// UserRepo 用户读写与查询（业务错误在层内转为 errorx）。
type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

// ---- Membership (user_member + member_level_benefit + member_benefit) ----

// MemberBenefitRow 会员权益行（用于用户侧查询）。
type MemberBenefitRow struct {
	BenefitCode string
	BenefitName string
	Description string
}

// GetActiveMemberProfile 返回用户当前有效会员档案（level_code/expire_at/is_permanent）。
// 规则：
// - 未找到或状态无效/已过期：回退 ordinary
// - level_code 为空：回退 ordinary
func (r *UserRepo) GetActiveMemberProfile(ctx context.Context, userID int64) (levelCode string, expireAt *time.Time, isPermanent bool, err error) {
	if r == nil || r.db == nil {
		return "ordinary", nil, false, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if userID <= 0 {
		return "ordinary", nil, false, errorx.NewCodeError(errorx.CodeInvalidParam, "invalid user_id")
	}

	var lc sql.NullString
	var exp sql.NullTime
	var perm sql.NullInt16
	var status sql.NullInt16
	q := `SELECT level_code, expire_at, is_permanent, status FROM user_member WHERE user_id = $1`
	qerr := r.db.QueryRowContext(ctx, q, userID).Scan(&lc, &exp, &perm, &status)
	if qerr != nil {
		if errors.Is(qerr, sql.ErrNoRows) {
			return "ordinary", nil, false, nil
		}
		return "ordinary", nil, false, qerr
	}

	lv := strings.TrimSpace(lc.String)
	if lv == "" {
		lv = "ordinary"
	}
	permanent := perm.Valid && perm.Int16 == 1

	var expPtr *time.Time
	if exp.Valid {
		t := exp.Time
		expPtr = &t
	}

	// status!=1 => treat as ordinary
	if !status.Valid || status.Int16 != 1 {
		return "ordinary", nil, false, nil
	}
	// expired and not permanent => ordinary
	if !permanent && expPtr != nil && !expPtr.IsZero() && expPtr.Before(time.Now()) {
		return "ordinary", nil, false, nil
	}
	return lv, expPtr, permanent, nil
}

// ListBenefitsByLevelCode 查询指定等级的权益（仅返回启用权益）。
func (r *UserRepo) ListBenefitsByLevelCode(ctx context.Context, levelCode string) ([]MemberBenefitRow, error) {
	if r == nil || r.db == nil {
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	lv := strings.TrimSpace(levelCode)
	if lv == "" {
		lv = "ordinary"
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT b.benefit_code, b.benefit_name, COALESCE(b.description,'') AS description
FROM member_level_benefit mlb
JOIN member_benefit b ON b.benefit_code = mlb.benefit_code
WHERE mlb.level_code = $1
  AND b.lifecycle_status = 2
  AND b.status = 1
  AND (b.effect_start_at IS NULL OR b.effect_start_at <= NOW())
  AND (b.effect_end_at IS NULL OR b.effect_end_at >= NOW())
ORDER BY b.id ASC
`, lv)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]MemberBenefitRow, 0)
	for rows.Next() {
		var r0 MemberBenefitRow
		if err := rows.Scan(&r0.BenefitCode, &r0.BenefitName, &r0.Description); err != nil {
			return nil, err
		}
		out = append(out, r0)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// UserMemberRawRow user_member 表原始一行（个人中心展示档案；无行表示从未开通会员行）。
type UserMemberRawRow struct {
	HasRow       bool
	LevelCode    string
	ExpireAt     *time.Time
	IsPermanent  bool
	Status       int16
	RegisterType string
	CreatedAt    time.Time
}

// GetUserMemberRawRow 读取 user_member 原始字段，不做「过期回落 ordinary」改写。
func (r *UserRepo) GetUserMemberRawRow(ctx context.Context, userID int64) (UserMemberRawRow, error) {
	var out UserMemberRawRow
	if r == nil || r.db == nil {
		return out, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if userID <= 0 {
		return out, nil
	}
	var lc sql.NullString
	var exp sql.NullTime
	var perm sql.NullInt16
	var st sql.NullInt16
	var rt sql.NullString
	var created sql.NullTime
	q := `SELECT level_code, expire_at, is_permanent, status, register_type, created_at FROM user_member WHERE user_id = $1`
	err := r.db.QueryRowContext(ctx, q, userID).Scan(&lc, &exp, &perm, &st, &rt, &created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return out, nil
		}
		return out, err
	}
	out.HasRow = true
	out.LevelCode = strings.TrimSpace(lc.String)
	if exp.Valid {
		t := exp.Time
		out.ExpireAt = &t
	}
	out.IsPermanent = perm.Valid && perm.Int16 == 1
	if st.Valid {
		out.Status = st.Int16
	}
	if rt.Valid {
		out.RegisterType = strings.TrimSpace(rt.String)
	}
	if created.Valid {
		out.CreatedAt = created.Time
	}
	return out, nil
}

// GetMemberLevelMeta 查询等级展示名与排序（无行时返回 code 本身、sort=0）。
func (r *UserRepo) GetMemberLevelMeta(ctx context.Context, levelCode string) (name string, sort int32, err error) {
	if r == nil || r.db == nil {
		return "", 0, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	lv := strings.TrimSpace(levelCode)
	if lv == "" {
		lv = "ordinary"
	}
	var n sql.NullString
	var s sql.NullInt32
	qerr := r.db.QueryRowContext(ctx, `SELECT level_name, sort FROM member_level WHERE level_code = $1 AND status = 1`, lv).Scan(&n, &s)
	if qerr != nil {
		if errors.Is(qerr, sql.ErrNoRows) {
			return lv, 0, nil
		}
		return "", 0, qerr
	}
	name = strings.TrimSpace(n.String)
	if name == "" {
		name = lv
	}
	if s.Valid {
		sort = s.Int32
	}
	return name, sort, nil
}

func nullableStr(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

func passwordAlgoForInsert(password *string) interface{} {
	if password == nil || *password == "" {
		return nil
	}
	return passwd.AlgoBcryptConcat
}

// InsertUser 插入 users（注册等场景）；按当前库实际列拼装 SQL，兼容未含 password_algo / register_* 等迁移的旧表。
func InsertUser(ctx context.Context, q RowQuerier, email, mobile, password, salt, nickname, avatar *string, status int, registerIP, registerChannel, inviteCode, deviceID *string) (int64, error) {
	if q == nil {
		return 0, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	cols, err := getUsersTableColumns(ctx, q)
	if err != nil {
		logx.WithContext(ctx).Errorf("InsertUser getUsersTableColumns: %v", err)
		return 0, err
	}
	return insertUserDynamic(ctx, q, cols, email, mobile, password, salt, nickname, avatar, status, registerIP, registerChannel, inviteCode, deviceID)
}

func insertUserDynamic(ctx context.Context, q RowQuerier, cols *usersTableColumns, email, mobile, password, salt, nickname, avatar *string, status int, registerIP, registerChannel, inviteCode, deviceID *string) (int64, error) {
	var fields []string
	var args []interface{}
	add := func(name string, v interface{}) {
		fields = append(fields, name)
		args = append(args, v)
	}
	add("email", nullableStr(email))
	add("mobile", nullableStr(mobile))
	add("password", nullableStr(password))
	add("salt", nullableStr(salt))
	if cols.PasswordAlgo {
		add("password_algo", passwordAlgoForInsert(password))
	}
	add("nickname", nullableStr(nickname))
	if cols.Avatar {
		add("avatar", nullableStr(avatar))
	}
	add("status", status)
	if cols.RegisterIP {
		add("register_ip", nullableStr(registerIP))
	}
	if cols.RegisterChannel {
		add("register_channel", nullableStr(registerChannel))
	}
	if cols.InviteCode {
		add("invite_code", nullableStr(inviteCode))
	}
	if cols.DeviceID {
		add("device_id", nullableStr(deviceID))
	}
	ph := make([]string, len(fields))
	for i := range fields {
		ph[i] = fmt.Sprintf("$%d", i+1)
	}
	query := fmt.Sprintf(
		`INSERT INTO users (%s) VALUES (%s) RETURNING id`,
		strings.Join(fields, ", "),
		strings.Join(ph, ", "))
	var id int64
	err := q.QueryRowContext(ctx, query, args...).Scan(&id)
	if err != nil {
		logx.WithContext(ctx).Errorf("InsertUser: %v", err)
	}
	return id, err
}

// EnsureNotRegistered 注册前检查邮箱/手机未被占用。
func (r *UserRepo) EnsureNotRegistered(ctx context.Context, channel, target string) error {
	if r == nil || r.db == nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	return r.ensureNotRegistered(ctx, r.db, channel, target)
}

func (r *UserRepo) ensureNotRegistered(ctx context.Context, q RowQuerier, channel, target string) error {
	if r == nil || q == nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if err := ctx.Err(); err != nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if err := verifycode.ValidateChannel(channel); err != nil {
		return err
	}

	var registered bool
	var err error
	switch channel {
	case verifycode.ChannelEmail:
		registered, err = EmailRegistered(ctx, q, target)
	case verifycode.ChannelMobile:
		registered, err = MobileRegistered(ctx, q, target)
	default:
		return errorx.NewCodeError(errorx.CodeInvalidParam, "无效的联系方式类型")
	}
	if err != nil {
		logx.WithContext(ctx).Errorf("EnsureNotRegistered channel=%s: %v", channel, err)
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if registered {
		return errorx.NewDefaultError(errorx.CodeUserExists)
	}
	return nil
}

func (r *UserRepo) nicknameInUse(ctx context.Context, q RowQuerier, nickname string) (bool, error) {
	if q == nil || strings.TrimSpace(nickname) == "" {
		return false, nil
	}
	cols, err := getUsersTableColumns(ctx, q)
	if err != nil {
		return false, err
	}
	sql := `SELECT EXISTS(SELECT 1 FROM users WHERE nickname = $1`
	if cols.DeletedAt {
		sql += ` AND deleted_at IS NULL`
	}
	sql += `)`
	var exists bool
	err = q.QueryRowContext(ctx, sql, strings.TrimSpace(nickname)).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// CreateRegisteredUser 注册成功写库（由 channel 决定写入 email 或 mobile）。
func (r *UserRepo) CreateRegisteredUser(ctx context.Context, channel, emailTrimmed, mobileTrimmed, passwordHash, salt string, nickname *string, status int) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	var emailPtr, mobilePtr *string
	switch channel {
	case verifycode.ChannelEmail:
		emailPtr = &emailTrimmed
	case verifycode.ChannelMobile:
		mobilePtr = &mobileTrimmed
	default:
		return 0, errorx.NewCodeError(errorx.CodeInvalidParam, "无效的联系方式类型")
	}
	pwd := passwordHash
	saltVal := salt
	id, err := InsertUser(ctx, r.db, emailPtr, mobilePtr, &pwd, &saltVal, nickname, nil, status, nil, nil, nil, nil)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			return 0, errorx.NewDefaultError(errorx.CodeUserExists)
		}
		logx.WithContext(ctx).Errorf("CreateRegisteredUser InsertUser: %v", err)
		return 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return id, nil
}

// RegisterUserTx 在单事务内完成「未注册校验 + 插入用户」，并写入 register_ip / register_channel。
func (r *UserRepo) RegisterUserTx(ctx context.Context, channel, target, emailTrimmed, mobileTrimmed, passwordHash, salt string, nickname *string, avatar *string, status int, registerIP, registerChannel *string) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	var emailPtr, mobilePtr *string
	switch channel {
	case verifycode.ChannelEmail:
		emailPtr = &emailTrimmed
	case verifycode.ChannelMobile:
		mobilePtr = &mobileTrimmed
	default:
		return 0, errorx.NewCodeError(errorx.CodeInvalidParam, "无效的联系方式类型")
	}

	tx, err := r.BeginTx(ctx)
	if err != nil {
		logx.WithContext(ctx).Errorf("RegisterUserTx BeginTx: %v", err)
		return 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer func() { _ = tx.Rollback() }()

	// 同一注册目标（邮箱/手机）在事务内串行化，降低并发下「校验通过但插入冲突」的窗口。
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1::text))`, target); err != nil {
		logx.WithContext(ctx).Errorf("RegisterUserTx advisory lock: %v", err)
		return 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	if err := r.ensureNotRegistered(ctx, tx, channel, target); err != nil {
		return 0, err
	}

	if nickname != nil {
		if s := strings.TrimSpace(*nickname); s != "" {
			inUse, nerr := r.nicknameInUse(ctx, tx, s)
			if nerr != nil {
				logx.WithContext(ctx).Errorf("RegisterUserTx nicknameInUse: %v", nerr)
				return 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
			}
			if inUse {
				return 0, errorx.NewDefaultError(errorx.CodeNicknameTaken)
			}
		}
	}

	pwd := passwordHash
	saltVal := salt
	id, err := InsertUser(ctx, tx, emailPtr, mobilePtr, &pwd, &saltVal, nickname, avatar, status, registerIP, registerChannel, nil, nil)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			return 0, errorx.NewDefaultError(errorx.CodeUserExists)
		}
		logx.WithContext(ctx).Errorf("RegisterUserTx InsertUser: %v", err)
		return 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if err := tx.Commit(); err != nil {
		logx.WithContext(ctx).Errorf("RegisterUserTx Commit: %v", err)
		return 0, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return id, nil
}

// RegisterOrGetByEmailCodeLogin 邮箱验证码登录：无密码用户；已存在则返回已有 id（created=false）。
func (r *UserRepo) RegisterOrGetByEmailCodeLogin(ctx context.Context, email string, nickname *string, registerIP, registerChannel, inviteCode, deviceID *string) (userID int64, created bool, err error) {
	if r == nil || r.db == nil {
		return 0, false, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if email == "" {
		return 0, false, errorx.NewCodeError(errorx.CodeInvalidParam, "邮箱无效")
	}
	tx, err := r.BeginTx(ctx)
	if err != nil {
		logx.WithContext(ctx).Errorf("RegisterOrGetByEmailCodeLogin BeginTx: %v", err)
		return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer func() { _ = tx.Rollback() }()

	var existingID int64
	qerr := tx.QueryRowContext(ctx,
		`SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL`,
		email,
	).Scan(&existingID)
	if qerr == nil {
		if err := tx.Commit(); err != nil {
			logx.WithContext(ctx).Errorf("RegisterOrGetByEmailCodeLogin Commit: %v", err)
			return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
		}
		return existingID, false, nil
	}
	if qerr != sql.ErrNoRows {
		logx.WithContext(ctx).Errorf("RegisterOrGetByEmailCodeLogin select: %v", qerr)
		return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	emailCopy := email
	av := avatarutil.DefaultAvatarURL(avatarutil.SeedFromTarget(email))
	id, ierr := InsertUser(ctx, tx, &emailCopy, nil, nil, nil, nickname, &av, userconst.UserStatusActive, registerIP, registerChannel, inviteCode, deviceID)
	if ierr != nil {
		var pe *pgconn.PgError
		if errors.As(ierr, &pe) && pe.Code == "23505" {
			if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE email = $1 AND deleted_at IS NULL`, email).Scan(&existingID); err != nil {
				logx.WithContext(ctx).Errorf("RegisterOrGetByEmailCodeLogin after conflict: %v", err)
				return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
			}
			if err := tx.Commit(); err != nil {
				logx.WithContext(ctx).Errorf("RegisterOrGetByEmailCodeLogin Commit: %v", err)
				return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
			}
			return existingID, false, nil
		}
		logx.WithContext(ctx).Errorf("RegisterOrGetByEmailCodeLogin InsertUser: %v", ierr)
		return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if err := tx.Commit(); err != nil {
		logx.WithContext(ctx).Errorf("RegisterOrGetByEmailCodeLogin Commit: %v", err)
		return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return id, true, nil
}

// RegisterOrGetByMobileCodeLogin 手机验证码登录：无密码用户；已存在则返回已有 id（created=false）。
func (r *UserRepo) RegisterOrGetByMobileCodeLogin(ctx context.Context, mobile string, nickname *string, registerIP, registerChannel, inviteCode, deviceID *string) (userID int64, created bool, err error) {
	if r == nil || r.db == nil {
		return 0, false, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if strings.TrimSpace(mobile) == "" {
		return 0, false, errorx.NewCodeError(errorx.CodeInvalidParam, "手机号无效")
	}
	mobile = strings.TrimSpace(mobile)
	tx, err := r.BeginTx(ctx)
	if err != nil {
		logx.WithContext(ctx).Errorf("RegisterOrGetByMobileCodeLogin BeginTx: %v", err)
		return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer func() { _ = tx.Rollback() }()

	var existingID int64
	qerr := tx.QueryRowContext(ctx,
		`SELECT id FROM users WHERE mobile = $1 AND deleted_at IS NULL`,
		mobile,
	).Scan(&existingID)
	if qerr == nil {
		if err := tx.Commit(); err != nil {
			logx.WithContext(ctx).Errorf("RegisterOrGetByMobileCodeLogin Commit: %v", err)
			return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
		}
		return existingID, false, nil
	}
	if qerr != sql.ErrNoRows {
		logx.WithContext(ctx).Errorf("RegisterOrGetByMobileCodeLogin select: %v", qerr)
		return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	mobileCopy := mobile
	av := avatarutil.DefaultAvatarURL(avatarutil.SeedFromTarget(mobile))
	id, ierr := InsertUser(ctx, tx, nil, &mobileCopy, nil, nil, nickname, &av, userconst.UserStatusActive, registerIP, registerChannel, inviteCode, deviceID)
	if ierr != nil {
		var pe *pgconn.PgError
		if errors.As(ierr, &pe) && pe.Code == "23505" {
			if err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE mobile = $1 AND deleted_at IS NULL`, mobile).Scan(&existingID); err != nil {
				logx.WithContext(ctx).Errorf("RegisterOrGetByMobileCodeLogin after conflict: %v", err)
				return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
			}
			if err := tx.Commit(); err != nil {
				logx.WithContext(ctx).Errorf("RegisterOrGetByMobileCodeLogin Commit: %v", err)
				return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
			}
			return existingID, false, nil
		}
		logx.WithContext(ctx).Errorf("RegisterOrGetByMobileCodeLogin InsertUser: %v", ierr)
		return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if err := tx.Commit(); err != nil {
		logx.WithContext(ctx).Errorf("RegisterOrGetByMobileCodeLogin Commit: %v", err)
		return 0, false, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return id, true, nil
}

// FindByEmail 根据邮箱查用户（不含密码）。
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*User, error) {
	if r == nil || r.db == nil || email == "" {
		return nil, nil
	}
	var u User
	err := r.db.QueryRowContext(ctx,
		`SELECT `+sqlUserProfileColumns+` FROM users WHERE email = $1 AND deleted_at IS NULL`,
		email,
	).Scan(scanUserProfileArgs(&u)...)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByMobile 根据手机号查用户。
func (r *UserRepo) FindByMobile(ctx context.Context, mobile string) (*User, error) {
	if r == nil || r.db == nil || mobile == "" {
		return nil, nil
	}
	var u User
	err := r.db.QueryRowContext(ctx,
		`SELECT `+sqlUserProfileColumns+` FROM users WHERE mobile = $1 AND deleted_at IS NULL`,
		mobile,
	).Scan(scanUserProfileArgs(&u)...)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Create 创建用户（密码与盐已处理；OAuth 等无密码场景可传 nil）。
func (r *UserRepo) Create(ctx context.Context, email, mobile, passwordHash, salt, nickname, avatar *string, status int16) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	return InsertUser(ctx, r.db, email, mobile, passwordHash, salt, nickname, avatar, int(status), nil, nil, nil, nil)
}

// FindByAuth 根据第三方 auth_type + auth_id 查绑定用户 ID。
func (r *UserRepo) FindByAuth(ctx context.Context, authType, authId string) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	var userId int64
	err := r.db.QueryRowContext(ctx,
		`SELECT ua.user_id FROM user_auth ua
		 INNER JOIN users u ON u.id = ua.user_id AND u.deleted_at IS NULL
		 WHERE ua.auth_type = $1 AND ua.auth_id = $2`,
		authType, authId,
	).Scan(&userId)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return userId, nil
}

// CreateAuth 绑定第三方账号。
func (r *UserRepo) CreateAuth(ctx context.Context, userId int64, authType, authId, refreshToken string) error {
	if r == nil || r.db == nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_auth (user_id, auth_type, auth_id, refresh_token) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (auth_type, auth_id) DO UPDATE SET user_id = $1, refresh_token = $4`,
		userId, authType, authId, refreshToken,
	)
	return err
}

// FindByID 根据用户 ID 查询（用于 JWT 校验后取基本信息）。
func (r *UserRepo) FindByID(ctx context.Context, id int64) (*User, error) {
	if r == nil || r.db == nil || id <= 0 {
		return nil, nil
	}
	var u User
	err := r.db.QueryRowContext(ctx,
		`SELECT `+sqlUserProfileColumns+` FROM users WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(scanUserProfileArgs(&u)...)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByIDForLogin 根据 ID 查用户（含密码、盐）。
func (r *UserRepo) FindByIDForLogin(ctx context.Context, id int64) (*UserWithPassword, error) {
	if r == nil || r.db == nil || id <= 0 {
		return nil, nil
	}
	cols, err := getUsersTableColumns(ctx, r.db)
	if err != nil {
		return nil, err
	}
	q := userLoginSelectWithPassword(cols) + "WHERE " + userWhereActiveByID(cols)
	var u UserWithPassword
	err = r.db.QueryRowContext(ctx, q, id).Scan(&u.Id, &u.Email, &u.Mobile, &u.Nickname, &u.Avatar, &u.Status, &u.Password, &u.Salt, &u.CancellationCoolingUntil, &u.AccountCancelledAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByEmailForLogin 根据邮箱查（含密码、盐）。
func (r *UserRepo) FindByEmailForLogin(ctx context.Context, email string) (*UserWithPassword, error) {
	if r == nil || r.db == nil || email == "" {
		return nil, nil
	}
	cols, err := getUsersTableColumns(ctx, r.db)
	if err != nil {
		return nil, err
	}
	q := userLoginSelectWithPassword(cols) + "WHERE " + userWhereActiveByField(cols, "email")
	var u UserWithPassword
	err = r.db.QueryRowContext(ctx, q, email).Scan(&u.Id, &u.Email, &u.Mobile, &u.Nickname, &u.Avatar, &u.Status, &u.Password, &u.Salt, &u.CancellationCoolingUntil, &u.AccountCancelledAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByMobileForLogin 根据手机号查（含密码、盐）。
func (r *UserRepo) FindByMobileForLogin(ctx context.Context, mobile string) (*UserWithPassword, error) {
	if r == nil || r.db == nil || mobile == "" {
		return nil, nil
	}
	cols, err := getUsersTableColumns(ctx, r.db)
	if err != nil {
		return nil, err
	}
	q := userLoginSelectWithPassword(cols) + "WHERE " + userWhereActiveByField(cols, "mobile")
	var u UserWithPassword
	err = r.db.QueryRowContext(ctx, q, mobile).Scan(&u.Id, &u.Email, &u.Mobile, &u.Nickname, &u.Avatar, &u.Status, &u.Password, &u.Salt, &u.CancellationCoolingUntil, &u.AccountCancelledAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdateBasicInfo 更新用户基本信息（nickname/avatar 传 nil 表示不修改）。
func (r *UserRepo) UpdateBasicInfo(ctx context.Context, userId int64, nickname, avatar *string) error {
	if r == nil || r.db == nil || userId <= 0 {
		return nil
	}

	var sets []string
	var args []interface{}
	idx := 1

	if nickname != nil {
		sets = append(sets, fmt.Sprintf("nickname = $%d", idx))
		args = append(args, *nickname)
		idx++
	}
	if avatar != nil {
		sets = append(sets, fmt.Sprintf("avatar = $%d", idx))
		args = append(args, *avatar)
		idx++
	}
	if len(sets) == 0 {
		return nil
	}

	args = append(args, userId)
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d AND deleted_at IS NULL", strings.Join(sets, ", "), idx)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func strPtrCopy(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

// UpdateBasicInfoTransactional 事务内 FOR UPDATE 锁定行后更新昵称/头像，返回合并后的用户快照及修改前昵称/头像副本（供审计）；不再额外 FindByID。
func (r *UserRepo) UpdateBasicInfoTransactional(ctx context.Context, userId int64, nickname, avatar *string) (*User, *string, *string, error) {
	if nickname == nil && avatar == nil {
		return nil, nil, nil, nil
	}
	return r.UpdateUserProfileTransactional(ctx, userId, &ProfileUpdate{Nickname: nickname, Avatar: avatar})
}

// UpdateUserProfileTransactional 事务内锁定行后更新资料（昵称/头像及扩展字段，见 migrations 025）。
func (r *UserRepo) UpdateUserProfileTransactional(ctx context.Context, userId int64, p *ProfileUpdate) (*User, *string, *string, error) {
	if r == nil || r.db == nil || userId <= 0 {
		return nil, nil, nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if p == nil || !p.Any() {
		return nil, nil, nil, nil
	}

	tx, err := r.BeginTx(ctx)
	if err != nil {
		logx.WithContext(ctx).Errorf("UpdateUserProfileTransactional BeginTx: %v", err)
		return nil, nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var u User
	err = tx.QueryRowContext(ctx,
		`SELECT `+sqlUserProfileColumns+` FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`,
		userId,
	).Scan(scanUserProfileArgs(&u)...)
	if err == sql.ErrNoRows {
		return nil, nil, nil, nil
	}
	if err != nil {
		return nil, nil, nil, err
	}

	oldNick := strPtrCopy(u.Nickname)
	oldAvatar := strPtrCopy(u.Avatar)

	var sets []string
	var args []interface{}
	idx := 1
	if p.Nickname != nil {
		sets = append(sets, fmt.Sprintf("nickname = $%d", idx))
		args = append(args, *p.Nickname)
		idx++
		v := *p.Nickname
		u.Nickname = &v
	}
	if p.Avatar != nil {
		sets = append(sets, fmt.Sprintf("avatar = $%d", idx))
		args = append(args, *p.Avatar)
		idx++
		v := *p.Avatar
		u.Avatar = &v
	}
	if p.Birthday != nil {
		sets = append(sets, fmt.Sprintf("birthday = $%d", idx))
		args = append(args, *p.Birthday)
		idx++
		u.Birthday = sql.NullTime{Time: *p.Birthday, Valid: true}
	}
	if p.Gender != nil {
		sets = append(sets, fmt.Sprintf("gender = $%d", idx))
		args = append(args, *p.Gender)
		idx++
		u.Gender = sql.NullInt16{Int16: *p.Gender, Valid: true}
	}
	if p.Constellation != nil {
		sets = append(sets, fmt.Sprintf("constellation = $%d", idx))
		if *p.Constellation == "" {
			args = append(args, nil)
			u.Constellation = sql.NullString{}
		} else {
			args = append(args, *p.Constellation)
			u.Constellation = sql.NullString{String: *p.Constellation, Valid: true}
		}
		idx++
	}
	if p.Age != nil {
		sets = append(sets, fmt.Sprintf("age = $%d", idx))
		args = append(args, *p.Age)
		idx++
		u.Age = sql.NullInt16{Int16: *p.Age, Valid: true}
	}
	if p.Signature != nil {
		sets = append(sets, fmt.Sprintf("signature = $%d", idx))
		if *p.Signature == "" {
			args = append(args, nil)
			u.Signature = sql.NullString{}
		} else {
			args = append(args, *p.Signature)
			u.Signature = sql.NullString{String: *p.Signature, Valid: true}
		}
		idx++
	}
	if p.Bio != nil {
		sets = append(sets, fmt.Sprintf("bio = $%d", idx))
		if *p.Bio == "" {
			args = append(args, nil)
			u.Bio = sql.NullString{}
		} else {
			args = append(args, *p.Bio)
			u.Bio = sql.NullString{String: *p.Bio, Valid: true}
		}
		idx++
	}
	if p.BirthdayVisibility != nil {
		sets = append(sets, fmt.Sprintf("birthday_visibility = $%d", idx))
		args = append(args, *p.BirthdayVisibility)
		idx++
		u.BirthdayVisibility = sql.NullInt16{Int16: *p.BirthdayVisibility, Valid: true}
	}
	if p.GenderVisibility != nil {
		sets = append(sets, fmt.Sprintf("gender_visibility = $%d", idx))
		args = append(args, *p.GenderVisibility)
		idx++
		u.GenderVisibility = sql.NullInt16{Int16: *p.GenderVisibility, Valid: true}
	}
	if p.ProfileComplete != nil {
		sets = append(sets, fmt.Sprintf("profile_complete = $%d", idx))
		args = append(args, *p.ProfileComplete)
		idx++
		u.ProfileComplete = *p.ProfileComplete
	}
	if p.ProfileCompleteScore != nil {
		sets = append(sets, fmt.Sprintf("profile_complete_score = $%d", idx))
		args = append(args, *p.ProfileCompleteScore)
		idx++
		u.ProfileCompleteScore = *p.ProfileCompleteScore
	}
	if p.Hobbies != nil {
		sets = append(sets, fmt.Sprintf("hobbies = $%d", idx))
		if *p.Hobbies == "" {
			args = append(args, nil)
			u.Hobbies = sql.NullString{}
		} else {
			args = append(args, *p.Hobbies)
			u.Hobbies = sql.NullString{String: *p.Hobbies, Valid: true}
		}
		idx++
	}
	if p.Location != nil {
		sets = append(sets, fmt.Sprintf("location = $%d", idx))
		if *p.Location == "" {
			args = append(args, nil)
			u.Location = sql.NullString{}
		} else {
			args = append(args, *p.Location)
			u.Location = sql.NullString{String: *p.Location, Valid: true}
		}
		idx++
	}

	args = append(args, userId)
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d AND deleted_at IS NULL", strings.Join(sets, ", "), idx)
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, nil, nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, nil, nil, err
	}
	if n == 0 {
		return nil, nil, nil, sql.ErrNoRows
	}

	if err := tx.Commit(); err != nil {
		logx.WithContext(ctx).Errorf("UpdateUserProfileTransactional Commit: %v", err)
		return nil, nil, nil, err
	}
	return &u, oldNick, oldAvatar, nil
}

// BeginTx 开启数据库事务（用于密码更新等与后续扩展的原子组合）。
func (r *UserRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("user repo unavailable")
	}
	return r.db.BeginTx(ctx, nil)
}

// UpdatePassword 更新用户密码与盐。
func (r *UserRepo) UpdatePassword(ctx context.Context, userId int64, passwordHash, salt string) error {
	if r == nil || r.db == nil || userId <= 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET password = $1, salt = $2, password_algo = $3 WHERE id = $4 AND deleted_at IS NULL`,
		passwordHash, salt, passwd.AlgoBcryptConcat, userId,
	)
	return err
}

// UpdatePasswordTx 在事务内更新用户密码与盐。
func (r *UserRepo) UpdatePasswordTx(ctx context.Context, tx *sql.Tx, userId int64, passwordHash, salt string) error {
	if tx == nil || userId <= 0 {
		return errors.New("invalid tx or user id")
	}
	_, err := tx.ExecContext(ctx,
		`UPDATE users SET password = $1, salt = $2, password_algo = $3 WHERE id = $4 AND deleted_at IS NULL`,
		passwordHash, salt, passwd.AlgoBcryptConcat, userId,
	)
	return err
}

// RecordLoginMeta 更新最近登录时间与 IP（OAuth / 密码登录成功后调用；IP 为空则只更新时间）。
func (r *UserRepo) RecordLoginMeta(ctx context.Context, userId int64, clientIP string) error {
	if r == nil || r.db == nil || userId <= 0 {
		return nil
	}
	var ip interface{}
	if s := strings.TrimSpace(clientIP); s != "" {
		if len(s) > 45 {
			s = s[:45]
		}
		ip = s
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET last_login_at = CURRENT_TIMESTAMP, last_login_ip = $2 WHERE id = $1 AND deleted_at IS NULL`,
		userId, ip,
	)
	return err
}

// BindEmail 绑定邮箱。
func (r *UserRepo) BindEmail(ctx context.Context, userId int64, email string) (int64, error) {
	if r == nil || r.db == nil || userId <= 0 || email == "" {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE users
SET email = $1
WHERE id = $2
  AND (email IS NULL OR email = $1)
  AND NOT EXISTS (SELECT 1 FROM users WHERE email = $1 AND id <> $2 AND deleted_at IS NULL)
`, email, userId)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// BindMobile 绑定手机号。
func (r *UserRepo) BindMobile(ctx context.Context, userId int64, mobile string) (int64, error) {
	if r == nil || r.db == nil || userId <= 0 || mobile == "" {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE users
SET mobile = $1
WHERE id = $2
  AND (mobile IS NULL OR mobile = $1)
  AND NOT EXISTS (SELECT 1 FROM users WHERE mobile = $1 AND id <> $2 AND deleted_at IS NULL)
`, mobile, userId)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// RebindEmail 换绑邮箱。
func (r *UserRepo) RebindEmail(ctx context.Context, userId int64, oldEmail, newEmail string) (int64, error) {
	if r == nil || r.db == nil || userId <= 0 || oldEmail == "" || newEmail == "" {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE users
SET email = $1
WHERE id = $2
  AND email = $3
  AND NOT EXISTS (SELECT 1 FROM users WHERE email = $1 AND id <> $2 AND deleted_at IS NULL)
`, newEmail, userId, oldEmail)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// RebindMobile 换绑手机号。
func (r *UserRepo) RebindMobile(ctx context.Context, userId int64, oldMobile, newMobile string) (int64, error) {
	if r == nil || r.db == nil || userId <= 0 || oldMobile == "" || newMobile == "" {
		return 0, nil
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE users
SET mobile = $1
WHERE id = $2
  AND mobile = $3
  AND NOT EXISTS (SELECT 1 FROM users WHERE mobile = $1 AND id <> $2 AND deleted_at IS NULL)
`, newMobile, userId, oldMobile)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// RebindContactTransactional 事务内 FOR UPDATE 锁定用户行后换绑邮箱或手机，依赖 UPDATE 子查询保证新号未被占用；返回更新后的用户快照（免二次 FindByID）。
func (r *UserRepo) RebindContactTransactional(ctx context.Context, userID int64, channel, oldTarget, newTarget string) (*User, error) {
	if r == nil || r.db == nil || userID <= 0 || oldTarget == "" || newTarget == "" {
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if err := verifycode.ValidateChannel(channel); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	tx, err := r.BeginTx(ctx)
	if err != nil {
		logx.WithContext(ctx).Errorf("RebindContactTransactional BeginTx: %v", err)
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var u User
	err = tx.QueryRowContext(ctx,
		`SELECT `+sqlUserProfileColumns+` FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`,
		userID,
	).Scan(scanUserProfileArgs(&u)...)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}

	if int32(u.Status) != int32(userconst.UserStatusActive) {
		return nil, errorx.NewDefaultError(errorx.CodeUserAccountDisabled)
	}
	if u.AccountCancelledAt.Valid {
		return nil, errorx.NewDefaultError(errorx.CodeAccountCancelled)
	}
	if u.CancellationCoolingUntil.Valid && u.CancellationCoolingUntil.Time.After(time.Now()) {
		return nil, errorx.NewDefaultError(errorx.CodeCancellationCooling)
	}

	switch channel {
	case verifycode.ChannelEmail:
		if u.Email == nil || *u.Email != oldTarget {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "绑定信息已变更，请刷新后重试")
		}
		res, err := tx.ExecContext(ctx, `
UPDATE users
SET email = $1
WHERE id = $2
  AND email = $3
  AND NOT EXISTS (SELECT 1 FROM users WHERE email = $1 AND id <> $2 AND deleted_at IS NULL)
`, newTarget, userID, oldTarget)
		if err != nil {
			return nil, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, errorx.NewDefaultError(errorx.CodeRebindContactConflict)
		}
		nt := newTarget
		u.Email = &nt
	case verifycode.ChannelMobile:
		if u.Mobile == nil || *u.Mobile != oldTarget {
			return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "绑定信息已变更，请刷新后重试")
		}
		res, err := tx.ExecContext(ctx, `
UPDATE users
SET mobile = $1
WHERE id = $2
  AND mobile = $3
  AND NOT EXISTS (SELECT 1 FROM users WHERE mobile = $1 AND id <> $2 AND deleted_at IS NULL)
`, newTarget, userID, oldTarget)
		if err != nil {
			return nil, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, errorx.NewDefaultError(errorx.CodeRebindContactConflict)
		}
		nt := newTarget
		u.Mobile = &nt
	default:
		return nil, errorx.NewDefaultError(errorx.CodeInvalidParam)
	}

	if err := tx.Commit(); err != nil {
		logx.WithContext(ctx).Errorf("RebindContactTransactional Commit: %v", err)
		return nil, err
	}
	return &u, nil
}

// DeleteUserAuth 解绑第三方登录（删除 user_auth 中对应 auth_type 行）。
func (r *UserRepo) DeleteUserAuth(ctx context.Context, userID int64, authType string) error {
	if r == nil || r.db == nil || userID <= 0 || strings.TrimSpace(authType) == "" {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	res, err := r.db.ExecContext(ctx, `DELETE FROM user_auth WHERE user_id = $1 AND auth_type = $2`, userID, strings.TrimSpace(authType))
	if err != nil {
		logx.WithContext(ctx).Errorf("DeleteUserAuth: %v", err)
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if n == 0 {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "未绑定该第三方账号")
	}
	return nil
}
