package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// @QuitDeviceShare 退出设备共享
// @Summary      退出设备共享
// @Description  退出已接受共享的设备
// @Tags         设备分享
// @Accept       json
// @Produce      json
// @Param        body  body      errorx.Response  true  "请求参数"
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/device/share/quit [post]
// @Security     BearerAuth
// QuitDeviceShareHandler 退出设备共享处理器
// POST /api/v1/user/device/share/quit
// 用途：共享用户主动退出设备共享
func QuitDeviceShareHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeviceShareQuitReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		if err := logic.NewQuitDeviceShareLogic(r.Context(), svcCtx).QuitDeviceShare(&req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.SuccessMsg("退出设备共享成功"))
	}
}
