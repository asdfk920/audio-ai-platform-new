package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
)

// contentDownloadDeleteHandler 删除下载记录处理器
// DELETE /api/v1/content/downloads
// 必须登录：只能删除自己的下载记录
func contentDownloadDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			httpresp.Write(w, http.StatusMethodNotAllowed, httpresp.MsgMethodNotAllowed+`：仅支持 DELETE`, nil)
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

		var req types.DownloadDeleteReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `解析 JSON 失败`), nil)
			return
		}

		l := logic.NewContentDownloadLogic(r.Context(), svcCtx)
		err := l.DeleteDownloadRecords(&req, bearerCtx.UserID)
		if err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, err.Error()), nil)
			return
		}

		httpresp.WriteSuccessMsg(w, `删除成功`, nil)
	}
}

// contentDownloadListHandler 下载历史列表处理器
// GET /api/v1/content/downloads
// 必须登录：查看自己的下载历史
func contentDownloadListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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

		var req types.DownloadListReq
		pageStr := r.URL.Query().Get("page")
		sizeStr := r.URL.Query().Get("size")
		if pageStr != "" {
			if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil {
				req.Page = int32(p)
			}
		}
		if sizeStr != "" {
			if s, err := strconv.ParseInt(sizeStr, 10, 32); err == nil {
				req.Size = int32(s)
			}
		}

		l := logic.NewContentDownloadLogic(r.Context(), svcCtx)
		resp, err := l.GetDownloadList(&req, bearerCtx.UserID)
		if err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, err.Error()), nil)
			return
		}

		httpresp.WriteSuccess(w, resp)
	}
}
