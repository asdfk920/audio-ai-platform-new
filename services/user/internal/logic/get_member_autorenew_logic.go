package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetMemberAutoRenewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMemberAutoRenewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMemberAutoRenewLogic {
	return &GetMemberAutoRenewLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetMemberAutoRenewLogic) GetMemberAutoRenew() (resp *types.MemberAutoRenewInfoResp, err error) {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID == 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "未登录")
	}
	m, err := l.svcCtx.MemberOrder.GetUserMemberInfo(l.ctx, userID)
	if err != nil {
		logx.Errorf("get member row: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}
	if m == nil {
		return &types.MemberAutoRenewInfoResp{AutoRenew: false}, nil
	}
	out := &types.MemberAutoRenewInfoResp{AutoRenew: m.AutoRenew == 1, AutoRenewPackageCode: m.AutoRenewPackageCode}
	if m.AutoRenewPayType > 0 {
		out.AutoRenewPayType = int64(m.AutoRenewPayType)
	}
	if m.AutoRenewUpdatedAt.Valid {
		out.AutoRenewUpdatedAt = m.AutoRenewUpdatedAt.Time.Unix()
	}
	return out, nil
}
