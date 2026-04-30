package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/familysvc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type AcceptFamilyMemberLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAcceptFamilyMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AcceptFamilyMemberLogic {
	return &AcceptFamilyMemberLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *AcceptFamilyMemberLogic) AcceptFamilyMember(req *types.FamilyMemberAcceptReq) (*types.FamilyInfoResp, error) {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "")
	}
	view, err := familysvc.New(l.svcCtx).AcceptFamilyInvite(l.ctx, userID, req.InviteCode)
	if err != nil {
		return nil, err
	}
	return toFamilyInfoResp(view), nil
}
