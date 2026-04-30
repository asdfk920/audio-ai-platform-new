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

type InviteFamilyMemberLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewInviteFamilyMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InviteFamilyMemberLogic {
	return &InviteFamilyMemberLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *InviteFamilyMemberLogic) InviteFamilyMember(req *types.FamilyMemberInviteReq) (*types.FamilyMemberInviteResp, error) {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "")
	}
	view, err := familysvc.New(l.svcCtx).InviteFamilyMember(l.ctx, familysvc.InviteFamilyMemberInput{
		OperatorUserID: userID,
		TargetUserID:   req.TargetUserId,
		TargetAccount:  req.TargetAccount,
		Role:           req.Role,
		Remark:         req.Remark,
		ExpiresAt:      unixPtr(req.ExpireAt),
	})
	if err != nil {
		return nil, err
	}
	return toFamilyInviteResp(view), nil
}
