// Package model 包含设备服务的所有数据模型定义
// 用于定义数据库表结构、状态常量等
package model

import (
	"time"
)

// UserDeviceBind 用户设备绑定关系数据模型结构体
// 对应数据库中的 user_device_bind 表，记录用户与设备的绑定关系
type UserDeviceBind struct {
	ID        int64      `db:"id"`
	UserID    int64      `db:"user_id"`
	DeviceID  int64      `db:"device_id"`
	SN        string     `db:"sn"`
	Status    int16      `db:"status"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

// UserDeviceBindStatus 用户设备绑定状态常量定义
// 用于标识用户与设备绑定关系的状态（正常/解绑）
const (
	UserDeviceBindStatusNormal int16 = 1 // 绑定正常：用户与设备已绑定
	UserDeviceBindStatusUnbind int16 = 0 // 已解绑：用户与设备已解除绑定
)