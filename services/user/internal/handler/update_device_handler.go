package handler

import (
	"errors"
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateDeviceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateDeviceReq
		if err := httpx.Parse(r, &req); err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, err.Error()), nil)
			return
		}

		l := logic.NewUpdateDeviceLogic(r.Context(), svcCtx)
		resp, err := l.UpdateDevice(&req)

		if err != nil {
			var ce *errorx.CodeError
			if errors.As(err, &ce) {
				st := errorx.HTTPStatusForCode(ce.Code)
				httpresp.Write(w, st, ce.Msg, nil)
				return
			}
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.MsgInternal, err.Error()), nil)
			return
		}

		httpresp.WriteSuccessMsg(w, "修改成功", resp)
	}
}
