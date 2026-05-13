// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// RefreshTokenHandler 刷新 Token 处理器
// @Summary      刷新 Token
// @Description  使用刷新令牌获取新的访问令牌，延长登录有效期
// @Tags         用户认证
// @Accept       json
// @Produce      json
// @Param        body  body      errorx.Response  true  "刷新 Token 请求"
// @Success      200  {object}  errorx.Response  "刷新成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "Token 无效"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/token/refresh [post]
// POST /api/v1/user/token/refresh
// 用途：使用 refresh token 刷新 access token，延长用户登录有效期
func RefreshTokenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RefreshTokenReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewRefreshTokenLogic(r.Context(), svcCtx)
		resp, err := l.RefreshToken(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
