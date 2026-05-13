package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// deviceDetailHandler 设备详情查询处理器
// POST /api/device/detail
// 用途：用户查看指定设备的详细信息
func deviceDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.DeviceDetailReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "请求参数格式错误",
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceDetailLogic(r.Context(), svcCtx)
		resp, err := l.DeviceDetail(&req)
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
			"msg":  "查询成功",
			"data": resp,
		})
	})
}

// DeviceDetailHandler 设备详情
// @Summary      设备详情
// @Description  查询指定设备的详细信息
// @Tags         设备管理
// @Accept       json
// @Produce      json
// @Param        body  body      types.DeviceDetailReq  true  "设备详情请求"
// @Success      200  {object}  types.DeviceDetailResp  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /api/v1/device/detail [post]
// @Security     BearerAuth
func DeviceDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return deviceDetailHandler(svcCtx)
}
