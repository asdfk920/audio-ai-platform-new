package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// AcceptDeviceShareHandler 接受设备共享处理器
// @Summary      接受设备共享
// @Description  接受其他用户共享的设备
// @Tags         设备分享
// @Accept       json
// @Produce      json
// @Param        body  body      types.DeviceShareAcceptReq  true  "接受设备共享请求"
// @Success      200  {object}  types.DeviceShareItem  "接受成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/device/share/accept [post]
// @Security     BearerAuth
// POST /api/v1/user/device/share/accept
// 用途：用户接受设备共享邀请，获得设备使用权
func AcceptDeviceShareHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeviceShareAcceptReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := logic.NewAcceptDeviceShareLogic(r.Context(), svcCtx).AcceptDeviceShare(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.SuccessData("接受设备共享成功", resp))
	}
}
