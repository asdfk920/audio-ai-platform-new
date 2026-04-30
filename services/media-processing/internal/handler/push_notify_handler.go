package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func PushNotifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PushNotifyReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{
				"code":    errorx.CodeInvalidParam,
				"message": "参数错误",
				"data":    types.PushNotifyResp{Allowed: false, Reason: "invalid_params"},
			})
			return
		}

		l := logic.NewPushNotifyLogic(r.Context(), svcCtx)
		resp, err := l.PushNotify(&req)
		if err != nil {
			code := errorx.CodeOf(err)
			httpx.WriteJson(w, http.StatusOK, map[string]any{
				"code":    code,
				"message": err.Error(),
				"data":    types.PushNotifyResp{Allowed: false, Code: code, Reason: err.Error()},
			})
			return
		}
		httpx.WriteJson(w, http.StatusOK, resp)
	}
}
