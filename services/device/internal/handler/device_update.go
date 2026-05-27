package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// deviceUpdateHandler 设备信息更新处理器
// PUT /api/v1/user/device/upd
// 用途：用户通过 App 更新已绑定设备的基本信息和配置（别名、位置、分组、场景等）
func deviceUpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 PUT",
				"data": nil,
			})
			return
		}

		var req types.DeviceUpdateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "参数错误: " + err.Error(),
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceUpdateLogic(r.Context(), svcCtx)
		resp, err := l.DeviceUpdate(&req)
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
			"msg":  "修改成功",
			"data": resp,
		})
	})
}

// DeviceUpdateHandler 设备信息更新
// @Summary      设备信息更新
// @Description  用户更新已绑定设备的基本信息和配置（别名、位置、分组、场景等）
// @Tags         用户设备管理
// @Accept       json
// @Produce      json
// @Param        body  body      types.DeviceUpdateReq  true  "设备信息更新请求"
// @Success      200  {object}  types.DeviceUpdateResp  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /api/v1/user/device/upd [put]
// @Security     BearerAuth
func DeviceUpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return deviceUpdateHandler(svcCtx)
}
