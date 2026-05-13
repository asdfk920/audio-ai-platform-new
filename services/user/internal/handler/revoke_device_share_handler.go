package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// RevokeDeviceShareHandler 撤销设备共享处理器
// @Summary      撤销设备共享
// @Description  撤销已发出的设备共享邀请，收回设备使用权
// @Tags         设备分享
// @Accept       json
// @Produce      json
// @Param        body  body      types.DeviceShareRevokeReq  true  "撤销设备共享请求"
// @Success      200  {object}  errorx.Response  "撤销成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/device/share/revoke [post]
// @Security     BearerAuth
//
// POST /api/v1/user/device/share/revoke
// 用途：设备所有者撤销设备共享，收回设备使用权
func RevokeDeviceShareHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeviceShareRevokeReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		err := logic.NewRevokeDeviceShareLogic(r.Context(), svcCtx).RevokeDeviceShare(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.SuccessData("撤销设备共享成功", nil))
	}
}
