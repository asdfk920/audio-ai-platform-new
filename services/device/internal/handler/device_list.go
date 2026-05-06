package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// deviceListHandler 设备列表查询处理器
// GET /api/device/list
// 用途：用户查询已绑定设备列表
func deviceListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 GET",
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceListLogic(r.Context(), svcCtx)
		resp, err := l.DeviceList()
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

// DeviceListHandler 设备列表
// @Summary      设备列表
// @Description  查询当前用户已绑定的设备列表
// @Tags         设备管理
// @Accept       json
// @Produce      json
// @Success      200  {object}  types.DeviceListResp  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /api/v1/device/list [get]
// @Security     BearerAuth
func DeviceListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return deviceListHandler(svcCtx)
}

