package logic

import (
	"context"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

var memberUnsubscribeReasonCodes = map[string]struct{}{
	"price":            {},
	"features":         {},
	"low_usage":        {},
	"switch_platform":  {},
	"service":          {},
	"other":            {},
}

// UnsubscribeMemberLogic 会员退订
type UnsubscribeMemberLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUnsubscribeMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnsubscribeMemberLogic {
	return &UnsubscribeMemberLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *UnsubscribeMemberLogic) UnsubscribeMember(req *types.MemberUnsubscribeReq) (*types.MemberUnsubscribeResp, error) {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID == 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "未登录")
	}
	rc := strings.TrimSpace(req.ReasonCode)
	if _, ok := memberUnsubscribeReasonCodes[rc]; !ok {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "退订原因不合法")
	}
	if err := l.svcCtx.MemberOrder.RequestMemberUnsubscribe(l.ctx, userID, rc, req.Feedback); err != nil {
		return nil, err
	}
	return &types.MemberUnsubscribeResp{Success: true}, nil
}
