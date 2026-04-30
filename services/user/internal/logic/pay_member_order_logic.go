package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// PayMemberOrderLogic 支付会员订单逻辑处理
type PayMemberOrderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewPayMemberOrderLogic 创建支付会员订单逻辑处理实例
func NewPayMemberOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PayMemberOrderLogic {
	return &PayMemberOrderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// PayMemberOrder 支付会员订单
// 参数说明：
//   - req: 支付请求（包含订单号）
//
// 返回说明：
//   - resp: 支付响应（包含订单号、支付方式、支付参数等）
//   - err: 错误信息
func (l *PayMemberOrderLogic) PayMemberOrder(req *types.PayMemberOrderReq) (resp *types.PayMemberOrderResp, err error) {
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
	// 只允许支付待支付状态的订单
	if order.PayStatus != 0 {
		// 订单已支付或其他状态
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "订单已支付或状态异常")
	}

	// 5. Mock 环境：开发阶段不接真实支付，调用 pay 即视为支付成功并自动履约（开通会员）。
	// - 兼容旧客户端：不传 pay_params 时也自动履约
	// - 新客户端：可显式传 pay_params.mock=true
	if req.PayParams == nil || req.PayParams.Mock {
		raw := ""
		if b, merr := json.Marshal(req); merr == nil {
			raw = string(b)
		}

		tx, terr := l.svcCtx.MemberOrder.BeginTx(l.ctx)
		if terr != nil {
			logx.Errorf("begin transaction failed: %v", terr)
			return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
		}
		defer func() {
			if p := recover(); p != nil {
				_ = tx.Rollback()
				panic(p)
			}
		}()

		tradeNo := fmt.Sprintf("MOCK-%s", order.OrderNo)
		if ferr := l.svcCtx.MemberOrder.FulfillPaidOrderTx(l.ctx, tx, order.OrderNo, tradeNo, order.PayType, &raw); ferr != nil {
			_ = tx.Rollback()
			logx.Errorf("mock fulfill paid order failed: %v", ferr)
			return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "支付失败")
		}
		if cerr := tx.Commit(); cerr != nil {
			_ = tx.Rollback()
			logx.Errorf("commit transaction failed: %v", cerr)
			return nil, errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
		}

		return &types.PayMemberOrderResp{
			OrderNo: order.OrderNo,
			PayType: int64(order.PayType),
			PayParams: map[string]any{
				"mock":      true,
				"paid":      true,
				"order_no":  order.OrderNo,
				"trade_no":  tradeNo,
				"pay_type":  order.PayType,
				"timestamp": time.Now().Unix(),
			},
		}, nil
	}

	// 6. 根据支付方式处理（非 mock）
	// 余额支付：直接扣款并履约（当前仍为占位实现）
	// 微信/支付宝：返回 mock 支付参数，等待回调
	if order.PayType == 3 {
		// 余额支付（模拟）
		err = l.handleBalancePay(order.OrderNo, userID)
		if err != nil {
			return nil, err
		}
	}

	// 7. 返回支付响应
	// 包含订单号、支付方式、支付参数（mock）
	return &types.PayMemberOrderResp{
		OrderNo:   order.OrderNo,
		PayType:   int64(order.PayType),
		PayParams: l.getMockPayParams(order),
	}, nil
}

// handleBalancePay 处理余额支付（模拟）
func (l *PayMemberOrderLogic) handleBalancePay(orderNo string, userID int64) error {
	// TODO: 实现余额支付逻辑
	// 1. 检查用户余额是否充足
	// 2. 扣减用户余额
	// 3. 更新订单状态为已支付
	// 4. 开通会员权益
	// 5. 记录支付日志
	// 目前暂时返回 nil，后续根据业务需求补充
	return nil
}

// getMockPayParams 获取 mock 支付参数
// 用于开发阶段模拟微信/支付宝支付
func (l *PayMemberOrderLogic) getMockPayParams(order *dao.OrderMasterRow) map[string]any {
	// TODO: 根据实际支付渠道返回真实参数
	// 微信支付：返回 prepay_id、timeStamp、nonceStr、package、signType、paySign
	// 支付宝：返回 out_trade_no、total_amount、app_id、method、sign 等
	// 目前返回 mock 数据用于联调
	return map[string]any{
		"mock":      true,
		"order_no":  order.OrderNo,
		"trade_no":  "MOCK-" + order.OrderNo,
		"pay_type":  order.PayType,
		"timestamp": time.Now().Unix(),
	}
}
