// Package model 包含设备服务的所有数据模型定义
// 用于定义数据库表结构、状态常量等
package model

import (
	"time"
)

// Device 设备数据模型结构体
// 对应数据库中的 device 表，存储设备的基本信息、状态、版本等
type Device struct {
	ID              int64      `db:"id"`
	Sn              string     `db:"sn"`
	Model           string     `db:"model"`
	ProductKey      string     `db:"product_key"`
	FirmwareVersion string     `db:"firmware_version"`
	HardwareVersion string     `db:"hardware_version"`
	Mac             string     `db:"mac"`
	Ip              string     `db:"ip"`
	OnlineStatus    int16      `db:"online_status"`
	Status          int16      `db:"status"`
	UserID          int64      `db:"user_id"`
	LastHeartbeat   time.Time  `db:"last_heartbeat"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
	DeletedAt       *time.Time `db:"deleted_at"`
}

// DeviceStatus 设备状态常量定义
// 用于标识设备的生命周期状态（正常/禁用/报废）
const (
	DeviceStatusNormal   int16 = 1 // 正常：设备可正常使用
	DeviceStatusDisabled int16 = 0 // 禁用：设备被管理员禁用
	DeviceStatusScrapped int16 = 2 // 报废：设备已报废
)

// DeviceOnlineStatus 设备在线状态常量定义
// 用于标识设备的在线/离线状态
const (
	DeviceOnlineStatusOffline int16 = 0 // 离线：设备未连接云端
	DeviceOnlineStatusOnline  int16 = 1 // 在线：设备已连接云端
)