package logic

import (
	"context"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/auth/loginsession"
	"github.com/zeromicro/go-zero/core/logx"
)

type KickLoginSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewKickLoginSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KickLoginSessionLogic {
	return &KickLoginSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *KickLoginSessionLogic) KickLoginSession(req *types.KickLoginSessionReq) (resp *types.KickLoginSessionResp, err error) {
	uid := ctxuser.ParseUserID(l.ctx)
	if uid <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}
	if req == nil || strings.TrimSpace(req.SessionId) == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "请传入 session_id")
	}
	sid := strings.TrimSpace(req.SessionId)
	if err := loginsession.RevokeByPublicID(l.ctx, uid, sid); err != nil {
		l.Logger.Errorf("revoke session: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeParamError, err.Error())
	}
	return &types.KickLoginSessionResp{Ok: true}, nil
}
