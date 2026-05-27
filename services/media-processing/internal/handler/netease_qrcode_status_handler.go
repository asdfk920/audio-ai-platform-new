package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
)

// CheckQRCodeStatusHandler 检查二维码登录状态
// POST /openapi/music/basic/oauth2/device/login/qrcode/get
// 用途：轮询二维码登录状态（前端定时调用）
// 请求参数（Body）: {"key": "uniKey", "clientId": "AppID"}
func CheckQRCodeStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Key      string `json:"key"`
			ClientID string `json:"clientId"`
		}

		// 从 Body 解析参数（POST 请求）
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "参数解析失败: " + err.Error(),
			})
			return
		}

		if req.Key == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "参数错误: key 不能为空",
			})
			return
		}

		l := logic.NewCheckQRCodeStatusLogic(r.Context(), svcCtx)
		resp, err := l.CheckQRCodeStatus(req.Key)
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
