package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// RevokeMemberUnsubscribeLogic 撤销到期退订标记
type RevokeMemberUnsubscribeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRevokeMemberUnsubscribeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeMemberUnsubscribeLogic {
	return &RevokeMemberUnsubscribeLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *RevokeMemberUnsubscribeLogic) RevokeMemberUnsubscribe(_ *types.MemberUnsubscribeRevokeReq) (*types.MemberUnsubscribeResp, error) {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID == 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "未登录")
	}
	if err := l.svcCtx.MemberOrder.RevokeMemberUnsubscribe(l.ctx, userID); err != nil {
		return nil, err
	}
	return &types.MemberUnsubscribeResp{Success: true}, nil
}
