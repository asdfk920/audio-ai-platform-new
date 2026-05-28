// Package model 包含设备服务的所有数据模型定义
// 用于定义数据库表结构、状态常量等
package model

import (
	"time"
)

// Device 设备数据模型结构体
// 对应数据库中的 device 表，存储设备的基本信息、状态、版本等
type Device struct {
	ID                int64      `db:"id"`
	Sn                string     `db:"sn"`
	Model             string     `db:"model"`
	ProductKey        string     `db:"product_key"`
	DeviceSecret      string     `db:"device_secret"`
	RegisterSignature *string    `db:"register_signature"` // 设备注册签名：HMAC-SHA256(device_secret, sn + timestamp)，可为NULL
	RegisterTimestamp *int64     `db:"register_timestamp"` // 注册时间戳（毫秒级Unix时间戳），用于WebSocket认证签名验证
	FirmwareVersion   string     `db:"firmware_version"`
	HardwareVersion   string     `db:"hardware_version"`
	Mac               string     `db:"mac"`
	DeviceNameRaw     string     `db:"device_name_raw"` // 设备原始名称（出厂或注册上报）
	Ip                string     `db:"ip"`
	OnlineStatus      int16      `db:"online_status"`
	UsageStatus       int16      `db:"usage_status"`
	Status            int16      `db:"status"`
	CreateBy          int64      `db:"create_by"`
	LastActiveAt      *time.Time `db:"last_active_at"` // 可为 NULL（预录入/未上线设备）
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"`
	DeletedAt         *time.Time `db:"deleted_at"`
}

// DeviceStatus 设备状态常量定义
// 用于标识设备的生命周期状态
// 状态值：0=Default, 1=Normal, 2=Disabled, 3=Inactive, 4=Unregistered, 5=Unauthenticated
const (
	DeviceStatusDefault         int16 = 0 // 默认：初始状态（兼容旧数据）
	DeviceStatusNormal          int16 = 1 // 正常：设备已注册且已认证，可正常使用
	DeviceStatusDisabled        int16 = 2 // 禁用：设备被管理员禁用
	DeviceStatusInactive        int16 = 3 // 未激活/报废：设备已停用或报废
	DeviceStatusUnregistered    int16 = 4 // 未注册：设备刚创建，尚未完成注册流程
	DeviceStatusUnauthenticated int16 = 5 // 未认证：设备已注册但尚未完成WebSocket认证
)

// DeviceOnlineStatus 设备在线状态常量定义
// 用于标识设备的在线/离线状态
const (
	DeviceOnlineStatusOffline int16 = 0 // 离线：设备未连接云端
	DeviceOnlineStatusOnline  int16 = 1 // 在线：设备已连接云端
)

// DeviceUsageStatus 设备使用状态常量定义
// 用于标识设备的启用/禁用状态（管理员控制）
// 状态值：1=启用, 2=禁用
const (
	DeviceUsageStatusEnabled  int16 = 1 // 启用：设备可正常使用
	DeviceUsageStatusDisabled int16 = 2 // 禁用：设备被禁用，无法使用
)
