package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// @CreateDeviceShare 创建设备共享
// @Summary      创建设备共享
// @Description  创建设备共享，将设备权限分享给其他用户
// @Tags         设备分享
// @Accept       json
// @Produce      json
// @Param        body  body      errorx.Response  true  "请求参数"
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/device/share/create [post]
// @Security     BearerAuth
// CreateDeviceShareHandler 创建设备共享处理器
// POST /api/v1/user/device/share/create
// 用途：用户创建设备共享邀请，将设备共享给其他用户
func CreateDeviceShareHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeviceShareCreateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := logic.NewCreateDeviceShareLogic(r.Context(), svcCtx).CreateDeviceShare(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.SuccessData("设备共享邀请已创建", resp))
	}
}
