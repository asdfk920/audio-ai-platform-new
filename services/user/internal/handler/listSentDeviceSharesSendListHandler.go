package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
)

// listSentDeviceSharesSendListHandler 与 listSentDeviceSharesHandler 相同实现（路径 /device/share/send/list）。
func listSentDeviceSharesSendListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return listSentDeviceSharesHandler(svcCtx)
}
