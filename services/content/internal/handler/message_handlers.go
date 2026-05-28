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

// GET /api/v1/user/messages - 获取消息列表
func messageListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}

		var req types.MessageListReq
		if err := httpx.Parse(r, &req); err != nil {
			req.Page = 1
			req.PageSize = 20
		}

		l := logic.NewMessageLogic(r.Context(), svcCtx)
		resp, err := l.GetMessageList(bearer.UserID, &req)
		if err != nil {
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), err.Error()), nil)
			return
		}

		logx.Infof("[MessageList] userID=%d total=%d unreadCount=%d",
			bearer.UserID, resp.Total, resp.UnreadCount)
		httpresp.WriteSuccess(w, resp)
	}
}

// POST /api/v1/user/messages/read - 标记消息已读
func messageMarkReadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}

		var req types.MarkMessageReadReq
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

		l := logic.NewMessageLogic(r.Context(), svcCtx)
		resp, err := l.MarkMessageRead(bearer.UserID, req.MessageIDs)
		if err != nil {
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), err.Error()), nil)
			return
		}

		logx.Infof("[MessageMarkRead] userID=%d affectedRows=%d remainingUnread=%d",
			bearer.UserID, resp.AffectedRows, resp.UnreadCount)
		httpresp.WriteSuccessMsg(w, resp.Message, resp)
	}
}

// GET /api/v1/user/messages/unread-count - 获取未读消息数量
func messageUnreadCountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}

		l := logic.NewMessageLogic(r.Context(), svcCtx)
		unreadCount, err := l.GetUnreadCount(bearer.UserID)
		if err != nil {
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), err.Error()), nil)
			return
		}

		logx.Infof("[MessageUnreadCount] userID=%d unreadCount=%d", bearer.UserID, unreadCount)
		httpresp.WriteSuccess(w, &types.UnreadCountResp{
			UnreadCount: unreadCount,
		})
	}
}
