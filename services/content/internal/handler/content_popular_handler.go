package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// contentPopularHandler 热门推荐处理器
// GET /api/v1/content/popular
// 可选登录：根据用户会员等级过滤内容
func contentPopularHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpresp.Write(w, http.StatusMethodNotAllowed, httpresp.MsgMethodNotAllowed+`：仅支持 GET`, nil)
			return
		}

		var req types.PopularReq
		if err := httpx.Parse(r, &req); err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `请求格式错误：`+err.Error()), nil)
			return
		}

		// 解析用户 Token（可选登录）
		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		userID := bearerCtx.UserID

		l := logic.NewContentPopularLogic(r.Context(), svcCtx)
		resp, err := l.ContentPopular(&req, userID)
		if err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, err.Error()), nil)
			return
		}

		httpresp.WriteSuccess(w, resp)
	}
}
