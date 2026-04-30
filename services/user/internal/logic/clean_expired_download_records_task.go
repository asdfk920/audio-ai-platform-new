package logic

import (
	"context"
	"fmt"

	"github.com/jacklau/audio-ai-platform/services/user/internal/config"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

// CleanExpiredDownloadRecordsTask 清理过期下载记录定时任务
// 功能：根据用户会员等级，定时清理超过保留期限的下载记录
// 清理规则：
//   - 免费版（ordinary）：保留 7 天
//   - 标准版（vip/year_vip）：保留 365 天
//   - 专业版（svip）：永久保留，不清理
type CleanExpiredDownloadRecordsTask struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	config config.DownloadRecordCleanWorker
}

// NewCleanExpiredDownloadRecordsTask 创建清理过期下载记录任务实例
// 参数：
//   - ctx: 请求上下文
//   - svcCtx: 服务上下文，包含数据库连接等依赖
//   - cfg: 定时任务配置（保留天数、批次大小等）
func NewCleanExpiredDownloadRecordsTask(ctx context.Context, svcCtx *svc.ServiceContext, cfg config.DownloadRecordCleanWorker) *CleanExpiredDownloadRecordsTask {
	return &CleanExpiredDownloadRecordsTask{
		ctx:    ctx,
		svcCtx: svcCtx,
		config: cfg,
	}
}

// Execute 执行清理过期下载记录任务
// 流程：
//   1. 获取配置中的保留天数（免费版默认 7 天，标准版默认 365 天）
//   2. 获取批次大小配置（默认 1000 条/批）
//   3. 调用 Repository 层执行批量清理
//   4. 记录清理日志
//
// 返回：
//   - int64: 实际清理的记录总数
//   - error: 执行失败时返回错误信息
func (t *CleanExpiredDownloadRecordsTask) Execute() (int64, error) {
	freeRetentionDays := t.config.FreeRetentionDays
	if freeRetentionDays <= 0 {
		freeRetentionDays = 7
	}

	standardRetentionDays := t.config.StandardRetentionDays
	if standardRetentionDays <= 0 {
		standardRetentionDays = 365
	}

	batchSize := t.config.BatchSize
	if batchSize <= 0 {
		batchSize = 1000
	}

	logx.Infof("开始清理过期下载记录: 免费版保留%d天, 标准版保留%d天, 批次大小=%d",
		freeRetentionDays, standardRetentionDays, batchSize)

	cleaned, err := t.svcCtx.DownloadRecord.CleanExpiredDownloadRecords(
		t.ctx,
		freeRetentionDays,
		standardRetentionDays,
		batchSize,
	)
	if err != nil {
		return 0, fmt.Errorf("清理过期下载记录失败: %v", err)
	}

	if cleaned > 0 {
		logx.Infof("清理过期下载记录完成: 共清理 %d 条记录", cleaned)
	} else {
		logx.Info("清理过期下载记录完成: 无过期记录")
	}

	return cleaned, nil
}
