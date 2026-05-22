package dao

import (
	"context"
	"database/sql"
	"time"
)

// UserDeviceBindRepo 用户与平台设备（device 表）的绑定关系；status=1 绑定中，0 已解绑（见迁移 035）。
type UserDeviceBindRepo struct {
	db *sql.DB
}

func NewUserDeviceBindRepo(db *sql.DB) *UserDeviceBindRepo {
	return &UserDeviceBindRepo{db: db}
}

// FindDeviceIDBySN 按设备序列号查 device.id；未找到返回 ok=false。
func (r *UserDeviceBindRepo) FindDeviceIDBySN(ctx context.Context, sn string) (deviceID int64, ok bool, err error) {
	err = r.db.QueryRowContext(ctx, `SELECT id FROM public.device WHERE sn = $1 LIMIT 1`, sn).Scan(&deviceID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return deviceID, true, nil
}

// UserDeviceBindRow 绑定行（仅活跃绑定查询用）。
type UserDeviceBindRow struct {
	ID            int64
	UserID        int64
	DeviceID      int64
	SN            string
	Alias         string
	DeviceName    string
	DeviceModel   string
	SystemVersion string
	Status        int16
	BoundAt       time.Time
	UnboundAt     sql.NullTime
}

// FindActiveBindByDeviceID 查找当前绑定中（status=1）的记录。
// 仅使用 026_device_module 既有列（含 alias），避免未执行 033 迁移时引用 device_name 等列导致 42703。
func (r *UserDeviceBindRepo) FindActiveBindByDeviceID(ctx context.Context, deviceID int64) (*UserDeviceBindRow, error) {
	var row UserDeviceBindRow
	err := r.db.QueryRowContext(ctx, `
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

// FindActiveBindBySN 按 user_device_bind.sn 查当前绑定中（status=1）的一条记录。
func (r *UserDeviceBindRepo) FindActiveBindBySN(ctx context.Context, sn string) (*UserDeviceBindRow, error) {
	var row UserDeviceBindRow
	err := r.db.QueryRowContext(ctx, `
SELECT id, user_id, device_id, sn, COALESCE(alias,''),
       status, bound_at, unbound_at
  FROM public.user_device_bind
 WHERE sn = $1 AND status = 1
 LIMIT 1
`, sn).Scan(
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

// InsertBind 新建绑定（调用方已确认无其它用户活跃绑定）。
// 写入 026 既有列：展示名写入 alias（最长 32 字符）；model/system_version 在未执行 033 前不落库。
func (r *UserDeviceBindRepo) InsertBind(ctx context.Context, userID, deviceID int64, sn, deviceName, deviceModel, systemVersion string) error {
	_, _ = deviceModel, systemVersion // 026 表无独立列；执行 033 迁移后可扩展落库
	alias := deviceName
	if len([]rune(alias)) > 32 {
		alias = string([]rune(deviceName)[:32])
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO public.user_device_bind
  (user_id, device_id, sn, alias, is_default, bind_type, status, bound_at)
VALUES ($1, $2, $3, $4, 0, 1, 1, CURRENT_TIMESTAMP)
`, userID, deviceID, sn, alias)
	return err
}

// UpsertBind 插入或更新绑定记录（支持重新绑定已解绑的设备）
func (r *UserDeviceBindRepo) UpsertBind(ctx context.Context, userID, deviceID int64, sn, deviceName string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if ierr := bindUpsertInTx(ctx, tx, userID, deviceID, sn, deviceName); ierr != nil {
		err = ierr
		return err
	}
	return tx.Commit()
}

// UnbindForUser 将当前用户对某设备的活跃绑定标记为解绑（status=0，不删行）。
func (r *UserDeviceBindRepo) UnbindForUser(ctx context.Context, userID, deviceID int64) (affected int64, err error) {
	res, err := r.db.ExecContext(ctx, `
UPDATE public.user_device_bind
   SET status = 0, unbound_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
 WHERE user_id = $1 AND device_id = $2 AND status = 1
`, userID, deviceID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ClearDeviceBindStatus 解绑后清空设备表中的绑定状态（bind_status=0）
func (r *UserDeviceBindRepo) ClearDeviceBindStatus(ctx context.Context, deviceID int64) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE public.device
   SET bind_status = 0,
       bound_user_id = NULL,
       bound_at = NULL,
       updated_at = CURRENT_TIMESTAMP
 WHERE id = $1
`, deviceID)
	return err
}

// UnbindDeviceWithTransaction 使用事务完成完整的解绑流程（保证原子性）
func (r *UserDeviceBindRepo) UnbindDeviceWithTransaction(ctx context.Context, userID, deviceID int64) (affected int64, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	res, err := tx.ExecContext(ctx, `
UPDATE public.user_device_bind
   SET status = 0, unbound_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
 WHERE user_id = $1 AND device_id = $2 AND status = 1
`, userID, deviceID)

	if err != nil {
		return 0, err
	}

	n, _ := res.RowsAffected()

	_, err = tx.ExecContext(ctx, `
UPDATE public.device
   SET bind_status = 0,
       bound_user_id = NULL,
       bound_at = NULL,
       updated_at = CURRENT_TIMESTAMP
 WHERE id = $1
`, deviceID)

	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return n, nil
}

// UpdateDeviceBindStatusForUser 绑定时更新设备表的绑定状态（bind_status=1）
func (r *UserDeviceBindRepo) UpdateDeviceBindStatusForUser(ctx context.Context, deviceID, userID int64) error {
	_, err := r.db.ExecContext(ctx, `
UPDATE public.device
   SET bind_status = 1,
       bound_user_id = $2,
       bound_at = CURRENT_TIMESTAMP,
       updated_at = CURRENT_TIMESTAMP
 WHERE id = $1
`, deviceID, userID)
	return err
}

// BindDeviceWithTransaction 使用事务完成完整的绑定流程（保证原子性）
func (r *UserDeviceBindRepo) BindDeviceWithTransaction(ctx context.Context, userID, deviceID int64, sn, deviceName string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if ierr := bindUpsertInTx(ctx, tx, userID, deviceID, sn, deviceName); ierr != nil {
		err = ierr
		return err
	}

	_, err = tx.ExecContext(ctx, `
UPDATE public.device
   SET bind_status = 1,
       bound_user_id = $2,
       bound_at = CURRENT_TIMESTAMP,
       updated_at = CURRENT_TIMESTAMP
 WHERE id = $1
`, deviceID, userID)

	if err != nil {
		return err
	}

	return tx.Commit()
}

// bindUpsertInTx 先按 (user_id, device_id) 复活历史行（解绑后 status=0），否则插入。
// 不使用 ON CONFLICT：部分库未建好 uk_user_device_bind 时会报 42P10，导致「解绑后无法再绑定」。
func bindUpsertInTx(ctx context.Context, tx *sql.Tx, userID, deviceID int64, sn, deviceName string) error {
	alias := deviceName
	if len([]rune(alias)) > 32 {
		alias = string([]rune(deviceName)[:32])
	}

	res, err := tx.ExecContext(ctx, `
UPDATE public.user_device_bind
   SET sn = $3,
       alias = $4,
       status = 1,
       bound_at = CURRENT_TIMESTAMP,
       unbound_at = NULL,
       updated_at = CURRENT_TIMESTAMP
 WHERE user_id = $1 AND device_id = $2
`, userID, deviceID, sn, alias)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO public.user_device_bind
  (user_id, device_id, sn, alias, is_default, bind_type, status, bound_at)
VALUES ($1, $2, $3, $4, 0, 1, 1, CURRENT_TIMESTAMP)
`, userID, deviceID, sn, alias)
	return err
}

// ListActiveByUserID 用户当前绑定中的设备列表；nameSub/snSub/modelSub 非空时在库内做子串匹配（AND）。
func (r *UserDeviceBindRepo) ListActiveByUserID(ctx context.Context, userID int64, nameSub, snSub, modelSub string) ([]UserDeviceListItem, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT d.sn,
       COALESCE(udb.alias,''),
       COALESCE(d.model,''),
       COALESCE(d.firmware_version,''),
       udb.bound_at
  FROM public.user_device_bind udb
  JOIN public.device d ON d.id = udb.device_id
 WHERE udb.user_id = $1 AND udb.status = 1
   AND ($2::text = '' OR strpos(lower(COALESCE(udb.alias,'')), lower($2::text)) > 0)
   AND ($3::text = '' OR strpos(lower(d.sn), lower($3::text)) > 0)
   AND ($4::text = '' OR strpos(lower(COALESCE(d.model,'')), lower($4::text)) > 0)
 ORDER BY udb.bound_at DESC
`, userID, nameSub, snSub, modelSub)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []UserDeviceListItem
	for rows.Next() {
		var it UserDeviceListItem
		if err := rows.Scan(&it.DeviceSn, &it.DeviceName, &it.DeviceModel, &it.SystemVersion, &it.BoundAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// DeviceBindStatus 设备绑定状态信息
type DeviceBindStatus struct {
	BindStatus  int16
	BoundUserID *int64
}

// CheckDeviceBindStatus 检查设备表的绑定状态（用于调试和辅助判断）
func (r *UserDeviceBindRepo) CheckDeviceBindStatus(ctx context.Context, sn string) (*DeviceBindStatus, error) {
	var status DeviceBindStatus
	err := r.db.QueryRowContext(ctx, `
SELECT COALESCE(bind_status, 0)::smallint,
       bound_user_id
  FROM public.device
 WHERE sn = $1
 LIMIT 1
`, sn).Scan(&status.BindStatus, &status.BoundUserID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// UserDeviceListItem 列表一行（含绑定时间）。
type UserDeviceListItem struct {
	DeviceSn      string
	DeviceName    string
	DeviceModel   string
	SystemVersion string
	BoundAt       time.Time
}

// FindActiveBindByDeviceIDWithTx 使用事务查找当前绑定中（status=1）的记录
func (r *UserDeviceBindRepo) FindActiveBindByDeviceIDWithTx(ctx context.Context, tx *sql.Tx, deviceID int64) (*UserDeviceBindRow, error) {
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

// CountUserBindsWithTx 统计用户当前绑定中的设备数
func (r *UserDeviceBindRepo) CountUserBindsWithTx(ctx context.Context, tx *sql.Tx, userID int64) (int64, error) {
	var count int64
	err := tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM public.user_device_bind WHERE user_id = $1 AND status = 1
`, userID).Scan(&count)
	return count, err
}

