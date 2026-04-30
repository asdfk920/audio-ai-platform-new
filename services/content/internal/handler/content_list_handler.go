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

// contentListHandler 内容列表处理器
// GET /api/v1/content/list
// 可选登录：游客仅免费内容；带 Token 时按 JWT 解析用户会员等级
// 所有参数均为可选
func contentListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 GET",
				"data": nil,
			})
			return
		}

		// 手动解析查询参数，所有参数均为可选
		req := &types.ContentListReq{}

		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if page, err := strconv.ParseInt(pageStr, 10, 32); err == nil {
				req.Page = int32(page)
			}
		}

		if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
			if pageSize, err := strconv.ParseInt(pageSizeStr, 10, 32); err == nil {
				req.PageSize = int32(pageSize)
			}
		}

		if categoryIDStr := r.URL.Query().Get("category_id"); categoryIDStr != "" {
			if categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64); err == nil {
				req.CategoryID = categoryID
			}
		}

		req.TagIDs = r.URL.Query().Get("tag_ids")
		req.Keyword = r.URL.Query().Get("keyword")

		if sortStr := r.URL.Query().Get("sort"); sortStr != "" {
			if sort, err := strconv.ParseInt(sortStr, 10, 32); err == nil {
				req.Sort = int32(sort)
			}
		}

		if isVipStr := r.URL.Query().Get("is_vip"); isVipStr != "" {
			if isVip, err := strconv.ParseInt(isVipStr, 10, 32); err == nil {
				req.IsVip = int32(isVip)
			}
		}

		// 解析 Authorization Header 获取用户信息（可选）
		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		userID := bearerCtx.UserID

		l := logic.NewContentListLogic(r.Context(), svcCtx)
		resp, err := l.ContentList(req, userID)
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
