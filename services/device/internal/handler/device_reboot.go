package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// deviceRebootHandler 设备重启指令处理器
// POST /api/device/cmd/reboot
// 用途：用户通过 App 向设备下发重启指令，后端通过 MQTT 将指令推送给设备
func deviceRebootHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.DeviceRebootReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "参数错误: " + err.Error(),
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceRebootLogic(r.Context(), svcCtx)
		resp, err := l.DeviceReboot(&req)
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
			"msg":  "成功",
			"data": resp,
		})
	}
}

// DeviceRebootHandler 设备重启
// @Summary      设备重启
// @Description  用户通过 App 向设备下发重启指令
// @Tags         设备控制
// @Accept       json
// @Produce      json
// @Param        body  body      types.DeviceRebootReq  true  "设备重启请求"
// @Success      200  {object}  types.DeviceRebootResp  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /api/v1/device/cmd/reboot [post]
// @Security     BearerAuth
func DeviceRebootHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return deviceRebootHandler(svcCtx)
}

