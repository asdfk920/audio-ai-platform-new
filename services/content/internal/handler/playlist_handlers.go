package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"gorm.io/gorm"
)

// GET /api/v1/content/playlists — 我的歌单列表
func contentPlaylistListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}

		var req types.PlaylistListReq
		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
				req.Page = int32(p)
			}
		}
		if sizeStr := r.URL.Query().Get("page_size"); sizeStr != "" {
			if s, err := strconv.ParseInt(sizeStr, 10, 32); err == nil && s > 0 {
				req.PageSize = int32(s)
			}
		}
		req.Source = strings.TrimSpace(r.URL.Query().Get("source"))
		if v := strings.TrimSpace(r.URL.Query().Get("include_deleted")); v == "1" || strings.EqualFold(v, "true") {
			req.IncludeDeleted = true
		}

		l := logic.NewPlaylistListLogic(r.Context(), svcCtx)
		resp, err := l.List(bearer.UserID, &req)
		if err != nil {
			errMsg := err.Error()
			switch {
			case strings.Contains(errMsg, "未登录"):
				httpresp.Write(w, http.StatusUnauthorized, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusUnauthorized), errMsg), nil)
			default:
				httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), errMsg), nil)
			}
			return
		}
		httpresp.WriteSuccess(w, resp)
	}
}

// POST /api/v1/content/playlists
func contentPlaylistCreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}

		var req types.PlaylistCreateReq
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "读取请求体失败"), nil)
				return
			}
			_ = json.Unmarshal(body, &req)
		} else {
			_ = r.ParseForm()
			req.Name = r.FormValue("name")
			req.Description = r.FormValue("description")
			req.CoverURL = r.FormValue("cover_url")
			if v := r.FormValue("is_public"); v == "false" || v == "0" {
				b := false
				req.IsPublic = &b
			} else if v == "true" || v == "1" {
				b := true
				req.IsPublic = &b
			}
		}

		l := logic.NewPlaylistCreateLogic(r.Context(), svcCtx)
		resp, err := l.Create(&req, bearer.UserID)
		if err != nil {
			errMsg := err.Error()
			switch {
			case strings.Contains(errMsg, "未登录"):
				httpresp.Write(w, http.StatusUnauthorized, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusUnauthorized), errMsg), nil)
			case strings.Contains(errMsg, "不能为空"), strings.Contains(errMsg, "长度"), strings.Contains(errMsg, "不能超过"), strings.Contains(errMsg, "已存在"):
				httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), errMsg), nil)
			case strings.Contains(errMsg, "已达到最大"):
				httpresp.Write(w, http.StatusForbidden, httpresp.WithDetail(httpresp.MsgForbidden, errMsg), nil)
			default:
				httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), errMsg), nil)
			}
			return
		}
		httpresp.WriteSuccess(w, resp)
	}
}

