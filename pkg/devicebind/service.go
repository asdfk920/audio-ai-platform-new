package devicebind

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
)

type Options struct {
	MaxDeviceBinds int
}

type Result struct {
	UserID     int64
	DeviceID   int64
	DeviceSN   string
	DeviceName string
	BindTime   time.Time
}

type userRow struct {
	ID     int64
	Status int16
}

type deviceRow struct {
	ID     int64
	SN     string
	Status int16
}

type userDeviceBindRow struct {
	ID         int64
	UserID     int64
	DeviceID   int64
	SN         string
	Alias      string
	Status     int16
	BoundAt    time.Time
	UnboundAt  sql.NullTime
	DeviceName string
}

type deviceTableColumns struct {
	BoundUserID bool
	BoundAt     bool
	BindStatus  bool
	UpdatedAt   bool
}

type userProfileTableColumns struct {
	DeviceCount  bool
	LastBindTime bool
	UpdatedAt    bool
}

var (
	deviceColsMu  sync.RWMutex
	deviceCols    *deviceTableColumns
	profileColsMu sync.RWMutex
	profileCols   *userProfileTableColumns
)

func BindUserDevice(ctx context.Context, db *sql.DB, userID int64, deviceSN string, opts Options) (*Result, error) {
	if userID <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}

	snNorm := normalizeSN(deviceSN)
	if snNorm == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "设备序列号不能为空")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙，请稍后重试")
	}
	defer func() { _ = tx.Rollback() }()

	user, err := findUserByID(ctx, tx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errorx.NewCodeError(errorx.CodeUserNotFound, "用户不存在")
		}
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询用户失败")
	}
	if user.Status != 1 {
		if user.Status == 2 {
			return nil, errorx.NewCodeError(errorx.CodeUserAccountDisabled, "用户账号已被禁用")
		}
		if user.Status == 3 {
			return nil, errorx.NewCodeError(errorx.CodeUserAccountDisabled, "用户账号已被封禁")
		}
		return nil, errorx.NewCodeError(errorx.CodeUserAccountDisabled, "用户账号状态异常")
	}

	device, err := findDeviceBySN(ctx, tx, snNorm)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errorx.NewCodeError(errorx.CodeDeviceNotFound, "设备不存在")
		}
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询设备失败")
	}
	if device.Status != 1 {
		if device.Status == 2 {
			return nil, errorx.NewCodeError(errorx.CodeDeviceDisabled, "设备已被禁用")
		}
		if device.Status == 3 {
			return nil, errorx.NewCodeError(errorx.CodeDeviceInactive, "设备未激活，请先激活设备")
		}
		return nil, errorx.NewCodeError(errorx.CodeDeviceDisabled, "设备状态异常")
	}

	activeBind, err := findActiveBindByDeviceID(ctx, tx, device.ID)
	if err != nil {
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询绑定状态失败")
	}
	if activeBind != nil && activeBind.UserID != userID {
		return nil, errorx.NewCodeError(errorx.CodeDeviceBoundByOther, "该设备已被其他用户绑定")
	}
	if activeBind != nil && activeBind.UserID == userID {
		deviceName := strings.TrimSpace(activeBind.DeviceName)
		if deviceName == "" {
			deviceName = snNorm
		}
		return &Result{
			UserID:     userID,
			DeviceID:   device.ID,
			DeviceSN:   snNorm,
			DeviceName: deviceName,
			BindTime:   activeBind.BoundAt,
		}, nil
	}

	maxBinds := opts.MaxDeviceBinds
	if maxBinds <= 0 {
		maxBinds = 10
	}
	bindCount, err := countUserBinds(ctx, tx, userID)
	if err != nil {
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询绑定数失败")
	}
	if bindCount >= int64(maxBinds) {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, fmt.Sprintf("已达到最大绑定设备数限制（%d 台）", maxBinds))
	}

	bindTime := time.Now()
	historyBind, err := findBindByUserAndDevice(ctx, tx, userID, device.ID)
	if err != nil {
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询绑定状态失败")
	}
	if historyBind != nil {
		if err := reactivateBind(ctx, tx, userID, device.ID, snNorm, snNorm, bindTime); err != nil {
			return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "创建绑定关系失败")
		}
	} else {
		if err := insertBind(ctx, tx, userID, device.ID, snNorm, snNorm); err != nil {
			return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "创建绑定关系失败")
		}
	}

	if err := updateDeviceBindStatus(ctx, tx, device.ID, userID, bindTime); err != nil {
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "更新设备状态失败")
	}
	if err := incrementUserDeviceCount(ctx, tx, userID, bindTime); err != nil {
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "更新用户信息失败")
	}
	_ = insertBindLog(ctx, tx, userID, device.ID, snNorm, "", "bind", bindTime)

	if err := tx.Commit(); err != nil {
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "绑定失败，请稍后重试")
	}

	return &Result{
		UserID:     userID,
		DeviceID:   device.ID,
		DeviceSN:   snNorm,
		DeviceName: snNorm,
		BindTime:   bindTime,
	}, nil
}

