package logic

import (
	"context"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// MockPayMemberOrderLogic 模拟支付会员订单逻辑处理
// 用于开发/测试环境，直接标记订单为支付成功并开通会员
type MockPayMemberOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewMockPayMemberOrderLogic 创建模拟支付会员订单逻辑处理实例
func NewMockPayMemberOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MockPayMemberOrderLogic {
	return &MockPayMemberOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// MockPayMemberOrder 模拟支付会员订单
// 参数说明：
//   - req: 模拟支付请求（包含订单号）
//
// 返回说明：
//   - resp: 支付响应（包含订单号、支付结果、会员信息等）
//   - err: 错误信息
func (l *MockPayMemberOrderLogic) MockPayMemberOrder(req *types.MockPayMemberOrderReq) (resp *types.MockPayMemberOrderResp, err error) {
	// 1. 获取当前用户 ID（从 JWT Token 中解析）
	userID := ctxuser.ParseUserID(l.ctx)
	if userID == 0 {
		// 未登录或 Token 无效
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "未登录")
	}

	// 2. 参数校验
	if req.OrderNo == "" {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "订单号不能为空")
	}

	// 3. 查询订单信息
	// 验证订单是否存在、是否属于当前用户、状态是否正确
	order, err := l.svcCtx.MemberOrder.GetOrderByNoForUser(l.ctx, req.OrderNo, userID)
	if err != nil {
		logx.Errorf("get order failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "查询订单失败")
	}
	if order == nil {
		// 订单不存在或不属于当前用户
		return nil, errorx.NewCodeError(errorx.CodeMemberOrderNotFound, "订单不存在")
	}

	// 4. 校验订单状态
	// 只允许支付待支付状态的订单（pay_status = 0）
	if order.PayStatus != 0 {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "订单状态异常，仅支持待支付订单")
	}

	// 5. 开启数据库事务
	tx, err := l.svcCtx.MemberOrder.BeginTx(l.ctx)
	if err != nil {
		logx.Errorf("begin transaction failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}
	defer func() {
		// 如果有 panic，回滚事务
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	// 6. 执行模拟支付
	// 直接标记订单为已支付，并开通会员
	err = l.svcCtx.MemberOrder.FulfillPaidOrderTx(l.ctx, tx, order.OrderNo, "MOCK-PAY", 3, nil)
	if err != nil {
		_ = tx.Rollback()
		logx.Errorf("fulfill paid order failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "支付失败")
	}

	// 7. 提交事务
	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		logx.Errorf("commit transaction failed: %v", err)
		return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}

	// 8. 查询会员信息（用于返回）
	memberInfo, err := l.svcCtx.MemberOrder.GetUserMemberInfo(l.ctx, userID)
	if err != nil {
		logx.Errorf("get user member info failed: %v", err)
		// 不影响支付结果，继续返回
	}

	// 9. 记录日志
	logx.Infof("mock pay success: user_id=%d, order_no=%s, amount_cent=%d",
		userID, order.OrderNo, order.AmountCent)

	// 10. 返回支付成功响应
	resp = &types.MockPayMemberOrderResp{
		OrderNo:    order.OrderNo,
		Success:    true,
		Message:    "支付成功",
		PayType:    3, // 模拟支付
		AmountCent: order.AmountCent,
		MemberInfo: l.convertMemberInfo(memberInfo),
	}

	return resp, nil
}

// convertMemberInfo 转换会员信息为响应格式
func (l *MockPayMemberOrderLogic) convertMemberInfo(info *dao.UserMemberRow) *types.MemberInfo {
	if info == nil {
		return nil
	}

	return &types.MemberInfo{
		LevelCode:   info.LevelCode,
		LevelName:   l.getLevelName(info.LevelCode),
		ExpireAt:    info.ExpireAt.Unix(),
		IsPermanent: info.IsPermanent == 1,
		Status:      int64(info.Status),
	}
}

// getLevelName 根据等级编码返回等级名称
func (l *MockPayMemberOrderLogic) getLevelName(levelCode string) string {
	// TODO: 从配置或数据库读取
	switch levelCode {
	case "vip_monthly":
		return "月度会员"
	case "vip_quarterly":
		return "季度会员"
	case "vip_yearly":
		return "年度会员"
	default:
		return "会员"
	}
}
