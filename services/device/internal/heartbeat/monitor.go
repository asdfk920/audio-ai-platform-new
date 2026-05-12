package heartbeat

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jacklau/audio-ai-platform/services/device/internal/repository"
)

type HeartbeatMonitor struct {
	deviceRepo     *repository.DeviceRepo
	interval       time.Duration
	timeoutMinutes int
	stopCh         chan struct{}
	wg             sync.WaitGroup
	mu             sync.Mutex
	running        bool
}

func NewHeartbeatMonitor(deviceRepo *repository.DeviceRepo, interval time.Duration, timeoutMinutes int) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		deviceRepo:     deviceRepo,
		interval:       interval,
		timeoutMinutes: timeoutMinutes,
		stopCh:         make(chan struct{}),
	}
}

func (m *HeartbeatMonitor) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return
	}

	m.running = true
	m.wg.Add(1)
	go m.loop()
	slog.Info("心跳监控服务已启动", "interval", m.interval, "timeout_minutes", m.timeoutMinutes)
}

func (m *HeartbeatMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	close(m.stopCh)
	m.running = false
	m.wg.Wait()
	slog.Info("心跳监控服务已停止")
}

func (m *HeartbeatMonitor) loop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkAndMarkOfflineDevices()
		case <-m.stopCh:
			return
		}
	}
}

func (m *HeartbeatMonitor) checkAndMarkOfflineDevices() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	affected, err := m.deviceRepo.UpdateOfflineDevices(ctx, m.timeoutMinutes)
	if err != nil {
		slog.Error("心跳超时检测失败", "error", err)
		return
	}

	if affected > 0 {
		slog.Info("心跳超时检测完成", "marked_offline_count", affected, "timeout_minutes", m.timeoutMinutes)
	}
}
