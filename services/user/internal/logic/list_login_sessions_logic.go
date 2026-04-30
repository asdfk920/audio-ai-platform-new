package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/auth/loginsession"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListLoginSessionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListLoginSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLoginSessionsLogic {
	return &ListLoginSessionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLoginSessionsLogic) ListLoginSessions() (resp *types.ListLoginSessionsResp, err error) {
	uid := ctxuser.ParseUserID(l.ctx)
	if uid <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}
	views, err := loginsession.List(l.ctx, uid, 50)
	if err != nil {
		l.Logger.Errorf("list sessions: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeRedisError, err.Error())
	}
	list := make([]types.LoginSessionItem, 0, len(views))
	for _, v := range views {
		list = append(list, types.LoginSessionItem{
			SessionId:  v.SessionID,
			DeviceId:   v.DeviceID,
			DeviceName: v.DeviceName,
			Platform:   v.Platform,
			Ip:         v.IP,
			UserAgent:  v.UserAgent,
			CreatedAt:  v.CreatedAt,
			LastSeenAt: v.LastSeenAt,
			Current:    v.Current,
		})
	}
	return &types.ListLoginSessionsResp{List: list}, nil
}
