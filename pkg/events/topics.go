// Package events 定义跨服务 Kafka/Redpanda 主题名，避免魔法字符串分散在各微服务。
// 接入时：由各服务在启动时 EnsureTopic 或依赖 broker 自动建主题（仅开发环境）。
package events

const (
	// DeviceStatus 设备在线/离线、固件版本等状态变更（可映射到设备影子写 Redis / PG）
	DeviceStatusV1 = "device.status.v1"
	// DeviceCommandV1 云端经 MQTT 网关下发指令前的可选编排主题（按需使用）
	DeviceCommandV1 = "device.command.v1"
	// UserAuditV1 登录、改密、换绑等审计事件，供统计或 ELK 管道消费
	UserAuditV1 = "user.audit.v1"
	// OTAPackageV1 OTA 元数据或任务通知（与设备 MQTT 通道配合）
	OTAPackageV1 = "ota.package.v1"
)
