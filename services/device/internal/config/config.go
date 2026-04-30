// Package config 包含设备服务的所有配置结构体定义
// 用于定义服务启动时从 YAML 配置文件加载的各项配置参数
package config

import "github.com/zeromicro/go-zero/rest"

// Config 设备服务的主配置结构体
// 包含 REST 服务配置、认证配置、数据库配置、Redis 配置等所有核心配置项
type Config struct {
	rest.RestConf
	Auth struct {
		AccessSecret string
		AccessExpire int64
	}
	DeviceAuth DeviceAuth `json:",optional"`
	Postgres   struct {
		DataSource string
	}
	Redis             RedisConf         `json:",optional"`
	DeviceShadow      DeviceShadow      `json:",optional"`
	DeviceStatusQuery DeviceStatusQuery `json:",optional"`
	// DeviceRegister MQTT/HTTP 基址与签名校验信任代理等（本服务已不提供 POST /register）。
	DeviceRegister DeviceRegister `json:",optional"`
	// MqttIngest 订阅 device/{sn}/report，device_secret + SN 鉴权后写影子（与 HTTP 上报一致）。
	MqttIngest MqttIngest `json:",optional"`
	// RedisKeyspace 监听 device:online:* 过期 → 影子离线 + Pub/Sub + 异步落库（需 Redis notify-keyspace-events 含 E）。
	RedisKeyspace RedisKeyspace `json:",optional"`
	// StatusPersist 异步写 device_status 与更新 device 主表（通道 + worker）。
	StatusPersist StatusPersist `json:",optional"`
	// MaxDeviceBinds 用户最大绑定设备数，bind 接口使用，默认 10。
	MaxDeviceBinds int `json:",default=10"`
	// DeviceCommand 指令域配置：默认过期时间、timeout、重试与 worker 扫描。
	DeviceCommand DeviceCommand `json:",optional"`
	// StatusReport POST /report/status 响应 next_interval：正常/中间档/节能/紧急及阈值（秒）。
	StatusReport StatusReport `json:",optional"`
	// ShadowMQTT 影子下发 MQTT 主题：兼容 legacy device/{sn}/desired 与 device/shadow/{sn|id}/… 双写。
	ShadowMQTT ShadowMQTT `json:",optional"`
	// StatusReportHTTP POST /api/device/status/report：按设备 SN 限流窗口内最大次数。
	StatusReportHTTP StatusReportHTTP `json:",optional"`
}

// StatusReportHTTP 设备 HTTP 状态上报限流配置结构体
// 用于配置设备通过 HTTP 接口上报状态时的限流策略，防止设备频繁上报
type StatusReportHTTP struct {
	// RateLimitPerMinute 单设备每自然分钟最大请求数；<=0 时默认 60
	RateLimitPerMinute int `json:",optional"`
}

// ShadowMQTT 设备影子 MQTT 发布配置结构体
// 用于配置设备影子通过 MQTT 下发到设备时的主题模板和发布策略
type ShadowMQTT struct {
	EnableLegacyTopics bool `json:",optional"`
	// LegacyDesiredTopic 默认 device/{sn}/desired，兼容旧版设备订阅
	LegacyDesiredTopic string `json:",optional"`
	// PublishShadowBySN / PublishShadowByID 是否向新层级各发一份（与 legacy 并行）
	PublishShadowBySN    bool   `json:",optional"`
	ShadowDesiredTopicSN string `json:",optional"` // 默认 device/shadow/{sn}/desired
	PublishShadowByID    bool   `json:",optional"`
	ShadowDesiredTopicID string `json:",optional"` // 默认 device/shadow/{id}/desired
	// 可选：与 desired 同载荷再发到 command 主题（空模板则跳过）
	ShadowCommandTopicSN string `json:",optional"`
	ShadowCommandTopicID string `json:",optional"`
	// PublishJSONDelta 为 true 时向 desired/delta 主题发纯 JSON delta（无 shadow_delta 信封）
	PublishJSONDelta   bool   `json:",optional"`
	ShadowDeltaTopicSN string `json:",optional"` // 默认 device/shadow/{sn}/desired/delta
	ShadowDeltaTopicID string `json:",optional"` // 默认 device/shadow/{id}/desired/delta
	// DesiredPublishQOS / DeltaPublishQOS：0–2，未配置时由代码默认 1
	DesiredPublishQOS int `json:",optional"`
	DeltaPublishQOS   int `json:",optional"`
}

// StatusReport 设备状态上报间隔策略配置
// 根据设备电量和使用状态动态调整上报频率，实现节能和实时监控的平衡
type StatusReport struct {
	IntervalNormalSec      int `json:",optional"` // 默认 60，充电恢复与正常电量
	IntervalMidRangeSec    int `json:",optional"` // 默认 120，约 20%–50% 未充电
	IntervalEnergySec      int `json:",optional"` // 默认 300，低电未充电节能
	IntervalEmergencySec   int `json:",optional"` // 默认 10，紧急模式
	IntervalMinSec         int `json:",optional"` // 默认 10，全局下限
	IntervalMaxSec         int `json:",optional"` // 默认 300，全局上限
	EnergyBatteryPercent   int `json:",optional"` // 默认 20，低于此且未充电走节能间隔
	MidRangeBatteryPercent int `json:",optional"` // 默认 50，低于此（且 ≥Energy 阈值）走中间档
}

