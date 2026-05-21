package logic

import "time"

const (
	DeviceOnlineStatusOffline   = 0 // 离线（offline）
	DeviceOnlineStatusOnline    = 1 // 在线（online）
	DeviceOnlineStatusRebooting = 2 // 重启中（rebooting）
)

// RebootTimeoutDuration 设备重启响应超时时间
// 设备收到reboot指令后，必须在此时长内返回ACK，否则强制断开连接
// 原值：30秒（太长，导致大量超时日志）
// 新值：5秒（快速失败原则，设备正常情况下1-2秒内就能响应）
var RebootTimeoutDuration = 5 * time.Second

var DeviceOnlineStatusMap = map[int]string{
	DeviceOnlineStatusOffline:   "offline",
	DeviceOnlineStatusOnline:    "online",
	DeviceOnlineStatusRebooting: "rebooting",
}

var DeviceOnlineStatusNameMap = map[string]int{
	"offline":   DeviceOnlineStatusOffline,
	"online":    DeviceOnlineStatusOnline,
	"rebooting": DeviceOnlineStatusRebooting,
}
