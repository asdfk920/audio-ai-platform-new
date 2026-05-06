package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// deviceLocationQueryHandler 设备位置查询处理器
// GET /api/device/location?sn=xxx
// 用途：用户查询指定设备的最新 UWB 定位数据（需 JWT 鉴权）
func deviceLocationQueryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 GET",
				"data": nil,
			})
			return
		}

		sn := r.URL.Query().Get("sn")
		if sn == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "sn 参数不能为空",
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceLocationQueryLogic(r.Context(), svcCtx)
		resp, err := l.DeviceLocationQuery(sn)
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

// DeviceLocationQueryHandler 设备位置查询
// @Summary      设备位置查询
// @Description  查询指定设备的最新 UWB 定位数据
// @Tags         设备位置
// @Accept       json
// @Produce      json
// @Success      200  {object}  types.DeviceLocationResp  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /api/v1/device/location [get]
// @Security     BearerAuth
func DeviceLocationQueryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return deviceLocationQueryHandler(svcCtx)
}

