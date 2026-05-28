// Package repository 包含设备服务的所有数据访问层（Repository）实现
// 负责与数据库的直接交互，提供基础的 CRUD 操作
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// DeviceRepo 设备数据访问结构体
// 提供设备相关的数据库操作方法，如查询、创建、更新等
type DeviceRepo struct {
	db *sql.DB
}

// NewDeviceRepo 创建设备数据访问实例
// 参数 db *sql.DB: 数据库连接
// 返回 *DeviceRepo: 设备数据访问实例
func NewDeviceRepo(db *sql.DB) *DeviceRepo {
	return &DeviceRepo{db: db}
}

// FindBySn 根据设备 SN 查询设备信息
// 参数 ctx context.Context: 请求上下文
// 参数 sn string: 设备序列号
// 返回 *model.Device: 设备信息指针，如果未找到则返回 nil
// 返回 error: 查询失败时的错误信息
func (r *DeviceRepo) FindBySn(ctx context.Context, sn string) (*model.Device, error) {
	query := `
		SELECT id, sn, product_key, device_secret, status, online_status,
		       created_at, updated_at, deleted_at
		FROM public.device
		WHERE sn = $1 AND deleted_at IS NULL
	`

	var device model.Device
	err := r.db.QueryRowContext(ctx, query, sn).Scan(
		&device.ID, &device.Sn, &device.ProductKey,
		&device.DeviceSecret, &device.Status, &device.OnlineStatus,
		&device.CreatedAt, &device.UpdatedAt, &device.DeletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}

	return &device, nil
}

// FindBySnIncludingDeleted 按 SN 查询设备（不过滤 deleted_at）。
// 用于注册：软删除行仍会占用 sn 唯一约束，仅靠 FindBySn 会误判为「可 INSERT」导致 23505。
func (r *DeviceRepo) FindBySnIncludingDeleted(ctx context.Context, sn string) (*model.Device, error) {
	query := `
		SELECT id, sn, product_key, device_secret, status, online_status,
		       created_at, updated_at, deleted_at
		FROM public.device
		WHERE sn = $1
	`

	var device model.Device
	err := r.db.QueryRowContext(ctx, query, sn).Scan(
		&device.ID, &device.Sn, &device.ProductKey,
		&device.DeviceSecret, &device.Status, &device.OnlineStatus,
		&device.CreatedAt, &device.UpdatedAt, &device.DeletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}

	return &device, nil
}

// ClearDeletedAt 清除软删除标记（设备重新注册恢复）
func (r *DeviceRepo) ClearDeletedAt(ctx context.Context, deviceID int64) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE public.device
		SET deleted_at = NULL, updated_at = NOW()
		WHERE id = $1`, deviceID)
	if err != nil {
		return fmt.Errorf("恢复设备失败: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("恢复设备失败: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("恢复设备失败: 未更新任何行")
	}
	return nil
}

// FindById 根据ID查询设备（含 WS 注册用到的 register_signature / register_timestamp）
func (r *DeviceRepo) FindById(ctx context.Context, id int64) (*model.Device, error) {
	query := `
		SELECT id, sn, product_key, device_secret, status, online_status,
		       created_at, updated_at, deleted_at
		FROM public.device 
		WHERE id = $1 AND deleted_at IS NULL
	`

	var device model.Device
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&device.ID, &device.Sn, &device.ProductKey,
		&device.DeviceSecret, &device.Status, &device.OnlineStatus,
		&device.CreatedAt, &device.UpdatedAt, &device.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}

	return &device, nil
}

// UpdateLastActive 更新设备最后活跃时间
func (r *DeviceRepo) UpdateLastActive(ctx context.Context, deviceId int64, onlineStatus int16, lastActive time.Time) error {
	query := `
		UPDATE device 
		SET online_status = $1, last_active_at = $2, updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, onlineStatus, lastActive, deviceId)
	if err != nil {
		return fmt.Errorf("更新设备活跃时间失败: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("设备不存在或已被删除")
	}

	return nil
}

