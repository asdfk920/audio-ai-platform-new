// Package repository 包含设备注册的数据访问层（Repository）实现
// 负责设备注册相关的数据库操作
package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
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

// operationalDeviceStatusesSQL 生命周期中「可操作」的设备：排除禁用(2)、停用(3)、未注册(4)。
// 包含 正常(1)、待 WS 认证(5)、遗留默认(0)，与 DeviceRepo.FindBySn 可查到的有效行一致，
// 避免 HTTP 等业务仍用 FindBySn 时仅认 status=1 导致「WS 已成功但下发指令报设备不存在」。
const operationalDeviceStatusesSQL = `
  deleted_at IS NULL AND status NOT IN (2, 3, 4)
`

// FindBySn 根据设备 SN 查询设备是否已注册
// 参数 ctx context.Context: 请求上下文
// 参数 sn string: 设备序列号
// 返回 *DeviceRegisterInfo: 设备注册信息指针，如果未找到则返回 nil
// 返回 error: 查询失败时的错误信息
func (r *DeviceRegisterRepo) FindBySn(ctx context.Context, sn string) (*DeviceRegisterInfo, error) {
	query := `
		SELECT id, sn, device_secret
		FROM public.device
		WHERE sn = $1 AND ` + operationalDeviceStatusesSQL + `
	`

	var info DeviceRegisterInfo
	err := r.db.QueryRowContext(ctx, query, sn).Scan(
		&info.ID, &info.Sn, &info.Secret,
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
	randomNum := mathrand.Int63()

	data := fmt.Sprintf("%s%d%d", sn, timestamp, randomNum)
	hash := sha256.Sum256([]byte(data))

	return fmt.Sprintf("%x", hash)
}

// DeviceRegisterInfo 设备注册信息结构体
// 用于返回设备注册相关的数据
type DeviceRegisterInfo struct {
	ID     int64  `json:"id"`
	Sn     string `json:"sn"`
	Secret string `json:"secret"` // 设备密钥（device_secret）
}

// CreateDeviceWithSecret 创建新设备记录并设置密钥（新设备注册）
// 参数 ctx context.Context: 请求上下文
// 参数 sn string: 设备序列号
// 参数 deviceSecret string: 设备密钥（32位随机字符串）
// 返回 int64: 新创建的设备 ID
// 返回 error: 创建失败时的错误信息
func (r *DeviceRegisterRepo) CreateDeviceWithSecret(ctx context.Context, sn string, deviceSecret string) (int64, error) {
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(deviceSecret), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("加密设备密钥失败: %v", err)
	}

	query := `
		INSERT INTO device (sn, product_key, device_secret, status, online_status, created_at, updated_at)
		VALUES ($1, $2, $3, 1, 0, NOW(), NOW())
		RETURNING id
	`

	productKey := fmt.Sprintf("PK_%s", sn)

	var deviceID int64
	err = r.db.QueryRowContext(ctx, query, sn, productKey, string(hashedSecret)).Scan(&deviceID)
	if err != nil {
		return 0, fmt.Errorf("创建设备记录失败: %v", err)
	}

	return deviceID, nil
}

// GenerateDeviceSecret 生成32位随机设备密钥
// 使用加密安全的随机数生成器生成字母+数字组合的密钥
//
// 返回 string: 32位随机字符串（示例：a1b2c3d4e5f67890abcdef1234567890）
func (r *DeviceRegisterRepo) GenerateDeviceSecret() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const secretLength = 32

	result := make([]byte, secretLength)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			timestamp := time.Now().UnixNano()
			return fmt.Sprintf("%s%x", fmt.Sprintf("%032d", timestamp), timestamp)
		}
		result[i] = charset[num.Int64()]
	}

	return string(result)
}

// VerifyToken 根据 SN 和 Token 验证设备身份
// 参数 ctx context.Context: 请求上下文
// 参数 sn string: 设备序列号
// 参数 token string: 认证 token
// 返回 *DeviceRegisterInfo: 设备注册信息指针，如果验证失败则返回 nil
// 返回 error: 验证失败时的错误信息
func (r *DeviceRegisterRepo) VerifyToken(ctx context.Context, sn string, token string) (*DeviceRegisterInfo, error) {
	logx.Infof("[DEBUG] 开始验证Token(明文模式): sn=%s, token=%s", sn, token)

	query := `
		SELECT id, sn, device_secret
		FROM public.device
		WHERE sn = $1 AND ` + operationalDeviceStatusesSQL + `
	`

	var info DeviceRegisterInfo
	err := r.db.QueryRowContext(ctx, query, sn).Scan(
		&info.ID, &info.Sn, &info.Secret,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			logx.Slowf("[DEBUG] 未找到设备记录或设备已禁用: sn=%s", sn)
			return nil, nil
		}
		logx.Errorf("[DEBUG] 查询数据库失败: sn=%s, error=%v", sn, err)
		return nil, fmt.Errorf("查询设备信息失败: %v", err)
	}

	logx.Infof("[DEBUG] 找到设备: id=%d, sn=%s, secret=%s", info.ID, info.Sn, info.Secret)

	if info.Secret == "" {
		logx.Errorf("[DEBUG] 设备密钥为空: sn=%s", sn)
		return nil, fmt.Errorf("设备密钥为空: %s", sn)
	}

	if info.Secret != token {
		logx.Slowf("[DEBUG] Token不匹配: 输入=%s, 数据库=%s", token, info.Secret)
		return nil, nil
	}

	logx.Infof("[DEBUG] Token验证成功(明文匹配): sn=%s, id=%d", sn, info.ID)
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
		FROM public.device
		WHERE sn = $1 AND ` + operationalDeviceStatusesSQL + `
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
