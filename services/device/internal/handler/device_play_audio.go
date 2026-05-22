package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// POST /api/device/cmd/play_audio
// POST /api/v1/device/play/audio（同上）
// 点播：创建 play_audio 指令并通过 WebSocket 下发（与同服务 pause/play 链路一致）。
func devicePlayAudioHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.DevicePlayAudioReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "请求体格式错误",
				"data": nil,
			})
			return
		}

		l := logic.NewDevicePlayAudioLogic(r.Context(), svcCtx)
		resp, err := l.DevicePlayAudio(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  err.Error(),
				"data": nil,
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"msg":  "操作成功",
			"data": resp,
		})
	})
}