// FindBySnForActivation 根据SN查询预录入设备（用于注册/激活流程）
// 查询条件：SN匹配 + 未删除 + 状态为未激活(DeviceStatusUnregistered=4) 或已激活(DeviceStatusNormal=1)
// 返回完整设备信息（包含密钥、状态、版本等字段），用于密钥验证和状态检查
//
// 参数 ctx context.Context: 请求上下文
// 参数 sn string: 设备序列号
// 返回 *model.Device: 设备完整信息指针，如果未找到则返回 nil
// 返回 error: 查询失败时的错误信息
func (r *DeviceRepo) FindBySnForActivation(ctx context.Context, sn string) (*model.Device, error) {
	query := `
		SELECT id, sn, model, product_key, device_secret, register_signature,
		       register_timestamp, firmware_version, hardware_version, mac,
		       COALESCE(device_name_raw, ''), ip, online_status, usage_status, status, create_by,
		       last_active_at, created_at, updated_at, deleted_at
		FROM public.device
		WHERE sn = $1 AND deleted_at IS NULL
		  AND status IN ($2, $3, $4, $5)
	`

	var device model.Device
	err := r.db.QueryRowContext(ctx, query, sn,
		model.DeviceStatusDefault,
		model.DeviceStatusUnregistered,
		model.DeviceStatusUnauthenticated,
		model.DeviceStatusNormal,
	).Scan(
		&device.ID, &device.Sn, &device.Model, &device.ProductKey,
		&device.DeviceSecret, &device.RegisterSignature,
		&device.RegisterTimestamp, &device.FirmwareVersion,
		&device.HardwareVersion, &device.Mac, &device.DeviceNameRaw, &device.Ip,
		&device.OnlineStatus, &device.UsageStatus, &device.Status,
		&device.CreateBy, &device.LastActiveAt, &device.CreatedAt,
		&device.UpdatedAt, &device.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询预录入设备失败: %v", err)
	}

	return &device, nil
}

