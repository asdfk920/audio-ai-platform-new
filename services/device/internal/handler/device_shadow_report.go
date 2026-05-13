package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// deviceShadowReportHandler 设备影子定时上报处理器
// POST /api/device/shadow/report
// 用途：设备定时采集状态数据并通过 HTTP 上报到云端
func deviceShadowReportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.DeviceShadowReportReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "请求参数格式错误",
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceShadowReportLogic(r.Context(), svcCtx)
		resp, err := l.DeviceShadowReport(&req)
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
			"msg":  "上报成功",
			"data": resp,
		})
	}
}

// DeviceShadowReportHandler 设备影子上报
// @Summary      设备影子上报
// @Description  设备定时采集状态数据上报云端
// @Tags         设备影子
// @Accept       json
// @Produce      json
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /api/v1/device/shadow/report [post]
// @Security     BearerAuth
func DeviceShadowReportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return deviceShadowReportHandler(svcCtx)
}
