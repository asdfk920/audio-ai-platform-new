package handler

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// queryPickFold 按「忽略大小写」的 query key 取第一个非空值（兼容 Title/title、Page_Size/page_size 等）
func queryPickFold(values url.Values, wantKey string) string {
	wantKey = strings.TrimSpace(wantKey)
	if wantKey == "" {
		return ""
	}
	for k, vv := range values {
		if !strings.EqualFold(strings.TrimSpace(k), wantKey) || len(vv) == 0 {
			continue
		}
		if v := strings.TrimSpace(vv[0]); v != "" {
			return v
		}
	}
	return ""
}

// parseContentListQuery 仅解析 QueryString，避免 httpx.Parse 在 GET + application/json body（如 Apifox 默认 {}）时 ParseJsonBody 失败。
func parseContentListQuery(r *http.Request) *types.ContentListReq {
	q := r.URL.Query()
	req := &types.ContentListReq{}

	if s := queryPickFold(q, "page"); s != "" {
		if page, err := strconv.ParseInt(s, 10, 32); err == nil {
			req.Page = int32(page)
		}
	}
	if s := queryPickFold(q, "page_size"); s != "" {
		if pageSize, err := strconv.ParseInt(s, 10, 32); err == nil {
			req.PageSize = int32(pageSize)
		}
	}
	if s := queryPickFold(q, "category_id"); s != "" {
		if categoryID, err := strconv.ParseInt(s, 10, 64); err == nil {
			req.CategoryID = categoryID
		}
	}
	req.TagIDs = queryPickFold(q, "tag_ids")
	req.Title = queryPickFold(q, "title")
	req.Keyword = queryPickFold(q, "keyword")
	if req.Keyword == "" {
		req.Keyword = queryPickFold(q, "q")
	}
	if req.Keyword == "" {
		req.Keyword = queryPickFold(q, "search")
	}
	if s := queryPickFold(q, "sort"); s != "" {
		if sort, err := strconv.ParseInt(s, 10, 32); err == nil {
			req.Sort = int32(sort)
		}
	}
	if s := queryPickFold(q, "is_vip"); s != "" {
		if isVip, err := strconv.ParseInt(s, 10, 32); err == nil {
			req.IsVip = int32(isVip)
		}
	}
	return req
}

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

		req := parseContentListQuery(r)

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
