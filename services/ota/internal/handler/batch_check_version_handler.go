package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/ota/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/ota/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/ota/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/ota/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func BatchCheckVersionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := jwt.GetUserIdFromContext(r.Context())
		if !ok {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code": 401,
				"msg":  "未获取到用户信息",
				"data": nil,
			})
			return
		}

		var req types.BatchCheckVersionReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewBatchCheckVersionLogic(r.Context(), svcCtx)
		resp, err := l.BatchCheckVersion(userID, &req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
