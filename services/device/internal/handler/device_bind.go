package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// deviceBindHandler 设备绑定处理器
// POST /api/device/bind
// 用途：用户通过 App 将设备绑定到当前登录账户
func deviceBindHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.DeviceBindReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "参数错误: " + err.Error(),
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceBindLogic(r.Context(), svcCtx)
		resp, err := l.DeviceBind(&req)
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
			"msg":  "设备绑定成功",
			"data": resp,
		})
	})
}

// DeviceBindHandler 设备绑定
// @Summary      设备绑定
// @Description  用户将设备绑定到自己的账户下
// @Tags         设备管理
// @Accept       json
// @Produce      json
// @Param        body  body      types.DeviceBindReq  true  "设备绑定请求"
// @Success      200  {object}  types.DeviceBindResp  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /api/v1/device/bind [post]
// @Security     BearerAuth
func DeviceBindHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return deviceBindHandler(svcCtx)
}
