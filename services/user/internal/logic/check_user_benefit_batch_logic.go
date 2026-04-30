package logic

import (
	"context"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/entitlementsvc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

const checkUserBenefitBatchMaxItems = 50

type CheckUserBenefitBatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckUserBenefitBatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckUserBenefitBatchLogic {
	return &CheckUserBenefitBatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CheckUserBenefitBatch 批量权益/额度校验（顺序与 items 一致）。
func (l *CheckUserBenefitBatchLogic) CheckUserBenefitBatch(req *types.CheckUserBenefitBatchReq) (*types.CheckUserBenefitBatchResp, error) {
	start := time.Now()
	userID := ctxuser.ParseUserID(l.ctx)
	if userID <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}
	if req == nil || len(req.Items) == 0 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "items 不能为空")
	}
	if len(req.Items) > checkUserBenefitBatchMaxItems {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "单次最多 50 条校验项")
	}
	if l.svcCtx.Entitlement == nil {
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}

	out := make([]types.CheckUserBenefitResp, 0, len(req.Items))
	for _, it := range req.Items {
		mode := strings.ToLower(strings.TrimSpace(it.Mode))
		if mode == "" {
			mode = entitlementsvc.ModeMembership
		}
		singleReq := &types.CheckUserBenefitReq{
			Mode:        mode,
			BenefitCode: it.BenefitCode,
			FeatureKey:  it.FeatureKey,
			Delta:       it.Delta,
		}
		if mode == entitlementsvc.ModeMembership && strings.TrimSpace(it.BenefitCode) == "" {
			out = append(out, types.CheckUserBenefitResp{
				UserId: userID, Mode: mode, Reason: "权益标识不能为空",
			})
			continue
		}
		if mode == entitlementsvc.ModeQuota && strings.TrimSpace(it.FeatureKey) == "" {
			out = append(out, types.CheckUserBenefitResp{
				UserId: userID, Mode: mode, Reason: "feature_key 不能为空",
			})
			continue
		}

		r, eerr := l.svcCtx.Entitlement.Check(l.ctx, userID, entitlementsvc.Input{
			Mode:        mode,
			BenefitCode: it.BenefitCode,
			FeatureKey:  it.FeatureKey,
			Delta:       int64(it.Delta),
		})
		if eerr != nil {
			l.Logger.Errorf("批量权益校验查询会员档案失败：%v", eerr)
			out = append(out, types.CheckUserBenefitResp{
				UserId:        userID,
				Mode:          mode,
				BenefitCode:   it.BenefitCode,
				FeatureKey:    it.FeatureKey,
				HasPermission: false,
				Reason:        "查询会员信息失败",
			})
			continue
		}
		out = append(out, *buildCheckUserBenefitResp(userID, singleReq, mode, r))
	}

	l.Infof("user_benefit_check_batch user_id=%d count=%d duration_ms=%d", userID, len(out), time.Since(start).Milliseconds())
	return &types.CheckUserBenefitBatchResp{Results: out}, nil
}

func buildCheckUserBenefitResp(userID int64, req *types.CheckUserBenefitReq, mode string, r *entitlementsvc.Result) *types.CheckUserBenefitResp {
	return &types.CheckUserBenefitResp{
		UserId:        userID,
		Mode:          mode,
		BenefitCode:   req.BenefitCode,
		FeatureKey:    req.FeatureKey,
		HasPermission: r.HasPermission,
		Reason:        r.Reason,
		LevelCode:     r.LevelCode,
		Remaining:     float64(r.Remaining),
	}
}
