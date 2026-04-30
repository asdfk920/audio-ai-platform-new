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

type CreateFamilyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateFamilyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateFamilyLogic {
	return &CreateFamilyLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *CreateFamilyLogic) CreateFamily(req *types.FamilyCreateReq) (*types.FamilyInfoResp, error) {
	userID := ctxuser.ParseUserID(l.ctx)
	if userID <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "")
	}
	view, err := familysvc.New(l.svcCtx).CreateFamily(l.ctx, familysvc.CreateFamilyInput{
		OwnerUserID: userID,
		Name:        req.Name,
	})
	if err != nil {
		return nil, err
	}
	return toFamilyInfoResp(view), nil
}