func normalizeSN(sn string) string {
	return strings.ToUpper(strings.TrimSpace(sn))
}

func findUserByID(ctx context.Context, tx *sql.Tx, userID int64) (*userRow, error) {
	if userID <= 0 {
		return nil, sql.ErrNoRows
	}
	var row userRow
	err := tx.QueryRowContext(ctx, `
		SELECT id, status
		FROM public.users
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1`, userID,
	).Scan(&row.ID, &row.Status)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func findDeviceBySN(ctx context.Context, tx *sql.Tx, sn string) (*deviceRow, error) {
	if sn == "" {
		return nil, sql.ErrNoRows
	}
	var row deviceRow
	err := tx.QueryRowContext(ctx, `
		SELECT id, sn, status
		FROM public.device
		WHERE sn = $1
		LIMIT 1`, sn,
	).Scan(&row.ID, &row.SN, &row.Status)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func countUserBinds(ctx context.Context, tx *sql.Tx, userID int64) (int64, error) {
	var count int64
	err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM public.user_device_bind
		WHERE user_id = $1 AND status = 1
	`, userID).Scan(&count)
	return count, err
}

func findBindByUserAndDevice(ctx context.Context, tx *sql.Tx, userID, deviceID int64) (*userDeviceBindRow, error) {
	var row userDeviceBindRow
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

func findActiveBindByDeviceID(ctx context.Context, tx *sql.Tx, deviceID int64) (*userDeviceBindRow, error) {
	var row userDeviceBindRow
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

func insertBind(ctx context.Context, tx *sql.Tx, userID, deviceID int64, sn, deviceName string) error {
	alias := truncateAlias(deviceName)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.user_device_bind
		  (user_id, device_id, sn, alias, is_default, bind_type, status, bound_at)
		VALUES ($1, $2, $3, $4, 0, 1, 1, CURRENT_TIMESTAMP)
	`, userID, deviceID, sn, alias)
	return err
}

func reactivateBind(ctx context.Context, tx *sql.Tx, userID, deviceID int64, sn, deviceName string, bindTime time.Time) error {
	alias := truncateAlias(deviceName)
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

func insertBindLog(ctx context.Context, tx *sql.Tx, userID, deviceID int64, sn, operator, action string, actionTime time.Time) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.user_device_bind_log
		  (user_id, device_id, sn, operator, action, action_time)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, deviceID, sn, operator, action, actionTime)
	return err
}

func updateDeviceBindStatus(ctx context.Context, tx *sql.Tx, deviceID, userID int64, bindTime time.Time) error {
	cols, err := getDeviceTableColumns(ctx, tx)
	if err != nil {
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
		return nil
	}

	args = append(args, deviceID)
	query := fmt.Sprintf("UPDATE public.device SET %s WHERE id = $%d", strings.Join(sets, ", "), idx)
	_, err = tx.ExecContext(ctx, query, args...)
	return err
}

func incrementUserDeviceCount(ctx context.Context, tx *sql.Tx, userID int64, bindTime time.Time) error {
	cols, err := getUserProfileTableColumns(ctx, tx)
	if err != nil {
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
		args = append(args, bindTime)
		idx++
	}
	if cols.UpdatedAt {
		sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	}
	if len(sets) == 0 {
		return nil
	}

	args = append(args, userID)
	query := fmt.Sprintf("UPDATE public.user_profile SET %s WHERE user_id = $%d", strings.Join(sets, ", "), idx)
	_, err = tx.ExecContext(ctx, query, args...)
	return err
}

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

	cols := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols[strings.ToLower(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	deviceCols = &deviceTableColumns{
		BoundUserID: hasColumn(cols, "bound_user_id"),
		BoundAt:     hasColumn(cols, "bound_at"),
		BindStatus:  hasColumn(cols, "bind_status"),
		UpdatedAt:   hasColumn(cols, "updated_at"),
	}
	return deviceCols, nil
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

	cols := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols[strings.ToLower(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	profileCols = &userProfileTableColumns{
		DeviceCount:  hasColumn(cols, "device_count"),
		LastBindTime: hasColumn(cols, "last_bind_time"),
		UpdatedAt:    hasColumn(cols, "updated_at"),
	}
	return profileCols, nil
}

func hasColumn(m map[string]struct{}, key string) bool {
	_, ok := m[key]
	return ok
}

func truncateAlias(alias string) string {
	runes := []rune(alias)
	if len(runes) > 32 {
		return string(runes[:32])
	}
	return alias
}
