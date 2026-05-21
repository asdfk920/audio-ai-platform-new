package handler

import (
	"net/http"
	"strconv"

	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
)

// contentRecommendHandler 内容推荐处理器
// GET /api/v1/content/recommend
// 必须登录：需要用户 Token 提取用户特征
func contentRecommendHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpresp.Write(w, http.StatusMethodNotAllowed, httpresp.MsgMethodNotAllowed+`：仅支持 GET`, nil)
			return
		}

		// 手动解析参数，忽略不认识的参数
		req := &types.RecommendReq{}
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if limit, err := strconv.ParseInt(limitStr, 10, 32); err == nil {
				req.Limit = int32(limit)
			}
		}
		req.Type = r.URL.Query().Get("type")

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

		l := logic.NewContentRecommendLogic(r.Context(), svcCtx)
		resp, err := l.ContentRecommend(req, bearerCtx.UserID)
		if err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, err.Error()), nil)
			return
		}

		httpresp.WriteSuccess(w, resp)
	}
}
