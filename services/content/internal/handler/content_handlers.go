package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// GET /api/v1/content/list
func contentListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ContentListReq
		_ = httpx.Parse(r, &req)

		bearer := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		l := logic.NewContentListLogic(r.Context(), svcCtx)
		resp, err := l.List(&req, bearer.UserID)
		if err != nil {
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), err.Error()), nil)
			return
		}
		httpresp.WriteSuccess(w, resp)
	}
}

// GET /api/v1/content/:id
func contentDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}
		parts := pathSegments(r)
		if len(parts) == 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "内容 ID 不能为空"), nil)
			return
		}
		contentID, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
		if err != nil || contentID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "内容 ID 格式无效"), nil)
			return
		}

		l := logic.NewContentDetailLogic(r.Context(), svcCtx)
		resp, err := l.Detail(contentID, bearer.UserID)
		if err != nil {
			if strings.Contains(err.Error(), "不存在") {
				httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusNotFound), err.Error()), nil)
				return
			}
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), err.Error()), nil)
			return
		}
		httpresp.WriteSuccess(w, resp)
	}
}

// POST /api/v1/content/:id/like
func contentLikeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}
		contentID, err := parseIDBeforeSuffix(r, "like")
		if err != nil || contentID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "内容 ID 格式无效"), nil)
			return
		}

		l := logic.NewContentLikeLogic(r.Context(), svcCtx)
		resp, err := l.ToggleLike(contentID, bearer.UserID)
		if err != nil {
			switch err.Error() {
			case "歌曲不存在或已下架":
				httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusNotFound), err.Error()), nil)
			case "用户未登录":
				httpresp.Write(w, http.StatusUnauthorized, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusUnauthorized), err.Error()), nil)
			default:
				httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), err.Error()), nil)
			}
			return
		}
		httpresp.WriteSuccessMsg(w, resp.Message, resp)
	}
}

// POST /api/v1/content/:id/favorite
func contentFavoriteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}
		contentID, err := parseIDBeforeSuffix(r, "favorite")
		if err != nil || contentID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "内容 ID 格式无效"), nil)
			return
		}

		var req types.ContentFavoriteReq
		if r.Body != nil {
			body, readErr := io.ReadAll(r.Body)
			if readErr == nil && len(body) > 0 {
				_ = json.Unmarshal(body, &req)
			}
		}
		if req.ContentID <= 0 {
			req.ContentID = contentID
		}

		l := logic.NewContentFavoriteLogic(r.Context(), svcCtx)
		resp, err := l.AddFavorite(req.ContentID, bearer.UserID, req.FavoriteType)
		if err != nil {
			switch err.Error() {
			case "歌曲不存在或已下架":
				httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusNotFound), err.Error()), nil)
			case "用户未登录":
				httpresp.Write(w, http.StatusUnauthorized, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusUnauthorized), err.Error()), nil)
			default:
				httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), err.Error()), nil)
			}
			return
		}
		logx.Infof("[Favorite] userID=%d contentID=%d favorited=%v", bearer.UserID, contentID, resp.Favorited)
		httpresp.WriteSuccessMsg(w, resp.Message, resp)
	}
}
