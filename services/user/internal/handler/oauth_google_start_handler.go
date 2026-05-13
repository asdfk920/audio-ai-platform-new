package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
)

// OAuthGoogleStartHandler Google OAuth 授权开始处理器
// @Summary      Google OAuth
// @Description  跳转到 Google OAuth 授权页面
// @Tags         OAuth
// @Accept       json
// @Produce      json
// @Success      302  "跳转到 Google 授权页面"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/oauth/google [get]
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
