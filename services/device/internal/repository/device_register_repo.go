// Package repository 包含设备注册的数据访问层（Repository）实现
// 负责设备注册相关的数据库操作
package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"math/rand"
	"time"
)

// DeviceRegisterRepo 设备注册数据访问结构体
// 提供设备注册相关的数据库操作方法
type DeviceRegisterRepo struct {
	db *sql.DB
}

// NewDeviceRegisterRepo 创建设备注册数据访问实例
// 参数 db *sql.DB: 数据库连接
// 返回 *DeviceRegisterRepo: 设备注册数据访问实例
func NewDeviceRegisterRepo(db *sql.DB) *DeviceRegisterRepo {
	return &DeviceRegisterRepo{db: db}
}

// FindBySn 根据设备 SN 查询设备是否已注册
// 参数 ctx context.Context: 请求上下文
// 参数 sn string: 设备序列号
// 返回 *DeviceRegisterInfo: 设备注册信息指针，如果未找到则返回 nil
// 返回 error: 查询失败时的错误信息
func (r *DeviceRegisterRepo) FindBySn(ctx context.Context, sn string) (*DeviceRegisterInfo, error) {
	query := `
		SELECT id, sn, model, firmware_version, auth_token
		FROM device
		WHERE sn = $1 AND status = 1
	`

	var info DeviceRegisterInfo
	err := r.db.QueryRowContext(ctx, query, sn).Scan(
		&info.ID, &info.Sn, &info.Model, &info.FirmwareVersion, &info.AuthToken,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询设备注册信息失败: %v", err)
	}

	return &info, nil
}

// CreateDevice 创建新设备记录（新设备注册）
// 参数 ctx context.Context: 请求上下文
// 参数 sn string: 设备序列号
// 参数 model string: 设备型号
// 参数 firmwareVersion string: 固件版本
// 参数 authToken string: 认证 token
// 返回 int64: 新创建的设备 ID
// 返回 error: 创建失败时的错误信息
func (r *DeviceRegisterRepo) CreateDevice(ctx context.Context, sn string, model string, firmwareVersion string, authToken string) (int64, error) {
	query := `
		INSERT INTO device (sn, model, firmware_version, auth_token, product_key, device_secret, status, online_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'default', '', 1, 0, NOW(), NOW())
		RETURNING id
	`

	var deviceID int64
	err := r.db.QueryRowContext(ctx, query, sn, model, firmwareVersion, authToken).Scan(&deviceID)
	if err != nil {
		return 0, fmt.Errorf("创建设备记录失败: %v", err)
	}

	return deviceID, nil
}

// GenerateAuthToken 生成设备认证 token
// 使用 SHA256(SN + 时间戳 + 随机数) 算法生成唯一 token
// 参数 sn string: 设备序列号
// 返回 string: 生成的 token 字符串
func (r *DeviceRegisterRepo) GenerateAuthToken(sn string) string {
	timestamp := time.Now().UnixNano()
	randomNum := rand.Int63()

	data := fmt.Sprintf("%s%d%d", sn, timestamp, randomNum)
	hash := sha256.Sum256([]byte(data))

	return fmt.Sprintf("%x", hash)
}

// DeviceRegisterInfo 设备注册信息结构体
// 用于返回设备注册相关的数据
type DeviceRegisterInfo struct {
	ID              int64  `json:"id"`
	Sn              string `json:"sn"`
	Model           string `json:"model"`
	FirmwareVersion string `json:"firmware_version"`
	AuthToken       string `json:"auth_token"`
}

// VerifyToken 根据 SN 和 Token 验证设备身份
// 参数 ctx context.Context: 请求上下文
// 参数 sn string: 设备序列号
// 参数 token string: 认证 token
// 返回 *DeviceRegisterInfo: 设备注册信息指针，如果验证失败则返回 nil
// 返回 error: 验证失败时的错误信息
func (r *DeviceRegisterRepo) VerifyToken(ctx context.Context, sn string, token string) (*DeviceRegisterInfo, error) {
	query := `
		SELECT id, sn, model, firmware_version, auth_token
		FROM device
		WHERE sn = $1 AND auth_token = $2 AND status = 1
	`

	var info DeviceRegisterInfo
	err := r.db.QueryRowContext(ctx, query, sn, token).Scan(
		&info.ID, &info.Sn, &info.Model, &info.FirmwareVersion, &info.AuthToken,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("验证设备 token 失败: %v", err)
	}

	return &info, nil
}

// UpdateOnlineStatus 更新设备在线状态
// 参数 ctx context.Context: 请求上下文
// 参数 sn string: 设备序列号
// 返回 error: 更新失败时的错误信息
func (r *DeviceRegisterRepo) UpdateOnlineStatus(ctx context.Context, sn string) error {
	query := `
		UPDATE device
		SET online_status = 1, updated_at = NOW()
		WHERE sn = $1
	`

	_, err := r.db.ExecContext(ctx, query, sn)
	if err != nil {
		return fmt.Errorf("更新设备在线状态失败: %v", err)
	}

	return nil
}

// IsOnline 查询设备在线状态
// 参数 ctx context.Context: 请求上下文
// 参数 sn string: 设备序列号
// 返回 bool: 设备是否在线
// 返回 error: 查询失败时的错误信息
func (r *DeviceRegisterRepo) IsOnline(ctx context.Context, sn string) (bool, error) {
	query := `
		SELECT online_status
		FROM device
		WHERE sn = $1 AND status = 1
	`

	var onlineStatus int16
	err := r.db.QueryRowContext(ctx, query, sn).Scan(&onlineStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("查询设备在线状态失败: %v", err)
	}

	return onlineStatus == 1, nil
}
