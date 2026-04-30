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

type ListFamilyMembersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListFamilyMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFamilyMembersLogic {
	return &ListFamilyMembersLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ListFamilyMembersLogic) ListFamilyMembers() (*types.FamilyMemberListResp, error) {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "")
	}
	list, family, err := familysvc.New(l.svcCtx).ListFamilyMembers(l.ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := &types.FamilyMemberListResp{
		Family: *toFamilyInfoResp(family),
		List:   make([]types.FamilyMemberItem, 0, len(list)),
	}
	for _, item := range list {
		resp.List = append(resp.List, toFamilyMemberItem(item))
	}
	return resp, nil
}
