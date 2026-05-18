// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

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

// 接受设备共享邀请（单条 share_id/sn，或批量 devices / list）
func acceptDeviceShareHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeviceShareAcceptReq

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

		l := logic.NewAcceptDeviceShareLogic(r.Context(), svcCtx)

		batch := append(append([]types.DeviceShareAcceptDeviceRef(nil), req.Devices...), req.List...)
		if len(batch) > 0 {
			batchResp, err := l.AcceptDeviceSharesBatch(batch)
			if err != nil {
				var ce *errorx.CodeError
				if !errors.As(err, &ce) {
					err = errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
				}
				httpx.ErrorCtx(r.Context(), w, err)
				return
			}
			httpx.OkJsonCtx(r.Context(), w, batchResp)
			return
		}

		resp, err := l.AcceptDeviceShare(&req)
		if err != nil {
			var ce *errorx.CodeError
			if !errors.As(err, &ce) {
				err = errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
			}
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
