package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// POST /api/v1/content/artists/:id/subscribe
func artistSubscribeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}

		parts := pathSegments(r)
		artistID, err := parseArtistIDFromPath(parts, "subscribe")
		if err != nil || artistID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "艺术家 ID 格式无效或URL路径错误"), nil)
			return
		}

		l := logic.NewArtistSubscribeLogic(r.Context(), svcCtx)
		resp, err := l.Subscribe(artistID, bearer.UserID)
		if err != nil {
			switch err.Error() {
			case "用户未登录":
				httpresp.Write(w, http.StatusUnauthorized, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusUnauthorized), err.Error()), nil)
			case "艺术家不存在或已下架":
				httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusNotFound), err.Error()), nil)
			default:
				httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), err.Error()), nil)
			}
			return
		}

		logx.Infof("[ArtistSubscribe] userID=%d artistID=%d success=%v", bearer.UserID, artistID, resp.Success)
		httpresp.WriteSuccessMsg(w, resp.Message, resp)
	}
}

// POST /api/v1/content/artists/:id/unsubscribe
func artistUnsubscribeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}

		parts := pathSegments(r)
		artistID, err := parseArtistIDFromPath(parts, "unsubscribe")
		if err != nil || artistID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "艺术家 ID 格式无效或URL路径错误"), nil)
			return
		}

		l := logic.NewArtistSubscribeLogic(r.Context(), svcCtx)
		resp, err := l.Unsubscribe(artistID, bearer.UserID)
		if err != nil {
			switch err.Error() {
			case "用户未登录":
				httpresp.Write(w, http.StatusUnauthorized, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusUnauthorized), err.Error()), nil)
			case "艺术家不存在":
				httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusNotFound), err.Error()), nil)
			default:
				httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), err.Error()), nil)
			}
			return
		}

		logx.Infof("[ArtistUnsubscribe] userID=%d artistID=%d success=%v", bearer.UserID, artistID, resp.Success)
		httpresp.WriteSuccessMsg(w, resp.Message, resp)
	}
}

// GET /api/v1/content/artists/:id/subscription-status
func artistSubscriptionStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}

		parts := pathSegments(r)
		artistID, err := parseArtistIDFromPath(parts, "subscription-status")
		if err != nil || artistID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "艺术家 ID 格式无效或URL路径错误"), nil)
			return
		}

		l := logic.NewArtistSubscribeLogic(r.Context(), svcCtx)
		isSubscribed, err := l.CheckSubscriptionStatus(artistID, bearer.UserID)
		if err != nil {
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), err.Error()), nil)
			return
		}

		httpresp.WriteSuccess(w, map[string]interface{}{
			"artist_id":     artistID,
			"is_subscribed": isSubscribed,
		})
	}
}

// GET /api/v1/content/my/subscriptions/artists
func myArtistSubscriptionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}

		var req types.SubscribeListReq
		if err := httpx.Parse(r, &req); err != nil {
			req.Page = 1
			req.PageSize = 20
		}
		if req.Page <= 0 {
			req.Page = 1
		}
		if req.PageSize <= 0 || req.PageSize > 100 {
			req.PageSize = 20
		}

		l := logic.NewArtistSubscribeLogic(r.Context(), svcCtx)
		resp, err := l.GetSubscriptionList(bearer.UserID, req.Page, req.PageSize)
		if err != nil {
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), err.Error()), nil)
			return
		}

		logx.Infof("[MyArtistSubscriptions] userID=%d total=%d page=%d pageSize=%d",
			bearer.UserID, resp.Total, resp.Page, resp.PageSize)
		httpresp.WriteSuccess(w, resp)
	}
}

// POST /api/v1/content/subscribe (通用订阅接口，支持歌曲/歌手/专辑)
func contentSubscribeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}

		var req types.ArtistSubscribeReq
		if r.Body != nil && r.ContentLength > 0 {
			body, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "读取请求体失败"), nil)
				return
			}
			if len(body) > 0 {
				if unmarshalErr := json.Unmarshal(body, &req); unmarshalErr != nil {
					httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "JSON解析失败: "+unmarshalErr.Error()), nil)
					return
				}
			}
		}

		if req.ArtistID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "艺术家 ID 不能为空"), nil)
			return
		}

		l := logic.NewArtistSubscribeLogic(r.Context(), svcCtx)
		resp, err := l.Subscribe(req.ArtistID, bearer.UserID)
		if err != nil {
			switch err.Error() {
			case "用户未登录":
				httpresp.Write(w, http.StatusUnauthorized, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusUnauthorized), err.Error()), nil)
			case "艺术家不存在或已下架":
				httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusNotFound), err.Error()), nil)
			default:
				httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), err.Error()), nil)
			}
			return
		}

		logx.Infof("[ContentSubscribe] userID=%d artistID=%d success=%v", bearer.UserID, req.ArtistID, resp.Success)
		httpresp.WriteSuccessMsg(w, resp.Message, resp)
	}
}

// POST /api/v1/content/unsubscribe (通用取消订阅接口)
func contentUnsubscribeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}

		var req types.ArtistUnsubscribeReq
		if r.Body != nil && r.ContentLength > 0 {
			body, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "读取请求体失败"), nil)
				return
			}
			if len(body) > 0 {
				if unmarshalErr := json.Unmarshal(body, &req); unmarshalErr != nil {
					httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "JSON解析失败: "+unmarshalErr.Error()), nil)
					return
				}
			}
		}

		if req.ArtistID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "艺术家 ID 不能为空"), nil)
			return
		}

		l := logic.NewArtistSubscribeLogic(r.Context(), svcCtx)
		resp, err := l.Unsubscribe(req.ArtistID, bearer.UserID)
		if err != nil {
			switch err.Error() {
			case "用户未登录":
				httpresp.Write(w, http.StatusUnauthorized, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusUnauthorized), err.Error()), nil)
			case "艺术家不存在":
				httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusNotFound), err.Error()), nil)
			default:
				httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), err.Error()), nil)
			}
			return
		}

		logx.Infof("[ContentUnsubscribe] userID=%d artistID=%d success=%v", bearer.UserID, req.ArtistID, resp.Success)
		httpresp.WriteSuccessMsg(w, resp.Message, resp)
	}
}
