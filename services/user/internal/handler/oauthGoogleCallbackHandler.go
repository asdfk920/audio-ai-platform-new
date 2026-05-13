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

// Google OAuth 直接登录（授权后回调，自动创建/绑定用户并返回 token，无需单独注册）
func oauthGoogleCallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OAuthCallbackReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewOauthGoogleCallbackLogic(r.Context(), svcCtx)
		resp, err := l.OauthGoogleCallback(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
