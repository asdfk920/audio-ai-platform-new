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

// RealnameRebindHandler 实名认证换绑处理器
// POST /api/v1/user/realname/rebind
// 用途：用户实名认证信息换绑（更换身份证信息）
func RealnameRebindHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RealnameRebindReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewRealnameRebindLogic(r.Context(), svcCtx)
		_, err := l.RealnameRebind(&req)
		httpx.ErrorCtx(r.Context(), w, err)
	}
}
