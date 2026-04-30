package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/jacklau/audio-ai-platform/services/user/internal/config"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/robfig/cron/v3"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeviceShareScheduler struct {
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	config  config.DeviceShareWorker
	task    *logic.DeviceShareExpireTask
	cron    *cron.Cron
	mu      sync.RWMutex
	running bool
}

func NewDeviceShareScheduler(ctx context.Context, svcCtx *svc.ServiceContext, cfg config.DeviceShareWorker) *DeviceShareScheduler {
	return &DeviceShareScheduler{
		ctx:    ctx,
		svcCtx: svcCtx,
		config: cfg,
		task:   logic.NewDeviceShareExpireTask(ctx, svcCtx),
		cron:   cron.New(cron.WithSeconds()),
	}
}

func (s *DeviceShareScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}
	expr := s.config.ExpireCronExpr
	if expr == "" {
		expr = "0 */5 * * * *"
	}
	_, err := s.cron.AddFunc(expr, s.execute)
	if err != nil {
		return err
	}
	s.cron.Start()
	s.running = true
	logx.Infof("device share expire scheduler started: %s", expr)
	return nil
}

func (s *DeviceShareScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false
	return nil
}

func (s *DeviceShareScheduler) execute() {
	batch := s.config.BatchSize
	if batch <= 0 {
		batch = 100
	}
	count, err := s.task.Execute(batch)
	if err != nil {
		logx.Errorf("device share expire scheduler failed: %v", err)
		return
	}
	if count > 0 {
		logx.Infof("device share expire scheduler expired=%d at=%s", count, time.Now().Format(time.RFC3339))
	}
}
