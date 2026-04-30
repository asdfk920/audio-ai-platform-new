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

type RemoveFamilyMemberLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRemoveFamilyMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveFamilyMemberLogic {
	return &RemoveFamilyMemberLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *RemoveFamilyMemberLogic) RemoveFamilyMember(req *types.FamilyMemberRemoveReq) error {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID <= 0 {
		return errorx.NewCodeError(errorx.CodeTokenInvalid, "")
	}
	return familysvc.New(l.svcCtx).RemoveFamilyMember(l.ctx, userID, req.UserId)
}
