package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// deviceStatusUpdateHandler 设备状态更新处理器
// POST /api/device/status/update
// 用途：设备通过 HTTP 接口主动上报在线状态
func deviceStatusUpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.DeviceStatusUpdateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "参数错误: " + err.Error(),
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceStatusUpdateLogic(r.Context(), svcCtx)
		resp, err := l.DeviceStatusUpdate(&req)
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
			"msg":  "设备状态更新成功",
			"data": resp,
		})
	}
}

// DeviceStatusUpdateHandler 设备状态更新
// @Summary      设备状态更新
// @Description  设备上报自身状态变化
// @Tags         设备状态
// @Accept       json
// @Produce      json
// @Param        body  body      types.DeviceStatusUpdateReq  true  "设备状态更新请求"
// @Success      200  {object}  types.DeviceStatusUpdateResp  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /api/v1/device/status/update [post]
// @Security     BearerAuth
func DeviceStatusUpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return deviceStatusUpdateHandler(svcCtx)
}

