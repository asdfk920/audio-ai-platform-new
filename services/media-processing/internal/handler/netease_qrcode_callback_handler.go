package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
)

// QRCodeCallbackHandler 二维码扫码回调接口（接收第三方登录结果）
// POST /openapi/music/basic/oauth2/qrcode/callback
// 用途：第三方平台回调，通知用户已扫码并授权成功
func QRCodeCallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
			})
			return
		}

		var req logic.QRCodeCallbackReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "参数错误: " + err.Error(),
			})
			return
		}

		l := logic.NewQRCodeCallbackLogic(r.Context(), svcCtx)
		resp, err := l.QRCodeCallback(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
