package handler

import (
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// LogoutHandler 用户登出处理器
// @Summary      用户登出
// @Description  用户登出，将 token 加入黑名单
// @Tags         用户认证
// @Accept       json
// @Produce      json
// @Success      200  {object}  errorx.Response  "登出成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/logout [post]
// @Security     BearerAuth
// POST /api/v1/user/logout
// 用途：用户登出，将 refresh token 和 access token jti 加入黑名单
func LogoutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewLogoutLogic(r.Context(), svcCtx)
		resp, err := l.Logout(r.Header.Get("Authorization"))
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.Success(resp))
		}
	}
}
