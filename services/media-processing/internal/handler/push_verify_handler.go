package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func PushVerifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PushVerifyReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]any{"code": 40001, "msg": "参数错误", "data": types.PushVerifyResp{Allowed: false}})
			return
		}
		l := logic.NewPushVerifyLogic(r.Context(), svcCtx)
		resp, err := l.PushVerify(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]any{"code": errorx.CodeOf(err), "msg": err.Error(), "data": types.PushVerifyResp{Allowed: false, Msg: err.Error()}})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]any{"code": 0, "message": "ok", "data": resp})
	}
}
