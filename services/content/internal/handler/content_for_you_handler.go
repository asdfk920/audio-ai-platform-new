package handler

import (
	"net/http"
	"strconv"

	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// contentForYouHandler 猜你喜欢处理器
// GET /api/v1/content/for-you
// 必须登录：需要用户 Token 提取用户画像
func contentForYouHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 GET",
				"data": nil,
			})
			return
		}

		req := &types.ForYouReq{}
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if limit, err := strconv.ParseInt(limitStr, 10, 32); err == nil {
				req.Limit = int32(limit)
			}
		}
		req.ExcludeIDs = r.URL.Query().Get("exclude_ids")

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code": 401,
				"msg":  "请先登录",
				"data": nil,
			})
			return
		}

		l := logic.NewContentForYouLogic(r.Context(), svcCtx)
		resp, err := l.ForYou(req, bearerCtx.UserID)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  err.Error(),
				"data": nil,
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"msg":  "获取成功",
			"data": resp,
		})
	}
}
