package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// @QuitDeviceShare 退出设备共享
// @Summary      退出设备共享
// @Description  退出已接受共享的设备（share_id 或 sn）
// @Tags         设备分享
// @Accept       json
// @Produce      json
// @Param        body  body      types.DeviceShareQuitReq  true  "请求参数"
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

		const maxBody = 1 << 20
		raw, errRead := io.ReadAll(io.LimitReader(r.Body, maxBody))
		if errRead != nil {
			httpx.ErrorCtx(r.Context(), w, errorx.NewCodeError(errorx.CodeInvalidParam, "读取请求体失败"))
			return
		}
		if len(strings.TrimSpace(string(raw))) > 0 {
			if err := json.Unmarshal(raw, &req); err != nil {
				httpx.ErrorCtx(r.Context(), w, errorx.NewCodeError(errorx.CodeInvalidParam, "请求体不是合法 JSON"))
				return
			}
		}

		if err := logic.NewQuitDeviceShareLogic(r.Context(), svcCtx).QuitDeviceShare(&req); err != nil {
			var ce *errorx.CodeError
			if !errors.As(err, &ce) {
				err = errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
			}
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.SuccessMsg("退出设备共享成功"))
	}
}
