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

// CancellationScheduler 账号注销定时任务调度器
type CancellationScheduler struct {
	ctx           context.Context
	svcCtx        *svc.ServiceContext
	config        config.Cancellation
	cron          *cron.Cron
	task          *logic.AccountCancellationTask
	running       bool
	mu            sync.RWMutex
	lastRun       time.Time
	nextRun       time.Time
	totalRuns     int64
	totalExecuted int64
	totalFailed   int64
}

// NewCancellationScheduler 创建账号注销定时任务调度器
func NewCancellationScheduler(ctx context.Context, svcCtx *svc.ServiceContext, cfg config.Cancellation) *CancellationScheduler {
	return &CancellationScheduler{
		ctx:    ctx,
		svcCtx: svcCtx,
		config: cfg,
		cron:   cron.New(cron.WithSeconds()), // 支持秒级调度
		task:   logic.NewAccountCancellationTask(ctx, svcCtx),
	}
}

// Start 启动定时任务
func (s *CancellationScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// 解析定时任务配置（支持 cron 表达式）
	// 默认配置：每天凌晨 2 点执行
	// 格式：秒 分 时 日 月 周
	cronExpr := s.config.CleanupCronExpr
	if cronExpr == "" {
		cronExpr = "0 0 2 * * *" // 默认每天凌晨 2 点
	}

	_, err := s.cron.AddFunc(cronExpr, s.executeTask)
	if err != nil {
		logx.Errorf("添加账号注销定时任务失败：%v", err)
		return err
	}

	s.cron.Start()
	s.running = true
	s.nextRun = s.cron.Entries()[0].Next

	logx.Infof("账号注销定时任务已启动，cron 表达式=%s, 下次执行时间=%s",
		cronExpr, s.nextRun.Format(time.RFC3339))

	return nil
}

// Stop 停止定时任务
func (s *CancellationScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false

	logx.Infof("账号注销定时任务已停止，总执行次数=%d", s.totalRuns)
	return nil
}

// executeTask 执行注销任务（实际被 cron 调用的方法）
func (s *CancellationScheduler) executeTask() {
	s.mu.Lock()
	s.lastRun = time.Now()
	s.totalRuns++
	s.mu.Unlock()

	logx.Infof("开始执行账号注销定时任务 (第 %d 次运行)", s.totalRuns)

	// 执行注销任务
	// batchSize: 每批次处理 100 个用户，避免一次性处理过多
	batchSize := s.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100 // 默认值
	}

	total, executed, skipped, failed, err := s.task.ExecuteCancellationTask(batchSize)

	s.mu.Lock()
	s.totalExecuted += executed
	s.totalFailed += failed
	s.mu.Unlock()

	if err != nil {
		logx.Errorf("账号注销定时任务执行失败：%v", err)
		return
	}

	logx.Infof(
		"账号注销定时任务执行完成：总用户数=%d, 成功=%d, 跳过=%d, 失败=%d",
		total, executed, skipped, failed,
	)
}

// GetStatus 获取定时任务状态
func (s *CancellationScheduler) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"running":        s.running,
		"last_run":       s.lastRun.Format(time.RFC3339),
		"next_run":       s.nextRun.Format(time.RFC3339),
		"total_runs":     s.totalRuns,
		"total_executed": s.totalExecuted,
		"total_failed":   s.totalFailed,
	}

	// 获取待处理数量
	if s.svcCtx != nil {
		pendingCount, err := s.task.GetPendingCancellationsCount()
		if err == nil {
			status["pending_count"] = pendingCount
		}

		coolingCount, err := s.task.GetCoolingPeriodUsers()
		if err == nil {
			status["cooling_count"] = coolingCount
		}
	}

	return status
}

// TriggerNow 立即触发一次任务（用于测试或手动触发）
func (s *CancellationScheduler) TriggerNow() (total, executed, skipped, failed int64, err error) {
	if !s.running {
		return 0, 0, 0, 0, nil
	}

	logx.Infof("手动触发账号注销定时任务")

	batchSize := s.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	return s.task.ExecuteCancellationTask(batchSize)
}
