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

// 设置自动续费开关与意向套餐/支付方式（需已有 user_member 行；不真实扣款）
func setMemberAutoRenewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
