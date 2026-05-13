// Package svc 服务上下文包
// 提供 ServiceContext 结构体，用于注入配置、数据库连接、Redis 连接等依赖
// 是 MQTT 上报、Command Worker、Redis 监听等后台逻辑的依赖入口
package svc

import (
	"database/sql"
	"net"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/pkg/mqttx"
	"github.com/jacklau/audio-ai-platform/services/device/internal/config"
	"github.com/jacklau/audio-ai-platform/services/device/internal/heartbeat"
	"github.com/jacklau/audio-ai-platform/services/device/internal/pkg/ip"
	"github.com/jacklau/audio-ai-platform/services/device/internal/repository"
)

// ServiceContext 设备进程上下文：配置、DB、Redis、仓储与 MQTT 客户端等。
type ServiceContext struct {
	Config config.Config
	DB     *sql.DB
	Redis  *redis.Client

	DeviceRepo         *repository.DeviceRepo
	UserDeviceBindRepo *repository.UserDeviceBindRepo
	PlaylistRepo       *repository.PlaylistRepo
	PlaylistItemRepo   *repository.PlaylistItemRepo
	AudioResourceRepo  *repository.AudioResourceRepo
	ContentFileRepo    *repository.ContentFileRepo
	DeviceRegister     *repository.DeviceRegisterRepo
	DeviceTopicACLRepo *repository.DeviceTopicACLRepo
	DeviceShadowRepo   *repository.DeviceShadowRepo

	// HeartbeatMonitor 心跳超时检测定时任务（双重保障机制）
	HeartbeatMonitor *heartbeat.HeartbeatMonitor

	// RegisterTrustedNets MQTT/HTTP 路径下线后仍可用于将来接入层解析 XFF（与 DeviceRegister.TrustedProxies 一致）。
	RegisterTrustedNets []*net.IPNet

	mqttMu     sync.RWMutex
	mqttClient *mqttx.Client
}

// NewServiceContext 创建并初始化服务上下文实例
func NewServiceContext(c config.Config, db *sql.DB, rdb *redis.Client) *ServiceContext {
	trusted, err := ip.ParseTrustedProxies(c.DeviceRegister.TrustedProxies)
	if err != nil {
		logx.Errorf("DeviceRegister.TrustedProxies invalid, using strict RemoteAddr only: %v", err)
		trusted = nil
	}

	deviceRepo := repository.NewDeviceRepo(db)

	heartbeatTimeoutMinutes := 5 // 默认5分钟超时（2倍KeepAlive时间）
	heartbeatMonitor := heartbeat.NewHeartbeatMonitor(
		deviceRepo,
		1*time.Minute,           // 每分钟检测一次
		heartbeatTimeoutMinutes, // 超过5分钟未活跃则标记离线
	)

	return &ServiceContext{
		Config:              c,
		DB:                  db,
		Redis:               rdb,
		DeviceRepo:          deviceRepo,
		UserDeviceBindRepo:  repository.NewUserDeviceBindRepo(db),
		PlaylistRepo:        repository.NewPlaylistRepo(db),
		PlaylistItemRepo:    repository.NewPlaylistItemRepo(db),
		AudioResourceRepo:   repository.NewAudioResourceRepo(db),
		ContentFileRepo:     repository.NewContentFileRepo(db),
		DeviceRegister:      repository.NewDeviceRegisterRepo(db),
		DeviceTopicACLRepo:  repository.NewDeviceTopicACLRepo(db),
		DeviceShadowRepo:    repository.NewDeviceShadowRepo(db),
		HeartbeatMonitor:    heartbeatMonitor,
		RegisterTrustedNets: trusted,
	}
}

func (s *ServiceContext) SetMQTTClient(client *mqttx.Client) {
	if s == nil {
		return
	}
	s.mqttMu.Lock()
	defer s.mqttMu.Unlock()
	s.mqttClient = client
}

func (s *ServiceContext) MQTTClient() *mqttx.Client {
	if s == nil {
		return nil
	}
	s.mqttMu.RLock()
	defer s.mqttMu.RUnlock()
	return s.mqttClient
}
