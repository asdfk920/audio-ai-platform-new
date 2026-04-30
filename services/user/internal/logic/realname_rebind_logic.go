package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

type RealnameRebindLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewRealnameRebindLogic 实名认证换绑（占位：当前项目未落实名认证信息时暂不支持）
func NewRealnameRebindLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RealnameRebindLogic {
	return &RealnameRebindLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RealnameRebindLogic) RealnameRebind(req *types.RealnameRebindReq) (resp *types.UserInfo, err error) {
	return nil, errorx.NewCodeError(errorx.CodeSystemError, "暂不支持实名认证换绑，请使用旧手机号/邮箱验证码换绑")
}
