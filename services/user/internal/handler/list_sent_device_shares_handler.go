package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
)

// @ListSentDeviceShares 发出的共享列表
// @Summary      发出的共享列表
// @Description  查询当前用户发出的所有设备共享记录
// @Tags         设备分享
// @Accept       json
// @Produce      json
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/device/share/sent [get]
// @Security     BearerAuth
// ListSentDeviceSharesHandler 查询已发出的设备共享列表处理器
// GET /api/v1/user/device/share/sent/list
// 用途：查询当前用户已发出的设备共享记录
func ListSentDeviceSharesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := logic.NewListSentDeviceSharesLogic(r.Context(), svcCtx).ListSentDeviceShares()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.Success(resp))
	}
}
