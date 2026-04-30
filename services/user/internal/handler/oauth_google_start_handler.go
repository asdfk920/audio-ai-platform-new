package handler

import (
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// OAuthGoogleStartHandler Google OAuth 授权开始处理器
// GET /api/v1/user/oauth/google/start
// 用途：302 跳转 Google 授权页，引导用户进行 Google 授权登录
func OAuthGoogleStartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewOauthGoogleStartLogic(r.Context(), svcCtx)
		u, err := l.AuthorizeURL()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		http.Redirect(w, r, u, http.StatusFound)
	}
}
