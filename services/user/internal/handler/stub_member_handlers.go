// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// CreateMemberOrderHandler 创建会员订单处理器
// 处理用户创建会员订单的 HTTP 请求
func CreateMemberOrderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 创建逻辑处理实例
		l := logic.NewCreateMemberOrderLogic(r.Context(), svcCtx)

		// 解析请求参数
		var req types.CreateMemberOrderReq
		if err := httpx.Parse(r, &req); err != nil {
			// 参数解析失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 执行创建订单逻辑
		resp, err := l.CreateMemberOrder(&req)
		if err != nil {
			// 创建失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 返回成功响应
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

// PayMemberOrderHandler 支付会员订单处理器
// 处理用户支付会员订单的 HTTP 请求
func PayMemberOrderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 创建逻辑处理实例
		l := logic.NewPayMemberOrderLogic(r.Context(), svcCtx)

		// 解析请求参数
		var req types.PayMemberOrderReq
		if err := httpx.Parse(r, &req); err != nil {
			// 参数解析失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 执行支付逻辑
		resp, err := l.PayMemberOrder(&req)
		if err != nil {
			// 支付失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 返回成功响应
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

// MockPayMemberOrderHandler 模拟支付会员订单处理器
// 用于开发/测试环境，直接标记订单为支付成功并开通会员
func MockPayMemberOrderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 创建逻辑处理实例
		l := logic.NewMockPayMemberOrderLogic(r.Context(), svcCtx)

		// 解析请求参数
		var req types.MockPayMemberOrderReq
		if err := httpx.Parse(r, &req); err != nil {
			// 参数解析失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 执行模拟支付逻辑
		resp, err := l.MockPayMemberOrder(&req)
		if err != nil {
			// 支付失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 返回成功响应
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

// MemberPayCallbackHandler 会员支付回调处理器
// 处理第三方支付平台（微信/支付宝/模拟支付）的异步回调通知
func MemberPayCallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 创建逻辑处理实例
		l := logic.NewMemberPayCallbackLogic(r.Context(), svcCtx)

		// 解析请求参数
		var req types.MemberPayCallbackReq
		if err := httpx.Parse(r, &req); err != nil {
			// 参数解析失败，返回失败给支付平台
			logx.Errorf("parse callback request failed: %v", err)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 执行支付回调逻辑
		err := l.MemberPayCallback(&req)
		if err != nil {
			// 回调处理失败，返回失败给支付平台（支付平台会重试）
			logx.Errorf("handle callback failed: order_no=%s, err=%v", req.OrderNo, err)
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 返回成功（告诉支付平台已收到回调）
		httpx.OkJsonCtx(r.Context(), w, map[string]string{
			"status": "success",
		})
	}
}
