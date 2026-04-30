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
		SELECT 
			id, sn, model, product_key, firmware_version, hardware_version,
			mac, ip, online_status, status, user_id, last_heartbeat,
			created_at, updated_at, deleted_at
		FROM devices 
		WHERE sn = $1 AND deleted_at IS NULL
	`

	var device model.Device
	err := r.db.QueryRowContext(ctx, query, sn).Scan(
		&device.ID, &device.Sn, &device.Model, &device.ProductKey,
		&device.FirmwareVersion, &device.HardwareVersion, &device.Mac,
		&device.Ip, &device.OnlineStatus, &device.Status, &device.UserID,
		&device.LastHeartbeat, &device.CreatedAt, &device.UpdatedAt, &device.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}

	return &device, nil
}

// FindById 根据ID查询设备
func (r *DeviceRepo) FindById(ctx context.Context, id int64) (*model.Device, error) {
	query := `
		SELECT 
			id, sn, model, product_key, firmware_version, hardware_version,
			mac, ip, online_status, status, user_id, last_heartbeat,
			created_at, updated_at, deleted_at
		FROM devices 
		WHERE id = $1 AND deleted_at IS NULL
	`

	var device model.Device
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&device.ID, &device.Sn, &device.Model, &device.ProductKey,
		&device.FirmwareVersion, &device.HardwareVersion, &device.Mac,
		&device.Ip, &device.OnlineStatus, &device.Status, &device.UserID,
		&device.LastHeartbeat, &device.CreatedAt, &device.UpdatedAt, &device.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询设备失败: %v", err)
	}

	return &device, nil
}

// UpdateHeartbeat 更新设备心跳信息
func (r *DeviceRepo) UpdateHeartbeat(ctx context.Context, deviceId int64, onlineStatus int16, lastHeartbeat time.Time) error {
	query := `
		UPDATE devices 
		SET online_status = $1, last_heartbeat = $2, updated_at = NOW()
		WHERE id = $3 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, onlineStatus, lastHeartbeat, deviceId)
	if err != nil {
		return fmt.Errorf("更新设备心跳失败: %v", err)
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
		UPDATE devices 
		SET online_status = $1, updated_at = NOW()
		WHERE online_status = $2 
		AND last_heartbeat < NOW() - INTERVAL '1 minute' * $3
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

	// 构建IN查询参数
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT 
			id, sn, model, product_key, firmware_version, hardware_version,
			mac, ip, online_status, status, user_id, last_heartbeat,
			created_at, updated_at, deleted_at
		FROM devices 
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
			&device.FirmwareVersion, &device.HardwareVersion, &device.Mac,
			&device.Ip, &device.OnlineStatus, &device.Status, &device.UserID,
			&device.LastHeartbeat, &device.CreatedAt, &device.UpdatedAt, &device.DeletedAt,
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
	query := `SELECT COUNT(*) FROM devices WHERE deleted_at IS NULL`

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
		FROM devices 
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
	query := `SELECT COUNT(*) FROM devices WHERE online_status = $1 AND deleted_at IS NULL`

	var count int64
	err := r.db.QueryRowContext(ctx, query, model.DeviceOnlineStatusOnline).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计在线设备数失败: %v", err)
	}

	return count, nil
}

// CountOffline 统计离线设备数
func (r *DeviceRepo) CountOffline(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM devices WHERE online_status = $1 AND deleted_at IS NULL`

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
		SELECT COUNT(*) FROM devices d 
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
	query := `SELECT COUNT(*) FROM devices WHERE created_at >= $1 AND deleted_at IS NULL`

	var count int64
	err := r.db.QueryRowContext(ctx, query, start).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计今日新增设备数失败: %v", err)
	}

	return count, nil
}

// CountTodayActive 统计今日活跃设备数
func (r *DeviceRepo) CountTodayActive(ctx context.Context, start time.Time) (int64, error) {
	query := `SELECT COUNT(*) FROM devices WHERE last_heartbeat >= $1 AND deleted_at IS NULL`

	var count int64
	err := r.db.QueryRowContext(ctx, query, start).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计今日活跃设备数失败: %v", err)
	}

	return count, nil
}
