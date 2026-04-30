package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func PushAddressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PushAddressReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"code": 40001, "msg": "参数错误"})
			return
		}
		l := logic.NewPushAddressLogic(r.Context(), svcCtx)
		resp, err := l.PushAddress(&req, r)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]any{"code": errorx.CodeOf(err), "msg": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]any{"code": 0, "message": "ok", "data": resp})
	}
}
