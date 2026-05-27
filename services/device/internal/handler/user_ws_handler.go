package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

// UserWsHandler App（用户 JWT）下载等实时通道
func UserWsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewUserWsLogic(r.Context(), svcCtx)
		l.UserAppWs(w, r)
	}
}
