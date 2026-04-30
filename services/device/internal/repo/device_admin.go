package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// AdminListFilter 管理端列表筛选（与 go-admin platform-device 对齐）
type AdminListFilter struct {
	SnExact      bool
	Sn           string
	UserID       int64
	UserQuery    string
	Status       *int16
	OnlineStatus *int16
	ProductKey   string
	FirmwareVer  string
	BindStatus   *int16
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
	SortBy       string
	SortOrder    string
}

// AdminListRow 列表一行
type AdminListRow struct {
	ID              int64
	Sn              string
	Model           string
	ProductKey      string
	FirmwareVersion string
	HardwareVersion string
	Mac             string
	Ip              string
	OnlineStatus    int16
	DisplayOnline   int16
	Status          int16
	UserID          int64
	UserNickname    string
	UserMobile      string
	BindTime        sql.NullTime
	LastActiveAt    sql.NullTime
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// DeviceStats 看板
type DeviceStats struct {
	Total       int64
	Online      int64
	Offline     int64
	Unbound     int64
	TodayNew    int64
	TodayActive int64
}

// rebindPG 将 SQL 中的 ? 转为 PostgreSQL 占位符 $1、$2…（与 jackc/pgx stdlib 一致）。
func rebindPG(q string) string {
	i := 1
	s := q
	for strings.Contains(s, "?") {
		s = strings.Replace(s, "?", fmt.Sprintf("$%d", i), 1)
		i++
	}
	return s
}

func maskMobile(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r := []rune(s)
	if len(r) <= 4 {
		return "****"
	}
	return "****" + string(r[len(r)-4:])
}

func displayOnline(lastActive sql.NullTime, dbOnline int16) int16 {
	if lastActive.Valid && time.Since(lastActive.Time) <= 5*time.Minute {
		return 1
	}
	if dbOnline == 1 && !lastActive.Valid {
		return 1
	}
	return 0
}

func listOrderSQL(sortBy, sortOrder string) string {
	col := "d.id"
	switch strings.ToLower(strings.TrimSpace(sortBy)) {
	case "created_at":
		col = "d.created_at"
	case "last_active_at":
		col = "d.last_active_at"
	case "bind_time":
		col = "udb.bound_at"
	case "id":
		col = "d.id"
	}
	ord := "DESC"
	if strings.EqualFold(strings.TrimSpace(sortOrder), "asc") {
		ord = "ASC"
	}
	return col + " " + ord
}

func buildAdminListWhere(f AdminListFilter) (string, []any) {
	var conds []string
	var args []any

	if s := strings.TrimSpace(f.Sn); s != "" {
		if f.SnExact {
			conds = append(conds, "d.sn = ?")
			args = append(args, s)
		} else {
			conds = append(conds, "d.sn ILIKE ?")
			args = append(args, "%"+s+"%")
		}
	}
	if f.UserID > 0 {
		conds = append(conds, "udb.user_id = ?")
		args = append(args, f.UserID)
	}
	if kw := strings.TrimSpace(f.UserQuery); kw != "" {
		like := "%" + kw + "%"
		conds = append(conds, "(u.nickname ILIKE ? OR u.mobile ILIKE ? OR CAST(u.id AS TEXT) = ?)")
		args = append(args, like, like, kw)
	}
	if f.Status != nil {
		conds = append(conds, "d.status = ?")
		args = append(args, *f.Status)
	}
	if f.OnlineStatus != nil {
		conds = append(conds, "d.online_status = ?")
		args = append(args, *f.OnlineStatus)
	}
	if pk := strings.TrimSpace(f.ProductKey); pk != "" {
		conds = append(conds, "d.product_key = ?")
		args = append(args, pk)
	}
	if fw := strings.TrimSpace(f.FirmwareVer); fw != "" {
		conds = append(conds, "d.firmware_version ILIKE ?")
		args = append(args, "%"+fw+"%")
	}
	if f.BindStatus != nil {
		if *f.BindStatus == 1 {
			conds = append(conds, "udb.user_id IS NOT NULL")
		} else {
			conds = append(conds, "udb.user_id IS NULL")
		}
	}
	if f.CreatedFrom != nil {
		conds = append(conds, "d.created_at >= ?")
		args = append(args, *f.CreatedFrom)
	}
	if f.CreatedTo != nil {
		conds = append(conds, "d.created_at < ?")
		args = append(args, f.CreatedTo.Add(24*time.Hour))
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	return where, args
}

// QueryDeviceStats 全库统计
func QueryDeviceStats(ctx context.Context, db *sql.DB) (*DeviceStats, error) {
	var out DeviceStats
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM device`).Scan(&out.Total); err != nil {
		return nil, err
	}
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM device
WHERE online_status = 1 OR (last_active_at IS NOT NULL AND last_active_at > NOW() - INTERVAL '5 minutes')`).Scan(&out.Online); err != nil {
		return nil, err
	}
	if out.Total >= out.Online {
		out.Offline = out.Total - out.Online
	}
	if err := db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM device d
WHERE NOT EXISTS (SELECT 1 FROM user_device_bind udb WHERE udb.device_id = d.id AND udb.status = 1)`).Scan(&out.Unbound); err != nil {
		return nil, err
	}
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM device WHERE created_at::date = CURRENT_DATE`).Scan(&out.TodayNew)
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM device WHERE last_active_at::date = CURRENT_DATE`).Scan(&out.TodayActive)
	return &out, nil
}

// ListProductKeys DISTINCT product_key
func ListProductKeys(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT product_key FROM device WHERE TRIM(product_key) <> '' ORDER BY product_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// CountAdminDevices 列表总数（DISTINCT 设备 id）
func CountAdminDevices(ctx context.Context, db *sql.DB, f AdminListFilter) (int64, error) {
	where, args := buildAdminListWhere(f)
	q := fmt.Sprintf(`SELECT COUNT(*) FROM (
  SELECT DISTINCT d.id FROM device d
  LEFT JOIN user_device_bind udb ON udb.device_id = d.id AND udb.status = 1
  LEFT JOIN users u ON u.id = udb.user_id
  %s
) t`, where)
	q = rebindPG(q)
	var total int64
	if err := db.QueryRowContext(ctx, q, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// QueryAdminDeviceList 分页列表
func QueryAdminDeviceList(ctx context.Context, db *sql.DB, page, pageSize int, f AdminListFilter) ([]AdminListRow, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	where, args := buildAdminListWhere(f)
	order := listOrderSQL(f.SortBy, f.SortOrder)
	offset := (page - 1) * pageSize

	q := fmt.Sprintf(`
SELECT d.id, d.sn, d.model, d.product_key, d.firmware_version, d.hardware_version,
  d.mac, d.ip, d.online_status, d.status, d.created_at, d.updated_at, d.last_active_at,
  udb.user_id AS bind_user_id, u.nickname AS user_nickname, u.mobile AS user_mobile_raw, udb.bound_at AS bind_time
FROM device d
LEFT JOIN user_device_bind udb ON udb.device_id = d.id AND udb.status = 1
LEFT JOIN users u ON u.id = udb.user_id
%s
ORDER BY %s
LIMIT ? OFFSET ?`, where, order)
	q = rebindPG(q)

	args = append(args, pageSize, offset)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AdminListRow
	for rows.Next() {
		var r AdminListRow
		var bindUID sql.NullInt64
		var nick, mob sql.NullString
		var bindTime sql.NullTime
		if err := rows.Scan(
			&r.ID, &r.Sn, &r.Model, &r.ProductKey, &r.FirmwareVersion, &r.HardwareVersion,
			&r.Mac, &r.Ip, &r.OnlineStatus, &r.Status, &r.CreatedAt, &r.UpdatedAt, &r.LastActiveAt,
			&bindUID, &nick, &mob, &bindTime,
		); err != nil {
			return nil, err
		}
		if bindUID.Valid {
			r.UserID = bindUID.Int64
		}
		if nick.Valid {
			r.UserNickname = nick.String
		}
		if mob.Valid {
			r.UserMobile = maskMobile(mob.String)
		}
		r.BindTime = bindTime
		r.DisplayOnline = displayOnline(r.LastActiveAt, r.OnlineStatus)
		out = append(out, r)
	}
	return out, rows.Err()
}
