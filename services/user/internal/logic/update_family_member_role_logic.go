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

type UpdateFamilyMemberRoleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateFamilyMemberRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateFamilyMemberRoleLogic {
	return &UpdateFamilyMemberRoleLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *UpdateFamilyMemberRoleLogic) UpdateFamilyMemberRole(req *types.FamilyMemberRoleUpdateReq) error {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID <= 0 {
		return errorx.NewCodeError(errorx.CodeTokenInvalid, "")
	}
	return familysvc.New(l.svcCtx).ChangeMemberRole(l.ctx, userID, req.UserId, req.Role)
}
