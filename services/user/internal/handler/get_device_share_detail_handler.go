package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// GetDeviceShareDetailHandler 查询设备共享详情处理器
// @Summary      共享详情
// @Description  查询指定设备共享的详细信息
// @Tags         设备分享
// @Accept       json
// @Produce      json
// @Param        share_id  query  string  true  "共享 ID"
// @Success      200  {object}  types.DeviceShareItem  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/device/share/detail [get]
// @Security     BearerAuth
// GET /api/v1/user/device/share/detail
// 用途：查询指定设备共享的详细信息
func GetDeviceShareDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeviceShareDetailReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := logic.NewGetDeviceShareDetailLogic(r.Context(), svcCtx).GetDeviceShareDetail(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.Success(resp))
	}
}
