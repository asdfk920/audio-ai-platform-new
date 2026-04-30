package logic

// 以下 Logic 为目录重构过程中丢失实现时的占位，便于编译与联调；请从 Cursor 本地历史或备份中还原完整业务代码后删除本文件。

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

func errLogicRecover() error {
	return errorx.NewCodeError(errorx.CodeSystemError, "该接口业务逻辑文件缺失，请从编辑器本地历史还原 internal/logic 下对应实现")
}

type AdminMemberListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminMemberListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminMemberListLogic {
	return &AdminMemberListLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *AdminMemberListLogic) AdminMemberList(*types.AdminMemberListReq) (*types.AdminMemberListResp, error) {
	return nil, errLogicRecover()
}

type OauthUnbindLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewOauthUnbindLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OauthUnbindLogic {
	return &OauthUnbindLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *OauthUnbindLogic) Unbind(string) error {
	return errLogicRecover()
}
