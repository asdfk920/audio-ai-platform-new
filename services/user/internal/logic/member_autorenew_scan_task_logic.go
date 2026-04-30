package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/user/internal/config"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

type MemberAutoRenewScanTask struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	cfg    config.MemberAutoRenewWorker
}

func NewMemberAutoRenewScanTask(ctx context.Context, svcCtx *svc.ServiceContext, cfg config.MemberAutoRenewWorker) *MemberAutoRenewScanTask {
	return &MemberAutoRenewScanTask{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		cfg:    cfg,
	}
}

// Execute 扫描即将到期且开启自动续费的用户，仅记录日志（MVP 不真实扣款）。
func (t *MemberAutoRenewScanTask) Execute() (int, error) {
	within := t.cfg.WithinDaysBeforeExpire
	if within <= 0 {
		within = 7
	}
	limit := t.cfg.BatchSize
	if limit <= 0 {
		limit = 200
	}
	list, err := t.svcCtx.MemberOrder.ListAutoRenewCandidates(t.ctx, within, limit)
	if err != nil {
		return 0, err
	}
	for _, c := range list {
		logx.WithContext(t.ctx).Infof(
			"member_autorenew_stub user_id=%d level=%s expire=%s pkg=%s pay_type=%d (no charge)",
			c.UserID, c.LevelCode, c.ExpireAt.Format("2006-01-02 15:04:05"), c.AutoRenewPackageCode, c.AutoRenewPayType,
		)
	}
	return len(list), nil
}
