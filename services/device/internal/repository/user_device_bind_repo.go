package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
)

type UserDeviceBindRepo struct {
	db *sql.DB
}

func NewUserDeviceBindRepo(db *sql.DB) *UserDeviceBindRepo {
	return &UserDeviceBindRepo{db: db}
}

// FindListByUserId 根据用户ID查询绑定的设备列表
func (r *UserDeviceBindRepo) FindListByUserId(ctx context.Context, userId int64) ([]*model.UserDeviceBind, error) {
	query := `
		SELECT 
			id, user_id, device_id, sn, status, created_at, updated_at, deleted_at
		FROM user_device_bind 
		WHERE user_id = $1 AND status = $2 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userId, model.UserDeviceBindStatusNormal)
	if err != nil {
		return nil, fmt.Errorf("查询用户设备绑定列表失败: %v", err)
	}
	defer rows.Close()

	var bindList []*model.UserDeviceBind
	for rows.Next() {
		var bind model.UserDeviceBind
		err := rows.Scan(
			&bind.ID, &bind.UserID, &bind.DeviceID, &bind.SN, &bind.Status,
			&bind.CreatedAt, &bind.UpdatedAt, &bind.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描绑定数据失败: %v", err)
		}

		bindList = append(bindList, &bind)
	}

	return bindList, nil
}

// FindByDeviceId 根据设备ID查询是否已被绑定
func (r *UserDeviceBindRepo) FindByDeviceId(ctx context.Context, deviceId int64) (*model.UserDeviceBind, error) {
	query := `
		SELECT 
			id, user_id, device_id, sn, status, created_at, updated_at, deleted_at
		FROM user_device_bind 
		WHERE device_id = $1 AND status = $2 AND deleted_at IS NULL
	`

	var bind model.UserDeviceBind
	err := r.db.QueryRowContext(ctx, query, deviceId, model.UserDeviceBindStatusNormal).Scan(
		&bind.ID, &bind.UserID, &bind.DeviceID, &bind.SN, &bind.Status,
		&bind.CreatedAt, &bind.UpdatedAt, &bind.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询设备绑定状态失败: %v", err)
	}

	return &bind, nil
}

// CountByUserId 统计用户已绑定的设备数量
func (r *UserDeviceBindRepo) CountByUserId(ctx context.Context, userId int64) (int64, error) {
	query := `
		SELECT COUNT(1)
		FROM user_device_bind 
		WHERE user_id = $1 AND status = $2 AND deleted_at IS NULL
	`

	var count int64
	err := r.db.QueryRowContext(ctx, query, userId, model.UserDeviceBindStatusNormal).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计用户绑定设备数量失败: %v", err)
	}

	return count, nil
}

// ExistsActiveByUserAndSN 用户是否仍绑定该 SN（status=1）
func (r *UserDeviceBindRepo) ExistsActiveByUserAndSN(ctx context.Context, userId int64, sn string) (bool, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM user_device_bind
		WHERE user_id = $1 AND UPPER(sn) = UPPER($2) AND status = $3
		  AND (deleted_at IS NULL)
	`, userId, sn, model.UserDeviceBindStatusNormal).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("查询用户设备绑定失败: %w", err)
	}
	return count > 0, nil
}

// FindByUserIdAndDeviceId 根据用户ID和设备ID查询绑定关系
func (r *UserDeviceBindRepo) FindByUserIdAndDeviceId(ctx context.Context, userId, deviceId int64) (*model.UserDeviceBind, error) {
	query := `
		SELECT 
			id, user_id, device_id, sn,
			COALESCE(device_name,''), COALESCE(location,''), COALESCE(group_name,''), COALESCE(scene,''),
			status, created_at, updated_at, deleted_at
		FROM user_device_bind 
		WHERE user_id = $1 AND device_id = $2 AND deleted_at IS NULL
	`

	var bind model.UserDeviceBind
	err := r.db.QueryRowContext(ctx, query, userId, deviceId).Scan(
		&bind.ID, &bind.UserID, &bind.DeviceID, &bind.SN,
		&bind.DeviceName, &bind.Location, &bind.GroupName, &bind.Scene,
		&bind.Status, &bind.CreatedAt, &bind.UpdatedAt, &bind.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询用户设备绑定失败: %v", err)
	}

	return &bind, nil
}

// Create 创建用户设备绑定
func (r *UserDeviceBindRepo) Create(ctx context.Context, bind *model.UserDeviceBind) error {
	query := `
		INSERT INTO user_device_bind (user_id, device_id, sn, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id
	`

	err := r.db.QueryRowContext(ctx, query, bind.UserID, bind.DeviceID, bind.SN, bind.Status).Scan(&bind.ID)
	if err != nil {
		return fmt.Errorf("创建用户设备绑定失败: %v", err)
	}

	return nil
}

// UpdateStatus 更新绑定状态
func (r *UserDeviceBindRepo) UpdateStatus(ctx context.Context, id int64, status int16) error {
	query := `
		UPDATE user_device_bind 
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("更新绑定状态失败: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("绑定记录不存在或已被删除")
	}

	return nil
}

// UpdateDeviceInfo 更新绑定记录的备注名、位置、分组、场景
func (r *UserDeviceBindRepo) UpdateDeviceInfo(ctx context.Context, bindID int64, deviceName, location, groupName, scene string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE user_device_bind
		SET device_name = COALESCE(NULLIF($2,''), device_name),
		    location = $3,
		    group_name = $4,
		    scene = $5,
		    updated_at = NOW()
		WHERE id = $6 AND deleted_at IS NULL
	`, bindID, deviceName, location, groupName, scene, bindID)
	if err != nil {
		return fmt.Errorf("更新设备信息失败: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("绑定记录不存在或已被删除")
	}
	return nil
}
