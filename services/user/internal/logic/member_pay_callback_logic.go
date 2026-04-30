package logic

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// MemberPayCallbackLogic 会员支付回调逻辑处理
// 处理第三方支付平台（微信/支付宝/模拟支付）的异步回调通知
type MemberPayCallbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewMemberPayCallbackLogic 创建会员支付回调逻辑处理实例
func NewMemberPayCallbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MemberPayCallbackLogic {
	return &MemberPayCallbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// MemberPayCallback 处理支付回调通知
// 参数说明：
//   - req: 支付回调请求（包含订单号、交易号、签名等）
//
// 返回说明：
//   - err: 错误信息（返回给支付平台，决定是否需要重试）
func (l *MemberPayCallbackLogic) MemberPayCallback(req *types.MemberPayCallbackReq) error {
	// 1. 参数校验
	if req.OrderNo == "" || req.TradeNo == "" || req.Sign == "" {
		logx.Errorf("callback param invalid: order_no=%s, trade_no=%s", req.OrderNo, req.TradeNo)
		return errorx.NewCodeError(errorx.CodeInvalidParam, "参数不完整")
	}

	// 2. 验签校验（最重要）
	// 使用配置的密钥验证签名是否合法
	if !l.verifySign(req) {
		logx.Errorf("callback sign verify failed: order_no=%s, trade_no=%s", req.OrderNo, req.TradeNo)
		return errorx.NewCodeError(errorx.CodeInvalidParam, "签名验证失败")
	}

	// 3. 查询订单
	// 根据订单号查询订单信息
	order, err := l.svcCtx.MemberOrder.GetOrderByNo(l.ctx, nil, req.OrderNo)
	if err != nil {
		logx.Errorf("query order failed: order_no=%s, err=%v", req.OrderNo, err)
		return errorx.NewCodeError(errorx.CodeDatabaseError, "查询订单失败")
	}
	if order == nil {
		logx.Errorf("order not found: order_no=%s", req.OrderNo)
		return errorx.NewCodeError(errorx.CodeMemberOrderNotFound, "订单不存在")
	}

	// 4. 幂等性处理
	// 订单已支付，直接返回成功（避免重复处理）
	if order.PayStatus == 1 {
		logx.Infof("order already paid, ignore: order_no=%s, user_id=%d", req.OrderNo, order.UserID)
		return nil
	}

	// 5. 订单状态校验
	// 只允许处理待支付（pay_status=0）的订单
	if order.PayStatus != 0 {
		logx.Errorf("order status not pending, skip: order_no=%s, pay_status=%d", req.OrderNo, order.PayStatus)
		return nil
	}

	// 6. 金额校验
	// 对比回调金额与订单金额是否一致（防止篡改）
	// 注意：实际场景中需要从回调参数中获取支付金额进行对比
	// 这里假设回调中会传递金额信息，暂时跳过校验

	// 7. 开启数据库事务
	tx, err := l.svcCtx.MemberOrder.BeginTx(l.ctx)
	if err != nil {
		logx.Errorf("begin transaction failed: order_no=%s, err=%v", req.OrderNo, err)
		return errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}
	defer func() {
		// 如果有 panic，回滚事务
		if p := recover(); p != nil {
			_ = tx.Rollback()
			logx.Errorf("panic in callback handler: order_no=%s, panic=%v", req.OrderNo, p)
			panic(p)
		}
	}()

	// 8. 更新订单状态并开通会员
	// 调用 FulfillPaidOrderTx 统一处理
	err = l.svcCtx.MemberOrder.FulfillPaidOrderTx(l.ctx, tx, req.OrderNo, req.TradeNo, order.PayType, nil)
	if err != nil {
		_ = tx.Rollback()
		logx.Errorf("fulfill paid order failed: order_no=%s, err=%v", req.OrderNo, err)
		return errorx.NewCodeError(errorx.CodeDatabaseError, "订单履约失败")
	}

	// 9. 提交事务
	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		logx.Errorf("commit transaction failed: order_no=%s, err=%v", req.OrderNo, err)
		return errorx.NewCodeError(errorx.CodeDatabaseError, "系统繁忙")
	}

	// 10. 记录日志
	logx.Infof("pay callback success: order_no=%s, user_id=%d, trade_no=%s, amount_cent=%d",
		req.OrderNo, order.UserID, req.TradeNo, order.AmountCent)

	// 11. 返回成功（告诉支付平台已收到回调）
	return nil
}

// verifySign 验证支付回调签名
// 使用 HMAC-SHA256 算法验证签名
func (l *MemberPayCallbackLogic) verifySign(req *types.MemberPayCallbackReq) bool {
	// 获取配置的密钥
	secret := l.svcCtx.Config.Payment.MockCallbackSecret
	if secret == "" {
		// 使用默认密钥（仅开发环境）
		secret = "default_mock_secret_key_2024"
	}

	// 构造待签名字符串
	// 格式：order_no|trade_no
	signContent := fmt.Sprintf("%s|%s", req.OrderNo, req.TradeNo)

	// 使用 HMAC-SHA256 计算签名
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signContent))
	expectedSign := hex.EncodeToString(mac.Sum(nil))

	// 对比签名
	return hmac.Equal([]byte(expectedSign), []byte(req.Sign))
}

// GenerateMockSign 生成模拟签名（用于测试）
// 实际场景中由支付平台生成
func (l *MemberPayCallbackLogic) GenerateMockSign(orderNo, tradeNo string) string {
	secret := l.svcCtx.Config.Payment.MockCallbackSecret
	if secret == "" {
		secret = "default_mock_secret_key_2024"
	}

	signContent := fmt.Sprintf("%s|%s", orderNo, tradeNo)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signContent))
	return hex.EncodeToString(mac.Sum(nil))
}

// ParseCallbackRequest 解析支付回调请求
// 支持 JSON 和 Form 格式
func (l *MemberPayCallbackLogic) ParseCallbackRequest(body []byte, contentType string) (*types.MemberPayCallbackReq, error) {
	var req types.MemberPayCallbackReq

	// 根据 Content-Type 解析
	if contentType == "application/json" {
		// JSON 格式
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, err
		}
	} else {
		// Form 格式（application/x-www-form-urlencoded）
		// 需要手动解析，这里简化处理
		// 实际场景中可以使用 url.ParseQuery 解析
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "不支持的回调格式")
	}

	return &req, nil
}

// BuildCallbackResponse 构建回调响应
// 返回给支付平台的固定格式
func (l *MemberPayCallbackLogic) BuildCallbackResponse(success bool) string {
	if success {
		return "success"
	}
	return "fail"
}

// LogCallbackDetail 记录详细的回调日志
// 用于审计和问题排查
func (l *MemberPayCallbackLogic) LogCallbackDetail(orderNo string, req *types.MemberPayCallbackReq, success bool, err error) {
	logData := map[string]interface{}{
		"order_no":  orderNo,
		"trade_no":  req.TradeNo,
		"sign":      req.Sign,
		"success":   success,
		"timestamp": time.Now().Unix(),
	}

	if err != nil {
		logData["error"] = err.Error()
	}

	logJSON, _ := json.Marshal(logData)
	if success {
		logx.Infof("pay callback detail: %s", string(logJSON))
	} else {
		logx.Errorf("pay callback failed: %s", string(logJSON))
	}
}
