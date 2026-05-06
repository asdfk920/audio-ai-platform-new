package handler

import (
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// OAuthWechatStartHandler 微信 OAuth 授权开始处理器
// @Summary      微信 OAuth
// @Description  跳转到微信 OAuth 授权页面
// @Tags         OAuth
// @Accept       json
// @Produce      json
// @Success      302  "跳转到微信授权页面"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/oauth/wechat [get]
// GET /api/v1/user/oauth/wechat/start
// 用途：302 跳转微信授权页，引导用户进行微信授权登录
func OAuthWechatStartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewOauthWechatStartLogic(r.Context(), svcCtx)
		u, err := l.AuthorizeURL()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		http.Redirect(w, r, u, http.StatusFound)
	}
}
