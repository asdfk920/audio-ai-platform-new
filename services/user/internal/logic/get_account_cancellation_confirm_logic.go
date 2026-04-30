package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// GetAccountCancellationConfirmLogic 获取注销确认页文案逻辑处理
type GetAccountCancellationConfirmLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetAccountCancellationConfirmLogic 创建获取注销确认页文案逻辑处理实例
func NewGetAccountCancellationConfirmLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAccountCancellationConfirmLogic {
	return &GetAccountCancellationConfirmLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Get 获取注销二次确认页说明文案
// 用于前端展示注销后果，确保用户充分了解注销的影响
// 返回说明：
//   - resp: 确认页响应（包含标题、后果说明列表）
//   - err: 错误信息
func (l *GetAccountCancellationConfirmLogic) Get() (resp *types.GetAccountCancellationConfirmResp, err error) {
	// 1. 定义注销后果说明列表
	// 这些文案用于告知用户注销账号的严重后果，避免用户误操作
	consequences := []string{
		"账号注销后，所有个人信息将被永久删除",
		"账号下的所有权益（会员、优惠券、余额等）将全部清空",
		"账号注销后无法恢复，请谨慎操作",
		"冷静期 7 天内可撤销注销申请",
	}

	// 2. 返回确认页响应
	// 包含标题和后果说明列表，供前端展示
	return &types.GetAccountCancellationConfirmResp{
		Title:        "账号注销确认",     // 确认页标题
		Consequences: consequences, // 后果说明列表
	}, nil
}
