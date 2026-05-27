package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
)

// SimulateQRCodeLoginHandler 模拟扫码登录成功（测试接口）
// POST /openapi/music/basic/oauth2/qrcode/login/simulate
// 用途：模拟用户扫码登录成功，将账号信息写入数据库并更新状态
func SimulateQRCodeLoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
			})
			return
		}

		var req logic.SimulateQRCodeLoginReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "参数错误: " + err.Error(),
			})
			return
		}

		l := logic.NewSimulateQRCodeLoginLogic(r.Context(), svcCtx)
		resp, err := l.SimulateQRCodeLogin(&req)
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
