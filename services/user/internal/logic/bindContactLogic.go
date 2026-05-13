// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type BindContactLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 绑定邮箱/手机号（已登录，邮箱或手机号二选一 + 验证码；验证码校验成功后进行绑定）
func NewBindContactLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindContactLogic {
	return &BindContactLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BindContactLogic) BindContact(req *types.BindContactReq) (resp *types.UserInfo, err error) {
	// todo: add your logic here and delete this line

	return
}
