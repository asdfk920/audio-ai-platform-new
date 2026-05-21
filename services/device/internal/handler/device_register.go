package handler

import (
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

func deviceRegisterHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.DeviceRegisterReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "参数错误: " + err.Error(),
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceRegisterLogic(r.Context(), svcCtx)
		resp, err := l.DeviceRegister(&req)
		if err != nil {
			errMsg := err.Error()

			statusCode := http.StatusBadRequest

			if strings.Contains(errMsg, "非法设备") {
				statusCode = http.StatusNotFound
			} else if strings.Contains(errMsg, "认证失败") || strings.Contains(errMsg, "密钥不正确") {
				statusCode = http.StatusUnauthorized
			}

			httpx.WriteJson(w, statusCode, map[string]interface{}{
				"code": statusCode,
				"msg":  errMsg,
				"data": nil,
			})
			return
		}

		msg := "注册成功"
		if resp.Status == "already_activated" {
			msg = resp.Message
		} else if resp.Status == "activated" {
			msg = "设备激活成功"
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"msg":  msg,
			"data": resp,
		})
	}
}
