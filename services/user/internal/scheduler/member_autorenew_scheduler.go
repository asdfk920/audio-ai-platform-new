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

type MemberAutoRenewScheduler struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	config config.MemberAutoRenewWorker
	task   *logic.MemberAutoRenewScanTask
	cron   *cron.Cron
	mu     sync.RWMutex
	running bool
}

func NewMemberAutoRenewScheduler(ctx context.Context, svcCtx *svc.ServiceContext, cfg config.MemberAutoRenewWorker) *MemberAutoRenewScheduler {
	return &MemberAutoRenewScheduler{
		ctx:    ctx,
		svcCtx: svcCtx,
		config: cfg,
		task:   logic.NewMemberAutoRenewScanTask(ctx, svcCtx, cfg),
		cron:   cron.New(cron.WithSeconds()),
	}
}

func (s *MemberAutoRenewScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}
	expr := s.config.CronExpr
	if expr == "" {
		expr = "0 0 3 * * *"
	}
	_, err := s.cron.AddFunc(expr, s.execute)
	if err != nil {
		return err
	}
	s.cron.Start()
	s.running = true
	logx.Infof("member auto_renew scan scheduler started: %s", expr)
	return nil
}

func (s *MemberAutoRenewScheduler) Stop() error {
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

func (s *MemberAutoRenewScheduler) execute() {
	n, err := s.task.Execute()
	if err != nil {
		logx.Errorf("member auto_renew scan failed: %v", err)
		return
	}
	if n > 0 {
		logx.Infof("member auto_renew scan candidates=%d at=%s", n, time.Now().Format(time.RFC3339))
	}
}
