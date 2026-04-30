// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func AdminMemberListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{"list": []interface{}{}})
	}
}

func GetUserMemberBenefitsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 创建逻辑处理实例
		l := logic.NewGetUserMemberBenefitsLogic(r.Context(), svcCtx)

		// 解析请求参数
		var req types.GetUserMemberBenefitsReq
		if err := httpx.Parse(r, &req); err != nil {
			// 参数解析失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 执行查询逻辑
		resp, err := l.GetUserMemberBenefits(&req)
		if err != nil {
			// 查询失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 返回成功响应
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

func GetMemberAutoRenewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewGetMemberAutoRenewLogic(r.Context(), svcCtx)
		resp, err := l.GetMemberAutoRenew()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

func UnsubscribeMemberHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.MemberUnsubscribeReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := logic.NewUnsubscribeMemberLogic(r.Context(), svcCtx)
		resp, err := l.UnsubscribeMember(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

func RevokeMemberUnsubscribeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.MemberUnsubscribeRevokeReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := logic.NewRevokeMemberUnsubscribeLogic(r.Context(), svcCtx)
		resp, err := l.RevokeMemberUnsubscribe(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

func SetMemberAutoRenewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SetMemberAutoRenewReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := logic.NewSetMemberAutoRenewLogic(r.Context(), svcCtx)
		resp, err := l.SetMemberAutoRenew(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

func GetUserMemberInfoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 创建逻辑处理实例
		l := logic.NewGetUserMemberInfoLogic(r.Context(), svcCtx)

		// 解析请求参数
		var req types.GetUserMemberInfoReq
		if err := httpx.Parse(r, &req); err != nil {
			// 参数解析失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 执行查询逻辑
		resp, err := l.GetUserMemberInfo(&req)
		if err != nil {
			// 查询失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 返回成功响应
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

func ApplyAccountCancellationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 创建逻辑处理实例
		l := logic.NewApplyAccountCancellationLogic(r.Context(), svcCtx)

		// 解析请求参数
		var req types.ApplyAccountCancellationReq
		if err := httpx.Parse(r, &req); err != nil {
			// 参数解析失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 获取客户端信息（用于日志记录）
		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		// 执行注销申请逻辑
		resp, err := l.Apply(&req, clientIP, userAgent)
		if err != nil {
			// 处理失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 返回成功响应
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

func GetAccountCancellationStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 创建逻辑处理实例
		l := logic.NewGetAccountCancellationStatusLogic(r.Context(), svcCtx)

		// 执行查询状态逻辑
		resp, err := l.Get()
		if err != nil {
			// 查询失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 返回成功响应
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

func GetAccountCancellationConfirmHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 创建逻辑处理实例
		l := logic.NewGetAccountCancellationConfirmLogic(r.Context(), svcCtx)

		// 执行获取确认页文案逻辑
		resp, err := l.Get()
		if err != nil {
			// 获取失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 返回成功响应
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}

func OAuthUnbindHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.OkJsonCtx(r.Context(), w, map[string]string{"status": "ok"})
	}
}
