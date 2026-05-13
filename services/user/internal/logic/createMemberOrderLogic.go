// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateMemberOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 会员套餐预下单：package_code 为上架套餐编码；pay_type 1微信 2支付宝 3余额
func NewCreateMemberOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMemberOrderLogic {
	return &CreateMemberOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateMemberOrderLogic) CreateMemberOrder(req *types.CreateMemberOrderReq) (resp *types.CreateMemberOrderResp, err error) {
	// todo: add your logic here and delete this line

	return
}
