package handler

import (
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// LogoutHandler 用户登出处理器
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
