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

type GetCurrentFamilyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCurrentFamilyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCurrentFamilyLogic {
	return &GetCurrentFamilyLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetCurrentFamilyLogic) GetCurrentFamily() (*types.FamilyInfoResp, error) {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "")
	}
	view, err := familysvc.New(l.svcCtx).GetCurrentFamily(l.ctx, userID)
	if err != nil {
		return nil, err
	}
	return toFamilyInfoResp(view), nil
}
