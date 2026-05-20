// Package svc 服务上下文包
// 提供 ServiceContext 结构体，用于注入配置、数据库连接、Redis 连接等依赖
// 是 MQTT 上报、Command Worker、Redis 监听等后台逻辑的依赖入口
package svc

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/config"
	"github.com/jacklau/audio-ai-platform/services/device/internal/heartbeat"
	"github.com/jacklau/audio-ai-platform/services/device/internal/pkg/ip"
	"github.com/jacklau/audio-ai-platform/services/device/internal/rabbitmq"
	"github.com/jacklau/audio-ai-platform/services/device/internal/repository"
)

// ServiceContext 设备进程上下文：配置、DB、Redis、仓储等。
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
	DeviceShadowRepo   *repository.DeviceShadowRepo

	// HeartbeatMonitor 心跳超时检测定时任务（双重保障机制）
	HeartbeatMonitor *heartbeat.HeartbeatMonitor

	// WsPushJSON 将即时指令推到设备 WebSocket（由 main 注入 logic.SendCmdToDevice）。
	// MQTT 下线后 commandsvc.DispatchPendingInstructions 依赖此路径投递在线设备。
	WsPushJSON func(deviceIDStr string, payload interface{}) error `json:"-"`

	// RegisterTrustedNets HTTP 路径下线后仍可用于将来接入层解析 XFF（与 DeviceRegister.TrustedProxies 一致）。
	RegisterTrustedNets []*net.IPNet

	// RabbitMQMgr RabbitMQ 消息队列管理器（指令异步下发、削峰填谷）
	RabbitMQMgr *rabbitmq.Manager
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

	svcCtx := &ServiceContext{
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
		DeviceShadowRepo:    repository.NewDeviceShadowRepo(db),
		HeartbeatMonitor:    heartbeatMonitor,
		RegisterTrustedNets: trusted,
	}

	if c.RabbitMQ.URL != "" {
		rabbitMgr := rabbitmq.NewManager(context.Background(), c.RabbitMQ)
		svcCtx.RabbitMQMgr = rabbitMgr
		logx.Infof("[ServiceContext] RabbitMQ manager created (URL configured)")
	}

	return svcCtx
}

func (s *ServiceContext) InitRabbitMQ() error {
	if s.RabbitMQMgr == nil {
		logx.Infof("[ServiceContext] RabbitMQ not configured, skipping initialization")
		return nil
	}
	logx.Infof("[ServiceContext] Initializing RabbitMQ connection...")
	if err := s.RabbitMQMgr.Connect(); err != nil {
		return fmt.Errorf("failed to connect RabbitMQ: %w", err)
	}
	logx.Infof("[ServiceContext] ✅ RabbitMQ connected and ready")
	return nil
}
