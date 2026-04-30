package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type SetMemberAutoRenewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetMemberAutoRenewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetMemberAutoRenewLogic {
	return &SetMemberAutoRenewLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *SetMemberAutoRenewLogic) SetMemberAutoRenew(req *types.SetMemberAutoRenewReq) (resp *types.MemberAutoRenewInfoResp, err error) {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID == 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "未登录")
	}
	if err := l.svcCtx.MemberOrder.UpdateUserMemberAutoRenew(l.ctx, userID, req.Enabled, req.PackageCode, int16(req.PayType)); err != nil {
		return nil, err
	}
	g := NewGetMemberAutoRenewLogic(l.ctx, l.svcCtx)
	return g.GetMemberAutoRenew()
}
