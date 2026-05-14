package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/userinfo"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetUserInfo 查询当前登录用户的详细信息
func (l *GetUserInfoLogic) GetUserInfo() (resp *types.UserInfoResp, err error) {
	userId := ctxuser.ParseUserID(l.ctx)
	if userId <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}

	user, err := l.svcCtx.UserRepo.FindByID(l.ctx, userId)
	if err != nil {
		l.Logger.Errorf("find user by id: %v", err)
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if user == nil {
		return nil, errorx.NewCodeError(errorx.CodeUserNotFound, "用户不存在")
	}

	userInfo := userinfo.FromDAO(user)

	return &types.UserInfoResp{
		UserInfo: userInfo,
	}, nil
}
