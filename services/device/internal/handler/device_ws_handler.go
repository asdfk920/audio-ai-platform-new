package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

func DeviceWsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewDeviceWsLogic(r.Context(), svcCtx)
		l.DeviceWs(w, r)
	}
}
