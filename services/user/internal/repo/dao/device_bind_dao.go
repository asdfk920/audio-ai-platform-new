package dao

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/logger"
)

type deviceTableColumns struct {
	BoundUserID bool
	BoundAt     bool
	BindStatus  bool
	LastActiveAt bool
	UpdatedAt   bool
}

type userProfileTableColumns struct {
	DeviceCount  bool
	LastBindTime bool
	UpdatedAt    bool
}

var (
	deviceColsMu sync.RWMutex
	deviceCols   *deviceTableColumns
	profileColsMu sync.RWMutex
	profileCols   *userProfileTableColumns
)

func getDeviceTableColumns(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}) (*deviceTableColumns, error) {
	deviceColsMu.RLock()
	cached := deviceCols
	deviceColsMu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	deviceColsMu.Lock()
	defer deviceColsMu.Unlock()
	if deviceCols != nil {
		return deviceCols, nil
	}

	rows, err := q.QueryContext(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'device'`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	m := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		m[strings.ToLower(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	col := &deviceTableColumns{
		BoundUserID: mapHas(m, "bound_user_id"),
		BoundAt:     mapHas(m, "bound_at"),
		BindStatus:  mapHas(m, "bind_status"),
		LastActiveAt: mapHas(m, "last_active_at"),
		UpdatedAt:   mapHas(m, "updated_at"),
	}
	deviceCols = col
	return col, nil
}

func getUserProfileTableColumns(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}) (*userProfileTableColumns, error) {
	profileColsMu.RLock()
	cached := profileCols
	profileColsMu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	profileColsMu.Lock()
	defer profileColsMu.Unlock()
	if profileCols != nil {
		return profileCols, nil
	}

	rows, err := q.QueryContext(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'user_profile'`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	m := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		m[strings.ToLower(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	col := &userProfileTableColumns{
		DeviceCount:  mapHas(m, "device_count"),
		LastBindTime: mapHas(m, "last_bind_time"),
		UpdatedAt:    mapHas(m, "updated_at"),
	}
	profileCols = col
	return col, nil
}

// DeviceRow 设备表行模型
type DeviceRow struct {
	ID              int64
	SN              string
	ProductKey      string
	Mac             string
	FirmwareVersion string
	HardwareVersion string
	IP              string
	Status          int16
	OnlineStatus    int16
	Secret          string
	LastActiveAt    *time.Time
	BoundUserID     *int64
	BoundAt         *time.Time
	BindStatus      int16
}

// FindDeviceBySN 根据 SN 查询设备（使用事务）
func FindDeviceBySN(ctx context.Context, tx *sql.Tx, sn string) (*DeviceRow, error) {
	if sn == "" {
		return nil, sql.ErrNoRows
	}

	// #region agent log
	logger.AgentNDJSON("H11", "device_bind_dao.go:FindDeviceBySN", "query device by sn", map[string]any{
		"sn":        sn,
		"snLen":     len(sn),
		"trimmedSn": strings.TrimSpace(sn),
	})
	// #endregion

	var r DeviceRow

	err := tx.QueryRowContext(ctx, `
		SELECT id, sn, product_key, mac, firmware_version, hardware_version, 
		       ip, status, online_status, device_secret
		FROM public.device
		WHERE sn = $1
		LIMIT 1`, sn).Scan(
		&r.ID, &r.SN, &r.ProductKey, &r.Mac, &r.FirmwareVersion,
		&r.HardwareVersion, &r.IP, &r.Status, &r.OnlineStatus,
		&r.Secret,
	)

	if err != nil {
		// #region agent log
		logger.AgentNDJSON("H11", "device_bind_dao.go:FindDeviceBySN", "query device by sn failed", map[string]any{
			"sn":         sn,
			"trimmedSn":  strings.TrimSpace(sn),
			"errType":    logger.ErrType(err),
			"errSnippet": trimBindingDBErr(err),
		})
		// #endregion
		return nil, err
	}

	// #region agent log
	logger.AgentNDJSON("H13", "device_bind_dao.go:FindDeviceBySN", "query device by sn succeeded", map[string]any{
		"sn":           r.SN,
		"deviceId":     r.ID,
		"status":       r.Status,
		"onlineStatus": r.OnlineStatus,
		"bindStatus":   r.BindStatus,
	})
	// #endregion

	return &r, nil
}

// UpdateDeviceBindStatus 更新设备绑定状态
func UpdateDeviceBindStatus(ctx context.Context, tx *sql.Tx, deviceID, userID int64, bindTime time.Time) error {
	// #region agent log
	logger.AgentNDJSON("H15", "device_bind_dao.go:UpdateDeviceBindStatus", "update device bind columns", map[string]any{
		"userId":   userID,
		"deviceId": deviceID,
	})
	// #endregion

	cols, err := getDeviceTableColumns(ctx, tx)
	if err != nil {
		// #region agent log
		logger.AgentNDJSON("H15", "device_bind_dao.go:UpdateDeviceBindStatus", "load device columns failed", map[string]any{
			"deviceId":   deviceID,
			"userId":     userID,
			"errType":    logger.ErrType(err),
			"errSnippet": trimBindingDBErr(err),
		})
		// #endregion
		return err
	}

	sets := make([]string, 0, 4)
	args := make([]interface{}, 0, 3)
	idx := 1
	if cols.BoundUserID {
		sets = append(sets, fmt.Sprintf("bound_user_id = $%d", idx))
		args = append(args, userID)
		idx++
	}
	if cols.BoundAt {
		sets = append(sets, fmt.Sprintf("bound_at = $%d", idx))
		args = append(args, bindTime)
		idx++
	}
	if cols.BindStatus {
		sets = append(sets, "bind_status = 1")
	}
	if cols.UpdatedAt {
		sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	}

	if len(sets) == 0 {
		// #region agent log
		logger.AgentNDJSON("H15", "device_bind_dao.go:UpdateDeviceBindStatus", "skip device bind update no compatible columns", map[string]any{
			"deviceId": deviceID,
			"userId":   userID,
		})
		// #endregion
		return nil
	}

	args = append(args, deviceID)
	query := fmt.Sprintf("UPDATE public.device SET %s WHERE id = $%d", strings.Join(sets, ", "), idx)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		// #region agent log
		logger.AgentNDJSON("H15", "device_bind_dao.go:UpdateDeviceBindStatus", "update device bind columns failed", map[string]any{
			"userId":     userID,
			"deviceId":   deviceID,
			"errType":    logger.ErrType(err),
			"errSnippet": trimBindingDBErr(err),
		})
		// #endregion
	}
	return err
}

// IncrementUserDeviceCount 累加用户绑定设备数
func IncrementUserDeviceCount(ctx context.Context, tx *sql.Tx, userID int64, lastBindTime time.Time) error {
	// #region agent log
	logger.AgentNDJSON("H16", "device_bind_dao.go:IncrementUserDeviceCount", "increment user device count", map[string]any{
		"userId": userID,
	})
	// #endregion

	cols, err := getUserProfileTableColumns(ctx, tx)
	if err != nil {
		// #region agent log
		logger.AgentNDJSON("H16", "device_bind_dao.go:IncrementUserDeviceCount", "load user_profile columns failed", map[string]any{
			"userId":     userID,
			"errType":    logger.ErrType(err),
			"errSnippet": trimBindingDBErr(err),
		})
		// #endregion
		return err
	}

	sets := make([]string, 0, 3)
	args := make([]interface{}, 0, 2)
	idx := 1
	if cols.DeviceCount {
		sets = append(sets, "device_count = COALESCE(device_count, 0) + 1")
	}
	if cols.LastBindTime {
		sets = append(sets, fmt.Sprintf("last_bind_time = $%d", idx))
		args = append(args, lastBindTime)
		idx++
	}
	if cols.UpdatedAt {
		sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	}

	if len(sets) == 0 {
		// #region agent log
		logger.AgentNDJSON("H16", "device_bind_dao.go:IncrementUserDeviceCount", "skip user_profile update no compatible columns", map[string]any{
			"userId": userID,
		})
		// #endregion
		return nil
	}

	args = append(args, userID)
	query := fmt.Sprintf("UPDATE public.user_profile SET %s WHERE user_id = $%d", strings.Join(sets, ", "), idx)
	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		// #region agent log
		logger.AgentNDJSON("H16", "device_bind_dao.go:IncrementUserDeviceCount", "increment user device count failed", map[string]any{
			"userId":     userID,
			"errType":    logger.ErrType(err),
			"errSnippet": trimBindingDBErr(err),
		})
		// #endregion
	}
	return err
}

// FindUserByID 根据 ID 查询用户（使用事务）
func FindUserByID(ctx context.Context, tx *sql.Tx, userID int64) (*UserRow, error) {
	if userID <= 0 {
		return nil, sql.ErrNoRows
	}

	// #region agent log
	logger.AgentNDJSON("H6", "device_bind_dao.go:FindUserByID", "query binding user row", map[string]any{
		"userId": userID,
		"table":  "public.users",
	})
	// #endregion

	var r UserRow
	err := tx.QueryRowContext(ctx, `
		SELECT id, email, mobile, COALESCE(nickname, ''), COALESCE(avatar, ''), status, created_at
		FROM public.users
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1`, userID).Scan(
		&r.ID, &r.Email, &r.Mobile, &r.Nickname, &r.Avatar,
		&r.Status, &r.CreatedAt,
	)

	if err != nil {
		// #region agent log
		logger.AgentNDJSON("H6", "device_bind_dao.go:FindUserByID", "query binding user failed", map[string]any{
			"userId":     userID,
			"table":      "public.users",
			"errType":    logger.ErrType(err),
			"errSnippet": trimBindingDBErr(err),
		})
		// #endregion
		return nil, err
	}

	return &r, nil
}

func trimBindingDBErr(err error) string {
	if err == nil {
		return ""
	}
	s := fmt.Sprint(err)
	if len(s) > 220 {
		return s[:220]
	}
	return s
}

// UserRow 用户表行模型
type UserRow struct {
	ID        int64
	Email     sql.NullString
	Mobile    sql.NullString
	Nickname  string
	Avatar    string
	Status    int16
	CreatedAt time.Time
}

// CountUserBinds 统计用户当前绑定中的设备数（使用事务）
func CountUserBinds(ctx context.Context, tx *sql.Tx, userID int64) (int64, error) {
	var count int64
	err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM public.user_device_bind 
		WHERE user_id = $1 AND status = 1
	`, userID).Scan(&count)
	return count, err
}

// InsertBind 插入绑定记录（使用事务）
func InsertBind(ctx context.Context, tx *sql.Tx, userID, deviceID int64, sn, deviceName, deviceModel, systemVersion, operator string) error {
	alias := deviceName
	if len([]rune(alias)) > 32 {
		alias = string([]rune(deviceName)[:32])
	}
	// #region agent log
	logger.AgentNDJSON("H14", "device_bind_dao.go:InsertBind", "insert bind relation", map[string]any{
		"userId":   userID,
		"deviceId": deviceID,
		"sn":       sn,
		"alias":    alias,
	})
	// #endregion
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.user_device_bind
		  (user_id, device_id, sn, alias, is_default, bind_type, status, bound_at)
		VALUES ($1, $2, $3, $4, 0, 1, 1, CURRENT_TIMESTAMP)
	`, userID, deviceID, sn, alias)
	if err != nil {
		// #region agent log
		logger.AgentNDJSON("H14", "device_bind_dao.go:InsertBind", "insert bind relation failed", map[string]any{
			"userId":     userID,
			"deviceId":   deviceID,
			"sn":         sn,
			"errType":    logger.ErrType(err),
			"errSnippet": trimBindingDBErr(err),
		})
		// #endregion
	}
	return err
}

