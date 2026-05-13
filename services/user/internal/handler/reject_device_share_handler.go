package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// RejectDeviceShareHandler 拒绝设备共享处理器
// @Summary      拒绝设备共享
// @Description  拒绝其他用户共享给自己的设备
// @Tags         设备分享
// @Accept       json
// @Produce      json
// @Param        body  body      types.DeviceShareRejectReq  true  "拒绝设备共享请求"
// @Success      200  {object}  errorx.Response  "拒绝成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/device/share/reject [post]
// @Security     BearerAuth
//
// POST /api/v1/user/device/share/reject
// 用途：用户拒绝别人共享给自己的设备
func RejectDeviceShareHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeviceShareRejectReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		err := logic.NewRejectDeviceShareLogic(r.Context(), svcCtx).RejectDeviceShare(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.SuccessData("已拒绝设备共享", nil))
	}
}