// PUT /api/v1/content/playlists/:id
func contentPlaylistUpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}
		playlistID, err := parsePlaylistID(r)
		if err != nil || playlistID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "歌单 ID 格式无效"), nil)
			return
		}

		var req types.PlaylistUpdateReq
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			body, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "读取请求体失败"), nil)
				return
			}
			_ = json.Unmarshal(body, &req)
		} else {
			_ = r.ParseForm()
			req.Name = r.FormValue("name")
			req.Description = r.FormValue("description")
			req.CoverURL = r.FormValue("cover_url")
			if v := r.FormValue("is_public"); v != "" {
				val, perr := strconv.ParseInt(v, 10, 16)
				if perr == nil {
					ip := int16(val)
					req.IsPublic = &ip
				}
			}
		}

		name := strings.TrimSpace(req.Name)
		description := strings.TrimSpace(req.Description)
		if name != "" && (len(name) < 1 || len(name) > 100) {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "歌单名称长度必须在1-100个字符之间"), nil)
			return
		}
		if description != "" && len(description) > 500 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "歌单描述长度不能超过500个字符"), nil)
			return
		}

		var playlist struct {
			ID          int64
			UserID      int64
			Name        string
			Description string
			CoverURL    string
			SongCount   int
			IsPublic    int16
		}
		if err := svcCtx.DB.Table("playlists").
			Select("id, user_id, name, description, cover_url, song_count, is_public").
			Where("id = ? AND status = 1 AND deleted_at IS NULL", playlistID).
			First(&playlist).Error; err != nil {
			httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusNotFound), "歌单不存在"), nil)
			return
		}
		if playlist.UserID != bearer.UserID {
			httpresp.Write(w, http.StatusForbidden, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusForbidden), "无权限修改此歌单"), nil)
			return
		}

		updates := map[string]interface{}{}
		if name != "" && name != playlist.Name {
			updates["name"] = name
		}
		if description != "" && description != playlist.Description {
			updates["description"] = description
		}
		coverURL := strings.TrimSpace(req.CoverURL)
		if coverURL != "" && coverURL != playlist.CoverURL {
			if playlist.CoverURL != "" && !strings.HasPrefix(playlist.CoverURL, "http") &&
				!strings.HasPrefix(playlist.CoverURL, "/static/default") {
				if p := getLocalFilePath(svcCtx, playlist.CoverURL); p != "" {
					_ = os.Remove(p)
				}
			}
			updates["cover_url"] = coverURL
		}
		if req.IsPublic != nil && *req.IsPublic != playlist.IsPublic {
			if *req.IsPublic == 0 || *req.IsPublic == 1 {
				updates["is_public"] = *req.IsPublic
			}
		}
		if len(updates) == 0 {
			httpresp.WriteSuccess(w, types.PlaylistUpdateResp{
				ID: playlist.ID, Name: playlist.Name, Description: playlist.Description,
				CoverURL: playlist.CoverURL, SongCount: playlist.SongCount, IsPublic: playlist.IsPublic,
			})
			return
		}
		now := time.Now()
		updates["updated_at"] = now
		if err := svcCtx.DB.Table("playlists").Where("id = ?", playlistID).Updates(updates).Error; err != nil {
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), "更新歌单失败"), nil)
			return
		}
		_ = svcCtx.DB.Table("playlists").Select("name, description, cover_url, is_public").
			Where("id = ?", playlistID).Scan(&playlist)
		httpresp.WriteSuccess(w, types.PlaylistUpdateResp{
			ID: playlist.ID, Name: playlist.Name, Description: playlist.Description,
			CoverURL: playlist.CoverURL, SongCount: playlist.SongCount, IsPublic: playlist.IsPublic,
			UpdatedAt: now.Format("2006-01-02 15:04:05"),
		})
	}
}

// DELETE /api/v1/content/playlists/:id
func contentPlaylistDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}
		playlistID, err := parsePlaylistID(r)
		if err != nil || playlistID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "歌单 ID 格式无效"), nil)
			return
		}

		var playlist struct {
			ID        int64
			UserID    int64
			Name      string
			IsDefault int16
			DeletedAt *time.Time
		}
		if err := svcCtx.DB.Table("playlists").Select("id, user_id, name, is_default, deleted_at").
			Where("id = ?", playlistID).First(&playlist).Error; err != nil {
			httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusNotFound), "歌单不存在"), nil)
			return
		}
		if playlist.DeletedAt != nil {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "歌单已被删除"), nil)
			return
		}
		if playlist.UserID != bearer.UserID {
			httpresp.Write(w, http.StatusForbidden, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusForbidden), "无权限删除此歌单"), nil)
			return
		}
		if playlist.IsDefault == 1 {
			httpresp.Write(w, http.StatusForbidden, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusForbidden), "默认歌单不能删除"), nil)
			return
		}

		_ = svcCtx.DB.Table("playlist_songs").Where("playlist_id = ?", playlistID).Delete(nil)
		now := time.Now()
		if err := svcCtx.DB.Table("playlists").Where("id = ?", playlistID).Updates(map[string]interface{}{
			"status": 2, "deleted_at": now, "updated_at": now,
		}).Error; err != nil {
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), "删除歌单失败"), nil)
			return
		}
		httpresp.WriteSuccess(w, types.PlaylistDeleteResp{Success: true, Message: "歌单删除成功"})
	}
}

