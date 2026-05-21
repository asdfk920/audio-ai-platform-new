package handler

import (
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

func deviceAuthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.DeviceAuthReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "参数错误: " + err.Error(),
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceAuthLogic(r.Context(), svcCtx)
		resp, err := l.DeviceAuth(&req)
		if err != nil {
			errMsg := err.Error()

			statusCode := http.StatusUnauthorized

			if strings.Contains(errMsg, "参数错误") || strings.Contains(errMsg, "校验失败") {
				statusCode = http.StatusBadRequest
			} else if strings.Contains(errMsg, "设备不存在") {
				statusCode = http.StatusNotFound
			}

			httpx.WriteJson(w, statusCode, map[string]interface{}{
				"code": statusCode,
				"msg":  errMsg,
				"data": nil,
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"msg":  resp.Message,
			"data": resp,
		})
	}
}
