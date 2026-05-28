package handler

import (
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

// DeviceShadowV2QueryHandler 查询设备影子（v2 路径，兼容 device_sn 参数）
// GET /api/v2/device/shadow?device_sn=xxx
func DeviceShadowV2QueryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 GET",
				"data": nil,
			})
			return
		}

		deviceSN := strings.TrimSpace(r.URL.Query().Get("device_sn"))
		if deviceSN == "" {
			deviceSN = strings.TrimSpace(r.URL.Query().Get("sn"))
		}
		if deviceSN == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "device_sn 参数不能为空",
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceShadowV2QueryLogic(r.Context(), svcCtx)
		resp, err := l.Query(deviceSN)
		if err != nil {
			status := http.StatusBadRequest
			if errorx.IsCode(err, errorx.CodeNotFound) {
				status = http.StatusNotFound
			}
			httpx.WriteJson(w, status, map[string]interface{}{
				"code": errorx.CodeOf(err),
				"msg":  err.Error(),
				"data": nil,
			})
			return
		}

		msg := "查询成功"
		if resp.Message != "" {
			msg = resp.Message
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"msg":  msg,
			"data": resp,
		})
	}
}