// InsertBindWithTx 使用事务插入绑定记录
func (r *UserDeviceBindRepo) InsertBindWithTx(ctx context.Context, tx *sql.Tx, userID, deviceID int64, sn, deviceName, deviceModel, systemVersion, operator string) error {
	alias := deviceName
	if len([]rune(alias)) > 32 {
		alias = string([]rune(deviceName)[:32])
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO public.user_device_bind
  (user_id, device_id, sn, alias, is_default, bind_type, status, bound_at, operator)
VALUES ($1, $2, $3, $4, 0, 1, 1, CURRENT_TIMESTAMP, $5)
`, userID, deviceID, sn, alias, operator)
	return err
}

// InsertBindLogWithTx 插入绑定操作日志
func (r *UserDeviceBindRepo) InsertBindLogWithTx(ctx context.Context, tx *sql.Tx, userID, deviceID int64, sn, operator, action string, actionTime time.Time) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO public.user_device_bind_log
  (user_id, device_id, sn, operator, action, action_time)
VALUES ($1, $2, $3, $4, $5, $6)
`, userID, deviceID, sn, operator, action, actionTime)
	return err
}

// UpdateDeviceAuxiliaryInfo 更新用户设备的辅助信息（备注名、位置、分组、场景）
// 只更新请求中非空字段，支持部分更新
// 返回值：
//   - affected: 影响的行数（0表示未找到记录或无权限）
//   - error: 错误信息
func (r *UserDeviceBindRepo) UpdateDeviceAuxiliaryInfo(ctx context.Context, userID int64, sn string, alias, location, groupName, scene string) (affected int64, err error) {
	result, err := r.db.ExecContext(ctx, `
UPDATE public.user_device_bind
   SET alias = COALESCE(NULLIF($3,''), alias),
       location = COALESCE(NULLIF($4,''), location),
       group_name = COALESCE(NULLIF($5,''), group_name),
       scene = COALESCE(NULLIF($6,''), scene),
       updated_at = CURRENT_TIMESTAMP
 WHERE user_id = $1 AND sn = $2 AND status = 1
`, userID, sn, alias, location, groupName, scene)

	if err != nil {
		return 0, err
	}

	n, _ := result.RowsAffected()
	return n, nil
}