// ActivateDevice 激活设备（更新状态和激活信息）
// 将设备从未激活状态(DeviceStatusUnregistered=4)更新为已激活状态(DeviceStatusNormal=1)
// 同时记录：激活时间、IP地址、固件版本、硬件版本、MAC地址、最后活跃时间
//
// 参数 ctx context.Context: 请求上下文
// 参数 deviceId int64: 设备ID
// 参数 ip string: 设备当前IP地址
// 参数 firmwareVersion string: 固件版本号
// 参数 hardwareVersion string: 硬件版本号
// 参数 mac string: MAC地址
// 返回 error: 更新失败时的错误信息
func (r *DeviceRepo) ActivateDevice(ctx context.Context, deviceId int64, ip string, firmwareVersion string, hardwareVersion string, mac string, deviceNameRaw string) error {
	query := `
		UPDATE public.device
		SET status = $1,
		    ip = $2,
		    firmware_version = COALESCE(NULLIF($3, ''), firmware_version),
		    hardware_version = COALESCE(NULLIF($4, ''), hardware_version),
		    mac = COALESCE(NULLIF($5, ''), mac),
		    device_name_raw = CASE WHEN NULLIF(TRIM($6), '') IS NOT NULL THEN TRIM($6) ELSE device_name_raw END,
		    last_active_at = NOW(),
		    updated_at = NOW()
		WHERE id = $7
		  AND deleted_at IS NULL
		  AND status IN ($8, $9, $10)
	`

	result, err := r.db.ExecContext(ctx, query,
		model.DeviceStatusNormal,
		ip,
		firmwareVersion,
		hardwareVersion,
		mac,
		deviceNameRaw,
		deviceId,
		model.DeviceStatusDefault,
		model.DeviceStatusUnregistered,
		model.DeviceStatusUnauthenticated,
	)

	if err != nil {
		return fmt.Errorf("激活设备失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("设备不存在或状态不正确（可能已被激活或已删除）")
	}

	return nil
}

// UpdateDeviceNameRaw 写入设备原始名称（非空时覆盖）
func (r *DeviceRepo) UpdateDeviceNameRaw(ctx context.Context, deviceID int64, deviceNameRaw string) error {
	name := strings.TrimSpace(deviceNameRaw)
	if name == "" || deviceID <= 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE public.device
		SET device_name_raw = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`, name, deviceID)
	if err != nil {
		return fmt.Errorf("更新设备原始名称失败: %w", err)
	}
	return nil
}

// ResetForPreRegistration 将设备重置为未激活预录入状态并更新密钥（用于注册前自动预录入/恢复）
func (r *DeviceRepo) ResetForPreRegistration(ctx context.Context, deviceID int64, deviceSecret, deviceNameRaw string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE public.device
		SET device_secret = $1,
		    status = $2,
		    online_status = $3,
		    device_name_raw = CASE WHEN NULLIF(TRIM($4), '') IS NOT NULL THEN TRIM($4) ELSE device_name_raw END,
		    updated_at = NOW()
		WHERE id = $5 AND deleted_at IS NULL
	`, strings.TrimSpace(deviceSecret), model.DeviceStatusUnregistered, model.DeviceOnlineStatusOffline, strings.TrimSpace(deviceNameRaw), deviceID)
	if err != nil {
		return fmt.Errorf("重置预录入设备失败: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("重置预录入设备失败: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("重置预录入设备失败: 未更新任何行")
	}
	return nil
}

// CreateWithFullInfo 创建新设备记录（包含完整信息）
// 用于设备注册时创建完整的设备记录，包括密钥、签名、时间戳等
//
// 参数 ctx context.Context: 请求上下文
// 参数 device *model.Device: 设备对象（包含所有字段信息）
// 返回 int64: 新创建的设备 ID
// 返回 error: 创建失败时的错误信息
func (r *DeviceRepo) CreateWithFullInfo(ctx context.Context, device *model.Device) (int64, error) {
	query := `
		INSERT INTO device (sn, product_key, device_secret, device_name_raw, status, online_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id
	`

	var deviceID int64
	err := r.db.QueryRowContext(ctx, query,
		device.Sn,
		device.ProductKey,
		device.DeviceSecret,
		strings.TrimSpace(device.DeviceNameRaw),
		device.Status,
		device.OnlineStatus,
	).Scan(&deviceID)

	if err != nil {
		return 0, fmt.Errorf("创建设备记录失败: %v", err)
	}

	return deviceID, nil
}

// UpdateRegisterSignature 更新设备注册签名
// 参数 ctx context.Context: 请求上下文
// 参数 deviceId int64: 设备ID
// 参数 signature string: 注册签名（HMAC-SHA256签名）
// 参数 registerTimestamp int64: 注册时间戳（毫秒级Unix时间戳）
// 返回 error: 更新失败时的错误信息
func (r *DeviceRepo) UpdateRegisterSignature(ctx context.Context, deviceId int64, signature string, registerTimestamp int64) error {
	query := `
		UPDATE public.device
		SET register_signature = $1,
		    register_timestamp = $2,
		    updated_at = $3
		WHERE id = $4 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, signature, registerTimestamp, time.Now(), deviceId)
	if err != nil {
		return fmt.Errorf("更新设备注册签名失败: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("设备不存在或已被删除")
	}

	return nil
}

// UpdateOfflineDevices 批量更新离线设备
func (r *DeviceRepo) UpdateOfflineDevices(ctx context.Context, timeoutMinutes int) (int64, error) {
	query := `
		UPDATE device 
		SET online_status = $1, updated_at = NOW()
		WHERE online_status = $2 
		AND last_active_at < NOW() - INTERVAL '1 minute' * $3
		AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, model.DeviceOnlineStatusOffline, model.DeviceOnlineStatusOnline, timeoutMinutes)
	if err != nil {
		return 0, fmt.Errorf("更新离线设备失败: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %v", err)
	}

	return rowsAffected, nil
}

// FindByIds 批量查询设备
func (r *DeviceRepo) FindByIds(ctx context.Context, ids []int64) (map[int64]*model.Device, error) {
	if len(ids) == 0 {
		return make(map[int64]*model.Device), nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT 
			id, sn, model, product_key, device_secret,
			firmware_version, hardware_version, mac, ip,
			online_status, status, create_by, last_active_at,
			created_at, updated_at, deleted_at
		FROM device 
		WHERE id IN (%s) AND deleted_at IS NULL
	`, strings.Join(placeholders, ", "))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("批量查询设备失败: %v", err)
	}
	defer rows.Close()

	deviceMap := make(map[int64]*model.Device)
	for rows.Next() {
		var device model.Device
		err := rows.Scan(
			&device.ID, &device.Sn, &device.Model, &device.ProductKey,
			&device.DeviceSecret, &device.FirmwareVersion, &device.HardwareVersion,
			&device.Mac, &device.Ip, &device.OnlineStatus, &device.Status,
			&device.CreateBy, &device.LastActiveAt,
			&device.CreatedAt, &device.UpdatedAt, &device.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描设备数据失败: %v", err)
		}

		deviceMap[device.ID] = &device
	}

	return deviceMap, nil
}

// CountTotal 统计总设备数
func (r *DeviceRepo) CountTotal(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM device WHERE deleted_at IS NULL`

	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计总设备数失败: %v", err)
	}

	return count, nil
}