// POST /api/v1/content/playlists/:id/songs
func contentPlaylistAddSongHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}
		playlistID, err := parsePlaylistID(r)
		if err != nil || playlistID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "歌单 ID 格式无效"), nil)
			return
		}

		var req types.PlaylistAddSongReq
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			body, readErr := io.ReadAll(r.Body)
			if readErr == nil {
				_ = json.Unmarshal(body, &req)
			}
		} else {
			_ = r.ParseForm()
			if id, perr := strconv.ParseInt(r.FormValue("content_id"), 10, 64); perr == nil {
				req.ContentID = id
			}
		}
		if req.ContentID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "歌曲 ID 不能为空"), nil)
			return
		}

		var playlist struct {
			ID       int64
			UserID   int64
			Name     string
			SongCount int
			IsPublic int16
		}
		if err := svcCtx.DB.Table("playlists").
			Select("id, user_id, name, song_count, is_public").
			Where("id = ? AND status = 1 AND deleted_at IS NULL", playlistID).
			First(&playlist).Error; err != nil {
			httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusNotFound), "歌单不存在"), nil)
			return
		}
		if playlist.UserID != bearer.UserID {
			httpresp.Write(w, http.StatusForbidden, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusForbidden), "无权限操作此歌单"), nil)
			return
		}

		var songID int64
		if err := svcCtx.DB.Table("content").Select("id").
			Where("id = ? AND status = 1 AND is_deleted = 0", req.ContentID).
			Scan(&songID).Error; err != nil || songID <= 0 {
			httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusNotFound), "歌曲不存在或已下架"), nil)
			return
		}

		var exist int64
		_ = svcCtx.DB.Table("playlist_songs").Where("playlist_id = ? AND content_id = ?", playlistID, req.ContentID).Count(&exist)
		if exist > 0 {
			httpresp.WriteSuccessMsg(w, "歌曲已在歌单中", types.PlaylistAddSongResp{
				Success: false, Message: "歌曲已在歌单中", SongCount: playlist.SongCount,
			})
			return
		}

		var maxSort int
		_ = svcCtx.DB.Table("playlist_songs").Select("COALESCE(MAX(sort_order), -1)").
			Where("playlist_id = ?", playlistID).Scan(&maxSort)
		now := time.Now()
		if err := svcCtx.DB.Table("playlist_songs").Create(map[string]interface{}{
			"playlist_id": playlistID, "content_id": req.ContentID,
			"sort_order": maxSort + 1, "created_at": now,
		}).Error; err != nil {
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), "添加歌曲到歌单失败"), nil)
			return
		}
		_ = svcCtx.DB.Table("playlists").Where("id = ?", playlistID).
			Updates(map[string]interface{}{
				"song_count": gorm.Expr("COALESCE(song_count, 0) + 1"), "updated_at": now,
			})
		var newCount int
		_ = svcCtx.DB.Table("playlists").Select("song_count").Where("id = ?", playlistID).Scan(&newCount)
		httpresp.WriteSuccessMsg(w, "添加歌曲到歌单成功", types.PlaylistAddSongResp{
			Success: true, Message: "添加歌曲到歌单成功", SongCount: newCount,
		})
	}
}

// DELETE /api/v1/content/playlists/:id/songs/:songId
func contentPlaylistRemoveSongHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bearer, ok := requireAuth(w, r, svcCtx)
		if !ok {
			return
		}
		playlistID, songID, err := parsePlaylistAndSongID(r)
		if err != nil || playlistID <= 0 || songID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusBadRequest), "歌单或歌曲 ID 格式无效"), nil)
			return
		}

		var playlist struct {
			ID     int64
			UserID int64
		}
		if err := svcCtx.DB.Table("playlists").
			Select("id, user_id").
			Where("id = ? AND status = 1 AND deleted_at IS NULL", playlistID).
			First(&playlist).Error; err != nil {
			httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusNotFound), "歌单不存在"), nil)
			return
		}
		if playlist.UserID != bearer.UserID {
			httpresp.Write(w, http.StatusForbidden, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusForbidden), "无权限操作此歌单"), nil)
			return
		}

		res := svcCtx.DB.Table("playlist_songs").
			Where("playlist_id = ? AND content_id = ?", playlistID, songID).
			Delete(nil)
		if res.Error != nil {
			httpresp.Write(w, http.StatusInternalServerError, httpresp.WithDetail(httpresp.DefaultMsg(http.StatusInternalServerError), "移除歌曲失败"), nil)
			return
		}
		if res.RowsAffected > 0 {
			now := time.Now()
			_ = svcCtx.DB.Table("playlists").Where("id = ?", playlistID).
				Updates(map[string]interface{}{
					"song_count": gorm.Expr("GREATEST(COALESCE(song_count, 0) - 1, 0)"),
					"updated_at": now,
				})
		}

		var remain int64
		_ = svcCtx.DB.Table("playlist_songs").Where("playlist_id = ?", playlistID).Count(&remain)
		httpresp.WriteSuccess(w, types.PlaylistSongRemoveResp{
			PlaylistID:  strconv.FormatInt(playlistID, 10),
			RemainTotal: remain,
		})
	}
}
