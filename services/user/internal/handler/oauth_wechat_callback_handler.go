package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// OAuthWechatCallbackHandler 微信 OAuth 回调处理器
// @Summary      微信 OAuth 回调
// @Description  微信 OAuth 授权回调，自动创建或绑定用户
// @Tags         OAuth
// @Accept       json
// @Produce      json
// @Param        code  query  string  true  "授权码"
// @Param        state  query  string  false  "状态参数"
// @Success      200  {object}  types.OAuthLoginResp  "认证成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/oauth/wechat/callback [get]
// GET /api/v1/user/oauth/wechat/callback
// 用途：处理微信授权回调，获取用户信息并映射到系统用户
func OAuthWechatCallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OAuthCallbackReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewOauthWechatCallbackLogic(r.Context(), svcCtx)
		resp, err := l.OauthWechatCallback(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