// InsertBindLog 插入绑定操作日志（使用事务）
func InsertBindLog(ctx context.Context, tx *sql.Tx, userID, deviceID int64, sn, operator, action string, actionTime time.Time) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.user_device_bind_log
		  (user_id, device_id, sn, operator, action, action_time)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, deviceID, sn, operator, action, actionTime)
	return err
}

// FindBindByUserAndDevice 查找用户与设备的绑定记录（含历史解绑记录）
func FindBindByUserAndDevice(ctx context.Context, tx *sql.Tx, userID, deviceID int64) (*UserDeviceBindRow, error) {
	var row UserDeviceBindRow
	err := tx.QueryRowContext(ctx, `
		SELECT id, user_id, device_id, sn, COALESCE(alias,''),
		       status, bound_at, unbound_at
		FROM public.user_device_bind
		WHERE user_id = $1 AND device_id = $2
		LIMIT 1
	`, userID, deviceID).Scan(
		&row.ID, &row.UserID, &row.DeviceID, &row.SN, &row.Alias,
		&row.Status, &row.BoundAt, &row.UnboundAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.DeviceName = row.Alias
	return &row, nil
}

// ReactivateBind 复用历史解绑记录恢复绑定
func ReactivateBind(ctx context.Context, tx *sql.Tx, userID, deviceID int64, sn, deviceName string, bindTime time.Time) error {
	alias := deviceName
	if len([]rune(alias)) > 32 {
		alias = string([]rune(deviceName)[:32])
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE public.user_device_bind
		SET sn = $1,
		    alias = $2,
		    status = 1,
		    bound_at = $3,
		    unbound_at = NULL,
		    updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $4 AND device_id = $5
	`, sn, alias, bindTime, userID, deviceID)
	return err
}

// FindActiveBindByDeviceID 查找当前绑定中（status=1）的记录（使用事务）
func FindActiveBindByDeviceID(ctx context.Context, tx *sql.Tx, deviceID int64) (*UserDeviceBindRow, error) {
	var row UserDeviceBindRow
	err := tx.QueryRowContext(ctx, `
		SELECT id, user_id, device_id, sn, COALESCE(alias,''),
		       status, bound_at, unbound_at
		FROM public.user_device_bind
		WHERE device_id = $1 AND status = 1
		LIMIT 1
	`, deviceID).Scan(
		&row.ID, &row.UserID, &row.DeviceID, &row.SN, &row.Alias,
		&row.Status, &row.BoundAt, &row.UnboundAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.DeviceName = row.Alias
	return &row, nil
}

func buildDeviceOptionalSelects(cols *deviceTableColumns, tableAlias string) (lastActiveAt, boundUserID, boundAt, bindStatus string) {
	lastActiveAt = "NULL::timestamp AS last_active_at"
	if cols.LastActiveAt {
		lastActiveAt = fmt.Sprintf("%s.last_active_at", tableAlias)
	}

	boundUserID = "NULL::bigint AS bound_user_id"
	if cols.BoundUserID {
		boundUserID = fmt.Sprintf("%s.bound_user_id", tableAlias)
	}

	boundAt = "NULL::timestamp AS bound_at"
	if cols.BoundAt {
		boundAt = fmt.Sprintf("%s.bound_at", tableAlias)
	}

	bindStatus = "0::smallint AS bind_status"
	if cols.BindStatus {
		bindStatus = fmt.Sprintf("%s.bind_status", tableAlias)
	}

	return lastActiveAt, boundUserID, boundAt, bindStatus
}

// MarkBindAsUnbound 将绑定关系标记为已解绑
func MarkBindAsUnbound(ctx context.Context, tx *sql.Tx, userID, deviceID int64, unbindTime time.Time) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE public.user_device_bind
		SET status = 0,
		    unbound_at = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $2 AND device_id = $3 AND status = 1
	`, unbindTime, userID, deviceID)
	return err
}

// ClearDeviceBindUser 清空设备表中的绑定用户信息
func ClearDeviceBindUser(ctx context.Context, tx *sql.Tx, deviceID int64, unbindTime time.Time) error {
	cols, err := getDeviceTableColumns(ctx, tx)
	if err != nil {
		return err
	}

	sets := make([]string, 0, 4)
	if cols.BoundUserID {
		sets = append(sets, "bound_user_id = NULL")
	}
	if cols.BoundAt {
		sets = append(sets, "bound_at = NULL")
	}
	if cols.BindStatus {
		sets = append(sets, "bind_status = 0")
	}
	if cols.UpdatedAt {
		sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	}
	if len(sets) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE public.device SET %s WHERE id = $1", strings.Join(sets, ", "))
	_, err = tx.ExecContext(ctx, query, deviceID)
	return err
}

// DecrementUserDeviceCount 减少用户已绑定设备数
func DecrementUserDeviceCount(ctx context.Context, tx *sql.Tx, userID int64, unbindTime time.Time) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'user_profile'`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	cols := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		cols[strings.ToLower(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(cols) == 0 {
		return nil
	}

	sets := make([]string, 0, 3)
	args := make([]interface{}, 0, 2)
	argIndex := 1
	if mapHas(cols, "device_count") {
		sets = append(sets, "device_count = GREATEST(COALESCE(device_count, 0) - 1, 0)")
	}
	if mapHas(cols, "last_unbind_time") {
		sets = append(sets, fmt.Sprintf("last_unbind_time = $%d", argIndex))
		args = append(args, unbindTime)
		argIndex++
	}
	if mapHas(cols, "updated_at") {
		sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	}
	if len(sets) == 0 {
		return nil
	}

	args = append(args, userID)
	query := fmt.Sprintf("UPDATE public.user_profile SET %s WHERE user_id = $%d", strings.Join(sets, ", "), argIndex)
	_, err = tx.ExecContext(ctx, query, args...)
	return err
}

// FindUserBinds 查询用户绑定关系列表（支持条件筛选和分页）
func FindUserBinds(ctx context.Context, tx *sql.Tx, userID int64, deviceSn, deviceName, deviceModel string, offset, limit int) ([]*UserDeviceBindRow, int64, error) {
	cols, err := getDeviceTableColumns(ctx, tx)
	if err != nil {
		return nil, 0, err
	}
	lastActiveAtExpr, boundUserIDExpr, boundAtExpr, bindStatusExpr := buildDeviceOptionalSelects(cols, "d")

	// 构建查询条件
	whereClause := "WHERE ub.user_id = $1 AND ub.status = 1"
	args := []interface{}{userID}
	argIndex := 2

	// 动态添加筛选条件
	if deviceSn != "" {
		whereClause += fmt.Sprintf(" AND d.sn ILIKE $%d", argIndex)
		args = append(args, "%"+toLower(deviceSn)+"%")
		argIndex++
	}
	if deviceName != "" {
		whereClause += fmt.Sprintf(" AND ub.alias ILIKE $%d", argIndex)
		args = append(args, "%"+toLower(deviceName)+"%")
		argIndex++
	}
	if deviceModel != "" {
		whereClause += fmt.Sprintf(" AND d.product_key ILIKE $%d", argIndex)
		args = append(args, "%"+toLower(deviceModel)+"%")
		argIndex++
	}

	// 查询总数
	var total int64
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM public.user_device_bind ub
		JOIN public.device d ON ub.device_id = d.id
		%s
	`, whereClause)
	err = tx.QueryRowContext(ctx, countSQL, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 查询绑定列表
	querySQL := fmt.Sprintf(`
		SELECT ub.id, ub.user_id, ub.device_id, ub.sn, COALESCE(ub.alias,''),
		       ub.status, ub.bound_at, ub.unbound_at,
		       d.sn, d.product_key, d.mac, d.firmware_version, d.hardware_version,
		       d.ip, d.status, d.online_status, d.device_secret, %s,
		       %s, %s, %s
		FROM public.user_device_bind ub
		JOIN public.device d ON ub.device_id = d.id
		%s
		ORDER BY ub.bound_at DESC
		LIMIT $%d OFFSET $%d
	`, lastActiveAtExpr, boundUserIDExpr, boundAtExpr, bindStatusExpr, whereClause, argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := tx.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var binds []*UserDeviceBindRow
	for rows.Next() {
		var bind UserDeviceBindRow
		var device DeviceRow
		var lastActiveAt, boundAt sql.NullTime
		var boundUserID sql.NullInt64

		err := rows.Scan(
			&bind.ID, &bind.UserID, &bind.DeviceID, &bind.SN, &bind.Alias,
			&bind.Status, &bind.BoundAt, &bind.UnboundAt,
			&device.SN, &device.ProductKey, &device.Mac, &device.FirmwareVersion,
			&device.HardwareVersion, &device.IP, &device.Status, &device.OnlineStatus,
			&device.Secret, &lastActiveAt, &boundUserID, &boundAt, &device.BindStatus,
		)
		if err != nil {
			return nil, 0, err
		}

		bind.DeviceName = bind.Alias
		bind.DeviceModel = device.ProductKey
		bind.SystemVersion = device.FirmwareVersion

		if lastActiveAt.Valid {
			t := lastActiveAt.Time
			device.LastActiveAt = &t
		}

		binds = append(binds, &bind)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return binds, total, nil
}

// FindDevicesBySNs 批量查询设备信息
func FindDevicesBySNs(ctx context.Context, tx *sql.Tx, sns []string) ([]*DeviceRow, error) {
	if len(sns) == 0 {
		return []*DeviceRow{}, nil
	}
	cols, err := getDeviceTableColumns(ctx, tx)
	if err != nil {
		return nil, err
	}
	lastActiveAtExpr, boundUserIDExpr, boundAtExpr, bindStatusExpr := buildDeviceOptionalSelects(cols, "device")

	// 构建 IN 查询
	placeholders := make([]string, len(sns))
	args := make([]interface{}, len(sns))
	for i, sn := range sns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = sn
	}

	query := fmt.Sprintf(`
		SELECT id, sn, product_key, mac, firmware_version, hardware_version,
		       ip, status, online_status, device_secret, %s,
		       %s, %s, %s
		FROM public.device
		WHERE sn IN (%s)
	`, lastActiveAtExpr, boundUserIDExpr, boundAtExpr, bindStatusExpr, strings.Join(placeholders, ","))

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*DeviceRow
	for rows.Next() {
		var device DeviceRow
		var lastActiveAt, boundAt sql.NullTime
		var boundUserID sql.NullInt64

		err := rows.Scan(
			&device.ID, &device.SN, &device.ProductKey, &device.Mac,
			&device.FirmwareVersion, &device.HardwareVersion, &device.IP,
			&device.Status, &device.OnlineStatus, &device.Secret,
			&lastActiveAt, &boundUserID, &boundAt, &device.BindStatus,
		)
		if err != nil {
			return nil, err
		}

		if lastActiveAt.Valid {
			t := lastActiveAt.Time
			device.LastActiveAt = &t
		}

		if boundUserID.Valid {
			id := boundUserID.Int64
			device.BoundUserID = &id
		}

		if boundAt.Valid {
			t := boundAt.Time
			device.BoundAt = &t
		}

		devices = append(devices, &device)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return devices, nil
}

func toLower(s string) string {
	return strings.ToLower(s)
}
