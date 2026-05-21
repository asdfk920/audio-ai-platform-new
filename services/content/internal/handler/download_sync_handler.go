package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
)

// downloadSyncAddHandler 加入同步队列处理器
// POST /api/v1/content/downloads/sync/add
// 必须登录：只能操作自己的下载记录
func downloadSyncAddHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpresp.Write(w, http.StatusMethodNotAllowed, httpresp.MsgMethodNotAllowed+`：仅支持 POST`, nil)
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

		var req types.DownloadSyncReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `解析 JSON 失败`), nil)
			return
		}

		l := logic.NewDownloadSyncLogic(r.Context(), svcCtx)
		err := l.AddToSyncQueue(&req, bearerCtx.UserID)
		if err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, err.Error()), nil)
			return
		}

		httpresp.WriteSuccessMsg(w, `已加入同步队列`, nil)
	}
}

// downloadSyncListHandler 待同步列表处理器
// GET /api/v1/content/downloads/sync/list
// 必须登录：查看自己的待同步记录
func downloadSyncListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpresp.Write(w, http.StatusMethodNotAllowed, httpresp.MsgMethodNotAllowed+`：仅支持 GET`, nil)
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

		l := logic.NewDownloadSyncLogic(r.Context(), svcCtx)
		resp, err := l.GetSyncList(bearerCtx.UserID)
		if err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, err.Error()), nil)
			return
		}

		httpresp.WriteSuccess(w, resp)
	}
}

// downloadSyncConfirmHandler 确认删除处理器
// POST /api/v1/content/downloads/sync/confirm
// 必须登录：只能确认删除自己的记录
func downloadSyncConfirmHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpresp.Write(w, http.StatusMethodNotAllowed, httpresp.MsgMethodNotAllowed+`：仅支持 POST`, nil)
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

		var req types.DownloadSyncConfirmReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `解析 JSON 失败`), nil)
			return
		}

		l := logic.NewDownloadSyncLogic(r.Context(), svcCtx)
		resp, err := l.ConfirmSyncDelete(&req, bearerCtx.UserID)
		if err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, err.Error()), nil)
			return
		}

		httpresp.WriteSuccessMsg(w, `同步完成`, resp)
	}
}

// downloadSyncCancelHandler 取消同步处理器
// POST /api/v1/content/downloads/sync/cancel
// 必须登录：只能取消自己的同步记录
func downloadSyncCancelHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpresp.Write(w, http.StatusMethodNotAllowed, httpresp.MsgMethodNotAllowed+`：仅支持 POST`, nil)
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

		var req types.DownloadSyncCancelReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `解析 JSON 失败`), nil)
			return
		}

		l := logic.NewDownloadSyncLogic(r.Context(), svcCtx)
		err := l.CancelSync(&req, bearerCtx.UserID)
		if err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, err.Error()), nil)
			return
		}

		httpresp.WriteSuccessMsg(w, `取消成功`, nil)
	}
}
