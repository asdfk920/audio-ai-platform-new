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

// 会员套餐预下单：package_code 为上架套餐编码；pay_type 1微信 2支付宝 3余额
func createMemberOrderHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateMemberOrderReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewCreateMemberOrderLogic(r.Context(), svcCtx)
		resp, err := l.CreateMemberOrder(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