// RedisConf 设备服务 Redis 配置结构体
// 用于配置设备影子、在线状态 TTL 等功能的 Redis 连接参数
type RedisConf struct {
	Addr     string `json:",optional"`
	Password string `json:",optional"`
	DB       int    `json:",optional"`
}

// DeviceShadow 设备影子配置结构体
// 用于配置设备影子的 Key TTL、在线集合与 Pub/Sub 等功能
type DeviceShadow struct {
	HeartbeatTTLSeconds int    `json:",optional"`
	EnableOnlineSet     bool   `json:",optional"`
	PubSubEnabled       bool   `json:",optional"`
	EventChannelPrefix  string `json:",optional"`
	SeedTTLSeconds      int    `json:",optional"`
}

// DeviceStatusQuery App 查询设备状态配置结构体
// 用于配置在线判定窗口、负缓存、按用户限流等功能
type DeviceStatusQuery struct {
	// OnlineStaleSeconds 距 last_update/last_active 超过该秒数视为离线（真实在线判定），默认 300
	OnlineStaleSeconds int `json:",optional"`
	// DisableNegCache 为 true 时关闭「无绑定」负缓存
	DisableNegCache bool `json:",optional"`
	// NegCacheTTLSeconds 无绑定负缓存 TTL（秒）；未配置且未关闭时默认 60
	NegCacheTTLSeconds  int  `json:",optional"`
	RateLimitWindowSec  int  `json:",optional"`
	RateLimitMaxPerUser int  `json:",optional"`
	DisableRateLimit    bool `json:",optional"`
}

// DeviceRegister 设备接入侧配置（已移除本服务的 POST /register；字段仍用于 /auth 响应中的接入地址、签名校验信任代理等）
type DeviceRegister struct {
	MqttBroker                 string            `json:",optional"`
	HttpBaseUrl                string            `json:",optional"`
	AllowedProductKeys         []string          `json:",optional"` // 保留字段；自助注册接口已下线
	BlacklistSNs               []string          `json:",optional"`
	SecretBcryptCost           int               `json:",optional"`
	TrustedProxies             []string          `json:",optional"`
	RateLimitWindowSec         int               `json:",optional"` // 已不再用于本服务注册限流，可忽略
	RateLimitMaxPerIP          int               `json:",optional"`
	DisableRegisterIPRateLimit bool              `json:",optional"`
	ProductModelByKey          map[string]string `json:",optional"`
	SnPattern                  string            `json:",optional"`
	ProductKeyPattern          string            `json:",optional"`
	MacPattern                 string            `json:",optional"`
}

// DeviceAuth 设备认证配置结构体
// 用于配置设备云端认证的 Token 生成、过期时间、失败锁定等参数
type DeviceAuth struct {
	TokenSecret              string `json:",optional"`
	TokenExpireSeconds       int64  `json:",optional"`
	TimestampToleranceSecond int64  `json:",optional"`
	FailureThreshold         int    `json:",optional"`
	LockSeconds              int64  `json:",optional"`
}

// MqttIngest MQTT 消息消费端配置结构体（后端订阅）
// 用于配置 MQTT Broker 连接、订阅主题、QoS 等参数
type MqttIngest struct {
	Enabled        bool   `json:",optional"`
	Broker         string `json:",optional"`
	ClientID       string `json:",optional"`
	Username       string `json:",optional"`
	Password       string `json:",optional"`
	SubscribeTopic string `json:",optional"` // 默认 device/+/report；EMQX 共享订阅示例 $share/device-api/device/+/report
	QOS            int    `json:",optional"` // 0–2，默认 1
}

// RedisKeyspace Redis Keyspace 通知配置结构体
// 用于配置是否启用在线键过期监听功能
type RedisKeyspace struct {
	Enabled bool `json:",optional"`
}

// StatusPersist 状态持久化配置结构体
// 用于配置异步落库队列（MQTT 上报与 Redis 离线均走此池）
type StatusPersist struct {
	QueueSize int `json:",optional"`
	Workers   int `json:",optional"`
}

// DeviceCommand 设备指令配置结构体
// 用于配置指令的默认过期时间、超时时间、重试次数与 worker 扫描参数
type DeviceCommand struct {
	DefaultExpiresSeconds int `json:",optional"`
	DefaultTimeoutSeconds int `json:",optional"`
	DefaultMaxRetry       int `json:",optional"`
	DispatchBatchSize     int `json:",optional"`
	WorkerIntervalSeconds int `json:",optional"`
}