// GetDistinctProductKey 获取去重的productKey列表
func (r *DeviceRepo) GetDistinctProductKey(ctx context.Context) ([]types.EnumItem, error) {
	query := `
		SELECT DISTINCT product_key as value, product_key as label 
		FROM device 
		WHERE product_key != '' AND deleted_at IS NULL
		ORDER BY product_key
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询去重productKey失败: %v", err)
	}
	defer rows.Close()

	var productList []types.EnumItem
	for rows.Next() {
		var value, label string
		err := rows.Scan(&value, &label)
		if err != nil {
			return nil, fmt.Errorf("扫描productKey数据失败: %v", err)
		}

		productList = append(productList, types.EnumItem{
			Label: label,
			Value: value,
		})
	}

	return productList, nil
}

// CountOnline 统计在线设备数
func (r *DeviceRepo) CountOnline(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM device WHERE online_status = $1 AND deleted_at IS NULL`

	var count int64
	err := r.db.QueryRowContext(ctx, query, model.DeviceOnlineStatusOnline).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计在线设备数失败: %v", err)
	}

	return count, nil
}

// CountOffline 统计离线设备数
func (r *DeviceRepo) CountOffline(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM device WHERE online_status = $1 AND deleted_at IS NULL`

	var count int64
	err := r.db.QueryRowContext(ctx, query, model.DeviceOnlineStatusOffline).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计离线设备数失败: %v", err)
	}

	return count, nil
}

// CountUnbound 统计未绑定设备数
func (r *DeviceRepo) CountUnbound(ctx context.Context) (int64, error) {
	query := `
		SELECT COUNT(*) FROM device d 
		WHERE d.deleted_at IS NULL 
		AND d.id NOT IN (
			SELECT device_id FROM user_device_bind 
			WHERE deleted_at IS NULL AND status = $1
		)
	`

	var count int64
	err := r.db.QueryRowContext(ctx, query, model.UserDeviceBindStatusNormal).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计未绑定设备数失败: %v", err)
	}

	return count, nil
}

// CountTodayAdd 统计今日新增设备数
func (r *DeviceRepo) CountTodayAdd(ctx context.Context, start time.Time) (int64, error) {
	query := `SELECT COUNT(*) FROM device WHERE created_at >= $1 AND deleted_at IS NULL`

	var count int64
	err := r.db.QueryRowContext(ctx, query, start).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计今日新增设备数失败: %v", err)
	}

	return count, nil
}

// CountTodayActive 统计今日活跃设备数
func (r *DeviceRepo) CountTodayActive(ctx context.Context, start time.Time) (int64, error) {
	query := `SELECT COUNT(*) FROM device WHERE last_active_at >= $1 AND deleted_at IS NULL`

	var count int64
	err := r.db.QueryRowContext(ctx, query, start).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计今日活跃设备数失败: %v", err)
	}

	return count, nil
}

// UpdateStatusAfterRegister 更新设备注册后的状态
// 参数 ctx context.Context: 请求上下文
// 参数 deviceId int64: 设备ID
// 参数 status int16: 新状态（1=已注册）
// 返回 error: 更新失败时的错误信息
func (r *DeviceRepo) UpdateStatusAfterRegister(ctx context.Context, deviceId int64, status int16) error {
	query := `
		UPDATE device
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, status, deviceId)
	if err != nil {
		return fmt.Errorf("更新设备注册状态失败: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("设备不存在或已被删除")
	}

	return nil
}

// UpdateDeviceName 更新 device 表用户备注名
func (r *DeviceRepo) UpdateDeviceName(ctx context.Context, deviceID int64, deviceName string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE public.device
		SET device_name = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, deviceID, deviceName)
	if err != nil {
		return fmt.Errorf("更新设备名称失败: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("设备不存在或已被删除")
	}
	return nil
}
