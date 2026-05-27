package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
)

// GetQRCodeKeyHandler 获取二维码 Key
// POST /openapi/music/basic/user/oauth2/qrcodekey/get/v2
// 用途：获取网易云音乐登录二维码的 Key 和图片地址
// 请求参数（Body）: {"type": 2, "expiredKey": 300}
func GetQRCodeKeyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewGetQRCodeKeyLogic(r.Context(), svcCtx)
		resp, err := l.GetQRCodeKey()
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  err.Error(),
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, resp)
	}
}
