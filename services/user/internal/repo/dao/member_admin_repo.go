package dao

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// MemberAdminFilter 后台会员列表筛选
type MemberAdminFilter struct {
	UserID          int64
	NicknameSub     string
	MobileSub       string
	LevelCode       string
	MemberStatus    *int16 // 1=有效 2=已过期 3=未开通；nil=全部
	OpenedFrom      *time.Time
	OpenedTo        *time.Time
	ExpireFrom      *time.Time
	ExpireTo        *time.Time
}

// MemberAdminRow 列表一行（数据库扫描）
type MemberAdminRow struct {
	UserID       int64
	Nickname     sql.NullString
	Mobile       sql.NullString
	LevelCode    sql.NullString
	LevelName    sql.NullString
	UMCreated    sql.NullTime
	ExpireAt     sql.NullTime
	IsPermanent  sql.NullInt16
	MemberStatus int16
}

const sqlMemberStatusExpr = `(CASE
  WHEN um.id IS NULL THEN 3
  WHEN COALESCE(um.is_permanent, 0) = 1 THEN 1
  WHEN COALESCE(um.expire_at, um.expired_at) IS NULL THEN 3
  WHEN COALESCE(um.expire_at, um.expired_at) < NOW() THEN 2
  ELSE 1
END)`

// IsBackofficeAdmin 是否后台管理员（super_admin / admin）
func IsBackofficeAdmin(ctx context.Context, db *sql.DB, userID int64) (bool, error) {
	if userID <= 0 {
		return false, nil
	}
	var ok bool
	err := db.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1 FROM user_role_rel ur
  INNER JOIN roles r ON r.id = ur.role_id
  WHERE ur.user_id = $1 AND r.name IN ('super_admin', 'admin')
)`, userID).Scan(&ok)
	return ok, err
}

func buildMemberAdminWhere(f MemberAdminFilter) (string, []any) {
	var conds []string
	var args []any
	arg := func() int {
		return len(args) + 1
	}
	conds = append(conds, "u.deleted_at IS NULL")
	if f.UserID > 0 {
		conds = append(conds, fmt.Sprintf("u.id = $%d", arg()))
		args = append(args, f.UserID)
	}
	if s := strings.TrimSpace(f.NicknameSub); s != "" {
		conds = append(conds, fmt.Sprintf("u.nickname ILIKE $%d", arg()))
		args = append(args, "%"+s+"%")
	}
	if s := strings.TrimSpace(f.MobileSub); s != "" {
		conds = append(conds, fmt.Sprintf("u.mobile ILIKE $%d", arg()))
		args = append(args, "%"+s+"%")
	}
	if s := strings.TrimSpace(f.LevelCode); s != "" {
		conds = append(conds, fmt.Sprintf("um.level_code = $%d", arg()))
		args = append(args, s)
	}
	if f.OpenedFrom != nil {
		conds = append(conds, fmt.Sprintf("um.created_at >= $%d", arg()))
		args = append(args, *f.OpenedFrom)
	}
	if f.OpenedTo != nil {
		conds = append(conds, fmt.Sprintf("um.created_at < $%d", arg()))
		args = append(args, f.OpenedTo.Add(24*time.Hour))
	}
	if f.ExpireFrom != nil {
		conds = append(conds, fmt.Sprintf("COALESCE(um.expire_at, um.expired_at) >= $%d", arg()))
		args = append(args, *f.ExpireFrom)
	}
	if f.ExpireTo != nil {
		conds = append(conds, fmt.Sprintf("COALESCE(um.expire_at, um.expired_at) < $%d", arg()))
		args = append(args, f.ExpireTo.Add(24*time.Hour))
	}
	if f.MemberStatus != nil {
		conds = append(conds, fmt.Sprintf("%s = $%d", sqlMemberStatusExpr, arg()))
		args = append(args, *f.MemberStatus)
	}
	where := "WHERE " + strings.Join(conds, " AND ")
	return where, args
}

// CountMemberAdmin 符合条件的用户数
func CountMemberAdmin(ctx context.Context, db *sql.DB, f MemberAdminFilter) (int64, error) {
	where, args := buildMemberAdminWhere(f)
	q := fmt.Sprintf(`
SELECT COUNT(*) FROM users u
LEFT JOIN user_member um ON um.user_id = u.id
%s`, where)
	var total int64
	err := db.QueryRowContext(ctx, q, args...).Scan(&total)
	return total, err
}

// ListMemberAdmin 分页查询
func ListMemberAdmin(ctx context.Context, db *sql.DB, f MemberAdminFilter, page, pageSize int) ([]MemberAdminRow, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	where, args := buildMemberAdminWhere(f)
	offset := (page - 1) * pageSize
	limitArg := len(args) + 1
	offArg := len(args) + 2
	args = append(args, pageSize, offset)

	q := fmt.Sprintf(`
SELECT
  u.id,
  u.nickname,
  u.mobile,
  um.level_code,
  ml.level_name,
  um.created_at,
  COALESCE(um.expire_at, um.expired_at) AS exp_at,
  um.is_permanent,
  %s AS st
FROM users u
LEFT JOIN user_member um ON um.user_id = u.id
LEFT JOIN member_level ml ON ml.level_code = um.level_code
%s
ORDER BY u.id DESC
LIMIT $%d OFFSET $%d`, sqlMemberStatusExpr, where, limitArg, offArg)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MemberAdminRow
	for rows.Next() {
		var r MemberAdminRow
		var nick, mob, lc, ln sql.NullString
		var umCreated sql.NullTime
		var expAt sql.NullTime
		var perm sql.NullInt16
		if err := rows.Scan(&r.UserID, &nick, &mob, &lc, &ln, &umCreated, &expAt, &perm, &r.MemberStatus); err != nil {
			return nil, err
		}
		r.Nickname = nick
		r.Mobile = mob
		r.LevelCode = lc
		r.LevelName = ln
		r.UMCreated = umCreated
		r.ExpireAt = expAt
		r.IsPermanent = perm
		out = append(out, r)
	}
	return out, rows.Err()
}
