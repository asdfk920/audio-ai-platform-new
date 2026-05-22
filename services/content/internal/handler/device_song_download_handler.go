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

// deviceSongDownloadHandler 设备歌曲下载处理器
// POST /api/v1/content/device/download
// 用户点击下载按钮，将歌曲下载到指定设备
func deviceSongDownloadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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

		var req types.DeviceSongDownloadReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `解析 JSON 失败: `+err.Error()), nil)
			return
		}

		l := logic.NewDeviceSongDownloadLogic(r.Context(), svcCtx)
		resp, err := l.DownloadSong(&req, bearerCtx.UserID)

		if err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, err.Error()), nil)
			return
		}

		httpresp.WriteSuccess(w, resp)
	}
}

// deviceDownloadCallbackHandler 设备下载回调处理器（设备微服务→内容微服务）
// POST /api/v1/content/device/download/callback
// 设备微服务调用此接口通知下载结果，需要内部鉴权或IP白名单验证
func deviceDownloadCallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpresp.Write(w, http.StatusMethodNotAllowed, httpresp.MsgMethodNotAllowed+`：仅支持 POST`, nil)
			return
		}

		var req types.DeviceDownloadCallbackReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `解析 JSON 失败: `+err.Error()), nil)
			return
		}

		l := logic.NewDeviceSongDownloadLogic(r.Context(), svcCtx)
		resp, err := l.HandleDownloadCallback(&req)

		if err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, err.Error()), nil)
			return
		}

		httpresp.WriteSuccess(w, resp)
	}
}

// deviceDownloadStatusHandler 查询设备下载状态处理器
// GET /api/v1/content/device/download/status?task_id=xxx
func deviceDownloadStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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

		taskID := r.URL.Query().Get("task_id")
		if taskID == "" {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `缺少 task_id 参数`), nil)
			return
		}

		l := logic.NewDeviceSongDownloadLogic(r.Context(), svcCtx)
		resp, err := l.GetDownloadStatus(taskID, bearerCtx.UserID)

		if err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, err.Error()), nil)
			return
		}

		httpresp.WriteSuccess(w, resp)
	}
}

// deviceDownloadListHandler 设备下载历史列表处理器
// GET /api/v1/content/device/downloads
// 支持参数：page, page_size, sn, status, song_name(歌名模糊查询), artist(艺术家模糊查询)
func deviceDownloadListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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

		var req types.DeviceDownloadListReq

		pageStr := r.URL.Query().Get("page")
		sizeStr := r.URL.Query().Get("page_size")
		sn := r.URL.Query().Get("sn")
		status := r.URL.Query().Get("status")
		songName := r.URL.Query().Get("song_name") // 🆕 歌曲名称模糊查询
		artist := r.URL.Query().Get("artist")      // 🆕 艺术家模糊查询

		if pageStr != "" {
			if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil {
				req.Page = int32(p)
			}
		}
		if sizeStr != "" {
			if s, err := strconv.ParseInt(sizeStr, 10, 32); err == nil {
				req.PageSize = int32(s)
			}
		}

		if sn != "" {
			req.DeviceSN = sn
		}
		if status != "" {
			req.Status = status
		}
		if songName != "" { // 🆕 解析歌名参数
			req.SongName = songName
		}
		if artist != "" { // 🆕 解析艺术家参数
			req.Artist = artist
		}

		l := logic.NewDeviceSongDownloadLogic(r.Context(), svcCtx)
		resp, err := l.GetDownloadList(&req, bearerCtx.UserID)

		if err != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, err.Error()), nil)
			return
		}

		httpresp.WriteSuccess(w, resp)
	}
}
