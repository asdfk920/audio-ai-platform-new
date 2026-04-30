package logic

import (
	"context"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// WithdrawAccountCancellationLogic 撤销注销申请逻辑处理
type WithdrawAccountCancellationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewWithdrawAccountCancellationLogic 创建撤销注销申请逻辑处理实例
func NewWithdrawAccountCancellationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WithdrawAccountCancellationLogic {
	return &WithdrawAccountCancellationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Withdraw 撤销注销申请
// 仅冷静期内可撤销，冷静期结束后无法撤销
// 参数说明：
//   - clientIP: 客户端 IP 地址（用于日志记录）
//
// 返回说明：
//   - resp: 撤销响应（包含撤销状态、时间等信息）
//   - err: 错误信息
func (l *WithdrawAccountCancellationLogic) Withdraw(clientIP string) (resp *types.WithdrawAccountCancellationResp, err error) {
	// 1. 获取当前用户 ID（从 JWT Token 中解析）
	userID := ctxuser.ParseUserID(l.ctx)
	if userID == 0 {
		// 未登录或 Token 无效
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "未登录")
	}

	// 2. 撤销注销申请
	// 调用数据仓库层方法，在事务中完成撤销操作
	// 包括：更新注销日志状态为已撤销、清除用户冷静期结束时间
	err = l.svcCtx.UserRepo.WithdrawAccountCancellationTx(l.ctx, userID)
	if err != nil {
		// 撤销失败，记录错误日志
		logx.Errorf("withdraw cancellation failed: %v", err)
		return nil, err
	}

	// 3. 返回成功响应
	// 包含撤销状态、时间等信息，供前端展示
	return &types.WithdrawAccountCancellationResp{
		Message:           "已成功撤销注销申请",
		UserId:            userID,
		Status:            "withdrawn",       // 已撤销状态标识
		WithdrawnAt:       time.Now().Unix(), // 撤销时间戳（秒级）
		CoolingEndCleared: true,              // 冷静期标记已清除
	}, nil
}
