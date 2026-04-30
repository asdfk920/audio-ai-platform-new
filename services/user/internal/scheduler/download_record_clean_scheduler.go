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

// DownloadRecordCleanScheduler 下载记录清理定时任务调度器
// 功能：根据用户会员等级，定时清理超过保留期限的下载记录
// 清理规则：
//   - 免费版（ordinary）：保留 7 天
//   - 标准版（vip/year_vip）：保留 365 天
//   - 专业版（svip）：永久保留，不清理
type DownloadRecordCleanScheduler struct {
	ctx     context.Context
	svcCtx  *svc.ServiceContext
	config  config.DownloadRecordCleanWorker
	task    *logic.CleanExpiredDownloadRecordsTask
	cron    *cron.Cron
	mu      sync.RWMutex
	running bool
}

// NewDownloadRecordCleanScheduler 创建下载记录清理定时任务调度器
// 参数：
//   - ctx: 请求上下文
//   - svcCtx: 服务上下文，包含数据库连接等依赖
//   - cfg: 定时任务配置（cron 表达式、保留天数等）
func NewDownloadRecordCleanScheduler(ctx context.Context, svcCtx *svc.ServiceContext, cfg config.DownloadRecordCleanWorker) *DownloadRecordCleanScheduler {
	return &DownloadRecordCleanScheduler{
		ctx:    ctx,
		svcCtx: svcCtx,
		config: cfg,
		task:   logic.NewCleanExpiredDownloadRecordsTask(ctx, svcCtx, cfg),
		cron:   cron.New(cron.WithSeconds()),
	}
}

// Start 启动定时任务调度器
// 流程：
//  1. 获取 cron 表达式（默认每天凌晨 3 点执行）
//  2. 注册定时任务
//  3. 启动 cron 调度器
//
// 返回：
//   - error: 启动失败时返回错误信息
func (s *DownloadRecordCleanScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}

	expr := s.config.CronExpr
	if expr == "" {
		expr = "0 0 3 * * *" // 默认每天凌晨 3 点执行
	}

	_, err := s.cron.AddFunc(expr, s.execute)
	if err != nil {
		return err
	}

	s.cron.Start()
	s.running = true
	logx.Infof("download record clean scheduler started: %s", expr)
	return nil
}

// Stop 停止定时任务调度器
// 返回：
//   - error: 停止失败时返回错误信息
func (s *DownloadRecordCleanScheduler) Stop() error {
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

// execute 执行清理任务（被 cron 调用的方法）
// 流程：
//  1. 调用 Logic 层执行清理任务
//  2. 记录清理结果日志
func (s *DownloadRecordCleanScheduler) execute() {
	n, err := s.task.Execute()
	if err != nil {
		logx.Errorf("download record clean failed: %v", err)
		return
	}
	if n > 0 {
		logx.Infof("download record clean candidates=%d at=%s", n, time.Now().Format(time.RFC3339))
	}
}
