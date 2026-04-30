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

// contentNewHandler 最新内容推荐处理器
// GET /api/v1/content/new
// 可选登录：根据用户会员等级过滤内容，登录后可查看订阅内容
func contentNewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 GET",
				"data": nil,
			})
			return
		}

		// 手动解析参数
		req := &types.NewContentReq{}
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if limit, err := strconv.ParseInt(limitStr, 10, 32); err == nil {
				req.Limit = int32(limit)
			}
		}
		if daysStr := r.URL.Query().Get("days"); daysStr != "" {
			if days, err := strconv.ParseInt(daysStr, 10, 32); err == nil {
				req.Days = int32(days)
			}
		}
		if categoryStr := r.URL.Query().Get("category"); categoryStr != "" {
			if category, err := strconv.ParseInt(categoryStr, 10, 64); err == nil {
				req.Category = category
			}
		}

		// 解析用户 Token（可选登录）
		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		userID := bearerCtx.UserID

		l := logic.NewContentNewLogic(r.Context(), svcCtx)
		resp, err := l.ContentNew(req, userID)
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
