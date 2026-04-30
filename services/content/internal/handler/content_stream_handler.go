package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"gorm.io/gorm"
)

// contentStreamHandler 流媒体播放处理器
// GET /api/v1/content/stream/:id
// 必须登录，支持 Range 请求实现边下载边播放
func contentStreamHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 GET",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 4 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "内容 ID 不能为空",
				"data":    nil,
			})
			return
		}

		contentID, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
		if err != nil || contentID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "内容 ID 格式错误",
				"data":    nil,
			})
			return
		}

		var content struct {
			ID        int64
			Title     string
			AudioURL  string
			VipLevel  int16
			Status    int16
			IsDeleted int16
		}
		err = svcCtx.DB.Table("content").
			Select("id, title, audio_url, vip_level, status, is_deleted").
			Where("id = ? AND status = 1 AND is_deleted = 0", contentID).
			First(&content).Error
		if err != nil {
			logx.Errorf("查询内容失败: %v", err)
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "内容不存在或已下架",
				"data":    nil,
			})
			return
		}

		userVipLevel := int16(0)
		var vipLevel int16
		err = svcCtx.DB.Table("user_vip").
			Select("vip_level").
			Where("user_id = ?", bearerCtx.UserID).
			First(&vipLevel).Error
		if err == nil {
			userVipLevel = vipLevel
		}

		if content.VipLevel > userVipLevel {
			httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
				"code":    403,
				"message": "该内容为VIP专属，请先升级会员",
				"data":    nil,
			})
			return
		}

		if content.AudioURL == "" {
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "音频文件不存在",
				"data":    nil,
			})
			return
		}

		fileName := fmt.Sprintf("%s%s", content.Title, getFileExtFromURL(content.AudioURL))
		contentType := getContentTypeFromExt(content.AudioURL)

		localPath := getLocalFilePath(svcCtx, content.AudioURL)
		if localPath == "" {
			logx.Errorf("无法解析本地文件路径: %s", content.AudioURL)
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "音频文件路径无效",
				"data":    nil,
			})
			return
		}

		file, err := os.Open(localPath)
		if err != nil {
			logx.Errorf("打开音频文件失败: %v, path=%s", err, localPath)
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "音频文件不存在",
				"data":    nil,
			})
			return
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			logx.Errorf("获取文件信息失败: %v", err)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "文件信息获取失败",
				"data":    nil,
			})
			return
		}

		fileSize := fileInfo.Size()

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", fileName))
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))

		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" && strings.HasPrefix(rangeHeader, "bytes=") {
			rangeStr := strings.TrimPrefix(rangeHeader, "bytes=")
			rangeParts := strings.SplitN(rangeStr, "-", 2)

			var start, end int64
			if rangeParts[0] != "" {
				start, _ = strconv.ParseInt(rangeParts[0], 10, 64)
			}
			if len(rangeParts) > 1 && rangeParts[1] != "" {
				end, _ = strconv.ParseInt(rangeParts[1], 10, 64)
			} else {
				end = fileSize - 1
			}

			if start < 0 || start >= fileSize || end < start || end >= fileSize {
				httpx.WriteJson(w, http.StatusRequestedRangeNotSatisfiable, map[string]interface{}{
					"code":    416,
					"message": "请求范围无效",
					"data":    nil,
				})
				return
			}

			contentLength := end - start + 1
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
			w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
			w.WriteHeader(http.StatusPartialContent)

			file.Seek(start, io.SeekStart)
			io.CopyN(w, file, contentLength)

			logx.Infof("流媒体播放(Range): userID=%d, contentID=%d, range=%s", bearerCtx.UserID, contentID, rangeHeader)
		} else {
			w.WriteHeader(http.StatusOK)
			io.Copy(w, file)

			logx.Infof("流媒体播放: userID=%d, contentID=%d, title=%s", bearerCtx.UserID, contentID, content.Title)
		}

		go recordPlayEvent(svcCtx, bearerCtx.UserID, contentID)
	}
}

func getLocalFilePath(svcCtx *svc.ServiceContext, audioURL string) string {
	if strings.HasPrefix(audioURL, "http://") || strings.HasPrefix(audioURL, "https://") {
		cdnBase := strings.TrimRight(svcCtx.Config.Storage.CdnBaseUrl, "/")
		if cdnBase != "" && strings.HasPrefix(audioURL, cdnBase) {
			objectKey := strings.TrimPrefix(audioURL, cdnBase)
			objectKey = strings.TrimLeft(objectKey, "/")
			return filepath.Join(svcCtx.Config.Local.Root, filepath.FromSlash(objectKey))
		}
		return ""
	}

	if strings.HasPrefix(audioURL, "/") {
		return filepath.Join(svcCtx.Config.Local.Root, filepath.FromSlash(strings.TrimLeft(audioURL, "/")))
	}

	return filepath.Join(svcCtx.Config.Local.Root, filepath.FromSlash(audioURL))
}

func recordPlayEvent(svcCtx *svc.ServiceContext, userID, contentID int64) {
	record := map[string]interface{}{
		"user_id":    userID,
		"content_id": contentID,
		"play_time":  time.Now(),
		"created_at": time.Now(),
	}
	svcCtx.DB.Table("user_play_records").Create(record)
}

func getFileExtFromURL(url string) string {
	idx := strings.LastIndex(url, ".")
	if idx == -1 {
		return ".mp3"
	}
	ext := url[idx:]
	qIdx := strings.Index(ext, "?")
	if qIdx != -1 {
		ext = ext[:qIdx]
	}
	return ext
}

func getContentTypeFromExt(audioURL string) string {
	ext := strings.ToLower(getFileExtFromURL(audioURL))
	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".flac":
		return "audio/flac"
	case ".aac":
		return "audio/aac"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	default:
		return "audio/mpeg"
	}
}

// contentDownloadHandler 歌曲下载处理器
// GET /api/v1/content/download/:id
// 必须登录，返回"下载中"状态，记录到用户下载列表
func contentDownloadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 GET",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 4 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "内容 ID 不能为空",
				"data":    nil,
			})
			return
		}

		contentID, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
		if err != nil || contentID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "内容 ID 格式错误",
				"data":    nil,
			})
			return
		}

		var content struct {
			ID        int64
			Title     string
			AudioURL  string
			VipLevel  int16
			Status    int16
			IsDeleted int16
		}
		err = svcCtx.DB.Table("content").
			Select("id, title, audio_url, vip_level, status, is_deleted").
			Where("id = ? AND status = 1 AND is_deleted = 0", contentID).
			First(&content).Error
		if err != nil {
			logx.Errorf("查询内容失败: %v", err)
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "内容不存在或已下架",
				"data":    nil,
			})
			return
		}

		userVipLevel := int16(0)
		var vipLevel int16
		err = svcCtx.DB.Table("user_vip").
			Select("vip_level").
			Where("user_id = ?", bearerCtx.UserID).
			First(&vipLevel).Error
		if err == nil {
			userVipLevel = vipLevel
		}

		if content.VipLevel > userVipLevel {
			httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
				"code":    403,
				"message": "该内容为VIP专属，请先升级会员",
				"data":    nil,
			})
			return
		}

		if content.AudioURL == "" {
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "音频文件不存在",
				"data":    nil,
			})
			return
		}

		localPath := getLocalFilePath(svcCtx, content.AudioURL)
		fileSize := int64(0)
		if localPath != "" {
			if fileInfo, err := os.Stat(localPath); err == nil {
				fileSize = fileInfo.Size()
			}
		}

		go recordDownloadEvent(svcCtx, bearerCtx.UserID, contentID, content.Title, content.AudioURL, fileSize)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "下载中",
			"data": map[string]interface{}{
				"content_id":    contentID,
				"title":         content.Title,
				"status":        "downloading",
				"stream_url":    fmt.Sprintf("/api/v1/content/stream/%d", contentID),
				"file_size":     fileSize,
				"download_time": time.Now().Format("2006-01-02 15:04:05"),
			},
		})
	}
}

// contentPlayStartHandler 播放开始处理器
// @Summary      播放开始
// @Description  记录播放开始并返回播放地址，需要登录
// @Tags         播放管理
// @Accept       json
// @Produce      json
// @Param        body  body      types.PlayStartReq  true  "播放请求"
// @Success      200  {object}  map[string]interface{}  "成功"
// @Failure      401  {object}  map[string]interface{}  "未登录"
// @Router       /play/start [post]
// @Security     BearerAuth
func contentPlayStartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 POST",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		var req types.PlayStartReq
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "读取请求体失败",
					"data":    nil,
				})
				return
			}
			json.Unmarshal(body, &req)
		} else {
			r.ParseForm()
			contentIDStr := r.FormValue("content_id")
			req.DeviceInfo = r.FormValue("device_info")
			if id, err := strconv.ParseInt(contentIDStr, 10, 64); err == nil {
				req.ContentID = id
			}
		}

		if req.ContentID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌曲 ID 不能为空",
				"data":    nil,
			})
			return
		}

		var song struct {
			ID       int64
			Title    string
			AudioURL string
			Status   int16
		}
		err := svcCtx.DB.Table("content").
			Select("id, title, audio_url, status").
			Where("id = ? AND status = 1", req.ContentID).
			First(&song).Error

		if err != nil {
			logx.Errorf("查询歌曲失败: %v", err)
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "歌曲不存在或已下架",
				"data":    nil,
			})
			return
		}

		clientIP := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
				clientIP = strings.TrimSpace(parts[0])
			}
		}

		now := time.Now()
		result := svcCtx.DB.Table("play_history").Create(map[string]interface{}{
			"user_id":     bearerCtx.UserID,
			"content_id":  req.ContentID,
			"status":      0,
			"progress":    0,
			"duration":    0,
			"play_url":    song.AudioURL,
			"client_ip":   clientIP,
			"device_info": req.DeviceInfo,
			"started_at":  now,
			"updated_at":  now,
		})

		if result.Error != nil {
			logx.Errorf("创建播放记录失败: %v", result.Error)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "创建播放记录失败",
				"data":    nil,
			})
			return
		}

		var playID int64
		svcCtx.DB.Table("play_history").Select("id").Where("user_id = ? AND content_id = ? AND started_at = ?", bearerCtx.UserID, req.ContentID, now).Scan(&playID)

		svcCtx.DB.Table("content").Where("id = ?", req.ContentID).UpdateColumns(map[string]interface{}{
			"play_count":     gorm.Expr("COALESCE(play_count, 0) + 1"),
			"last_played_at": now,
		})

		logx.Infof("播放开始: userID=%d, contentID=%d, title=%s, playID=%d, IP=%s",
			bearerCtx.UserID, req.ContentID, song.Title, playID, clientIP)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"play_id":    playID,
				"play_url":   song.AudioURL,
				"started_at": now.Format("2006-01-02 15:04:05"),
			},
		})
	}
}

// contentPlayProgressHandler 播放进度处理器
// POST /api/v1/content/play/progress
// 必须登录，更新播放进度（用于断点续播）
func contentPlayProgressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 POST",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		var req types.PlayProgressReq
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "读取请求体失败",
					"data":    nil,
				})
				return
			}
			json.Unmarshal(body, &req)
		} else {
			r.ParseForm()
			playIDStr := r.FormValue("play_id")
			progressStr := r.FormValue("progress")
			durationStr := r.FormValue("duration")
			if id, err := strconv.ParseInt(playIDStr, 10, 64); err == nil {
				req.PlayID = id
			}
			if p, err := strconv.Atoi(progressStr); err == nil {
				req.Progress = p
			}
			if d, err := strconv.Atoi(durationStr); err == nil {
				req.Duration = d
			}
		}

		if req.PlayID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "播放记录 ID 不能为空",
				"data":    nil,
			})
			return
		}

		var record struct {
			ID     int64
			UserID int64
			Status int16
		}
		err := svcCtx.DB.Table("play_history").
			Select("id, user_id, status").
			Where("id = ? AND user_id = ?", req.PlayID, bearerCtx.UserID).
			First(&record).Error

		if err != nil || record.ID <= 0 {
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "播放记录不存在",
				"data":    nil,
			})
			return
		}

		if record.Status == 2 {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{
					"success": false,
					"message": "播放已完成，无法更新进度",
				},
			})
			return
		}

		now := time.Now()
		updateData := map[string]interface{}{
			"progress":   req.Progress,
			"updated_at": now,
		}
		if req.Duration > 0 {
			updateData["duration"] = req.Duration
		}

		svcCtx.DB.Table("play_history").Where("id = ?", req.PlayID).UpdateColumns(updateData)

		logx.Infof("更新播放进度: playID=%d, progress=%ds, duration=%ds", req.PlayID, req.Progress, req.Duration)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"success":  true,
				"message":  "进度更新成功",
				"progress": req.Progress,
			},
		})
	}
}

// contentPlayCompleteHandler 播放完成处理器
// POST /api/v1/content/play/complete
// 必须登录，标记播放完成并更新统计
func contentPlayCompleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 POST",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		var req types.PlayCompleteReq
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "读取请求体失败",
					"data":    nil,
				})
				return
			}
			json.Unmarshal(body, &req)
		} else {
			r.ParseForm()
			playIDStr := r.FormValue("play_id")
			durationStr := r.FormValue("duration")
			if id, err := strconv.ParseInt(playIDStr, 10, 64); err == nil {
				req.PlayID = id
			}
			if d, err := strconv.Atoi(durationStr); err == nil {
				req.Duration = d
			}
		}

		if req.PlayID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "播放记录 ID 不能为空",
				"data":    nil,
			})
			return
		}

		var record struct {
			ID        int64
			UserID    int64
			ContentID int64
			Status    int16
			Duration  int
		}
		err := svcCtx.DB.Table("play_history").
			Select("id, user_id, content_id, status, duration").
			Where("id = ? AND user_id = ?", req.PlayID, bearerCtx.UserID).
			First(&record).Error

		if err != nil || record.ID <= 0 {
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "播放记录不存在",
				"data":    nil,
			})
			return
		}

		if record.Status == 2 {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{
					"success": false,
					"message": "该播放已完成，无需重复提交",
				},
			})
			return
		}

		now := time.Now()
		finalDuration := req.Duration
		if finalDuration <= 0 {
			finalDuration = record.Duration
		}

		svcCtx.DB.Table("play_history").Where("id = ?", req.PlayID).UpdateColumns(map[string]interface{}{
			"status":       2,
			"duration":     finalDuration,
			"completed_at": now,
			"updated_at":   now,
		})

		svcCtx.DB.Table("content").Where("id = ?", record.ContentID).UpdateColumn("today_play_count", gorm.Expr("COALESCE(today_play_count, 0) + 1"))

		logx.Infof("播放完成: userID=%d, playID=%d, contentID=%d, duration=%ds",
			bearerCtx.UserID, req.PlayID, record.ContentID, finalDuration)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"success":   true,
				"message":   "播放完成",
				"duration":  finalDuration,
				"played_at": now.Format("2006-01-02 15:04:05"),
			},
		})
	}
}

// contentPlayHistoryListHandler 播放记录列表处理器
// GET /api/v1/content/play/history
// 必须登录，查看用户播放历史记录
func contentPlayHistoryListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 GET",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		pageStr := r.URL.Query().Get("page")
		sizeStr := r.URL.Query().Get("page_size")
		statusStr := r.URL.Query().Get("status")

		page := int32(1)
		pageSize := int32(20)

		if pageStr != "" {
			if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
				page = int32(p)
			}
		}
		if sizeStr != "" {
			if s, err := strconv.ParseInt(sizeStr, 10, 32); err == nil && s > 0 && s <= 50 {
				pageSize = int32(s)
			}
		}

		query := svcCtx.DB.Table("play_history").Where("user_id = ?", bearerCtx.UserID)

		if statusStr != "" {
			if s, err := strconv.Atoi(statusStr); err == nil && (s == 0 || s == 2) {
				query = query.Where("status = ?", s)
			}
		}

		var total int64
		query.Count(&total)

		offset := (page - 1) * pageSize

		sqlQuery := `
			SELECT
				ph.id AS play_id,
				ph.content_id,
				COALESCE(c.title, '') AS title,
				COALESCE(c.artist, '') AS artist,
				COALESCE(c.cover_url, '') AS cover_url,
				ph.status,
				ph.progress,
				ph.duration,
				COALESCE(ph.play_url, '') AS play_url,
				COALESCE(ph.client_ip, '') AS client_ip,
				COALESCE(ph.device_info, '') AS device_info,
				ph.started_at,
				ph.completed_at,
				COALESCE(c.vip_level, 0) AS vip_level
			FROM play_history ph
			LEFT JOIN content c ON ph.content_id = c.id
			WHERE ph.user_id = ?
		`
		args := []interface{}{bearerCtx.UserID}

		if statusStr != "" {
			if s, err := strconv.Atoi(statusStr); err == nil && (s == 0 || s == 2) {
				sqlQuery += " AND ph.status = ?"
				args = append(args, s)
			}
		}

		sqlQuery += " ORDER BY ph.started_at DESC LIMIT ? OFFSET ?"
		args = append(args, pageSize, offset)

		rows, err := svcCtx.DB.Raw(sqlQuery, args...).Rows()

		if err != nil {
			logx.Errorf("查询播放记录失败: %v", err)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "查询播放记录失败",
				"data":    nil,
			})
			return
		}
		defer rows.Close()

		var list []map[string]interface{}
		for rows.Next() {
			var item struct {
				PlayID      int64
				ContentID   int64
				Title       string
				Artist      string
				CoverURL    string
				Status      int16
				Progress    int
				Duration    int
				PlayURL     string
				ClientIP    string
				DeviceInfo  string
				StartedAt   time.Time
				CompletedAt *time.Time
				VipLevel    int16
			}
			if err := rows.Scan(
				&item.PlayID, &item.ContentID, &item.Title, &item.Artist,
				&item.CoverURL, &item.Status, &item.Progress, &item.Duration,
				&item.PlayURL, &item.ClientIP, &item.DeviceInfo,
				&item.StartedAt, &item.CompletedAt, &item.VipLevel,
			); err != nil {
				continue
			}

			completedAt := ""
			if item.CompletedAt != nil {
				completedAt = item.CompletedAt.Format("2006-01-02 15:04:05")
			}

			list = append(list, map[string]interface{}{
				"play_id":      item.PlayID,
				"content_id":   item.ContentID,
				"title":        item.Title,
				"artist":       item.Artist,
				"cover_url":    item.CoverURL,
				"status":       item.Status,
				"progress":     item.Progress,
				"duration":     item.Duration,
				"play_url":     item.PlayURL,
				"client_ip":    item.ClientIP,
				"device_info":  item.DeviceInfo,
				"started_at":   item.StartedAt.Format("2006-01-02 15:04:05"),
				"completed_at": completedAt,
				"vip_level":    item.VipLevel,
			})
		}

		logx.Infof("查询播放记录列表: userID=%d, page=%d, size=%d, total=%d",
			bearerCtx.UserID, page, pageSize, total)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"total":     total,
				"list":      list,
				"page":      page,
				"page_size": pageSize,
			},
		})
	}
}

// contentSearchHandler 搜索歌曲处理器
// GET /api/v1/content/search
// 支持按歌名、歌词、歌手搜索
func contentSearchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 GET",
				"data":    nil,
			})
			return
		}

		keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
		if keyword == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "搜索关键词不能为空",
				"data":    nil,
			})
			return
		}

		searchType := r.URL.Query().Get("type")
		if searchType == "" {
			searchType = "all"
		}

		pageStr := r.URL.Query().Get("page")
		sizeStr := r.URL.Query().Get("page_size")

		page := int32(1)
		pageSize := int32(20)

		if pageStr != "" {
			if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
				page = int32(p)
			}
		}
		if sizeStr != "" {
			if s, err := strconv.ParseInt(sizeStr, 10, 32); err == nil && s > 0 && s <= 50 {
				pageSize = int32(s)
			}
		}

		var searchSQL string
		var countSQL string
		var args []interface{}

		switch searchType {
		case "title":
			countSQL = `SELECT COUNT(DISTINCT c.id) FROM content c WHERE c.status = 1 AND c.is_deleted = 0 AND c.title ILIKE $1`
			searchSQL = `
				SELECT DISTINCT
					c.id, c.title, c.artist, c.cover_url, c.duration_sec, c.vip_level,
					ARRAY['title']::text[] AS match_fields,
					1.0 AS score
				FROM content c
				WHERE c.status = 1 AND c.is_deleted = 0 AND c.title ILIKE $1
				ORDER BY score DESC, c.title
				LIMIT $2 OFFSET $3
			`
			args = []interface{}{"%" + keyword + "%", pageSize, (page - 1) * pageSize}
		case "artist":
			countSQL = `SELECT COUNT(DISTINCT c.id) FROM content c WHERE c.status = 1 AND c.is_deleted = 0 AND c.artist ILIKE $1`
			searchSQL = `
				SELECT DISTINCT
					c.id, c.title, c.artist, c.cover_url, c.duration_sec, c.vip_level,
					ARRAY['artist']::text[] AS match_fields,
					1.0 AS score
				FROM content c
				WHERE c.status = 1 AND c.is_deleted = 0 AND c.artist ILIKE $1
				ORDER BY score DESC, c.artist
				LIMIT $2 OFFSET $3
			`
			args = []interface{}{"%" + keyword + "%", pageSize, (page - 1) * pageSize}
		case "lyrics":
			countSQL = `SELECT COUNT(DISTINCT c.id) FROM content c WHERE c.status = 1 AND c.is_deleted = 0 AND c.lyrics ILIKE $1`
			searchSQL = `
				SELECT DISTINCT
					c.id, c.title, c.artist, c.cover_url, c.duration_sec, c.vip_level,
					ARRAY['lyrics']::text[] AS match_fields,
					1.0 AS score
				FROM content c
				WHERE c.status = 1 AND c.is_deleted = 0 AND c.lyrics ILIKE $1
				ORDER BY score DESC, c.title
				LIMIT $2 OFFSET $3
			`
			args = []interface{}{"%" + keyword + "%", pageSize, (page - 1) * pageSize}
		default:
			countSQL = `
				SELECT COUNT(DISTINCT c.id) FROM content c
				WHERE c.status = 1 AND c.is_deleted = 0
				AND (c.title ILIKE $1 OR c.artist ILIKE $1 OR c.lyrics ILIKE $1)
			`
			searchSQL = `
				SELECT
					c.id, c.title, c.artist, c.cover_url, c.duration_sec, c.vip_level,
					CASE
						WHEN c.title ILIKE $1 AND c.artist ILIKE $1 AND c.lyrics ILIKE $1 THEN ARRAY['title', 'artist', 'lyrics']::text[]
						WHEN c.title ILIKE $1 AND c.artist ILIKE $1 THEN ARRAY['title', 'artist']::text[]
						WHEN c.title ILIKE $1 AND c.lyrics ILIKE $1 THEN ARRAY['title', 'lyrics']::text[]
						WHEN c.artist ILIKE $1 AND c.lyrics ILIKE $1 THEN ARRAY['artist', 'lyrics']::text[]
						WHEN c.title ILIKE $1 THEN ARRAY['title']::text[]
						WHEN c.artist ILIKE $1 THEN ARRAY['artist']::text[]
						WHEN c.lyrics ILIKE $1 THEN ARRAY['lyrics']::text[]
					END AS match_fields,
					CASE
						WHEN c.title ILIKE $1 AND c.artist ILIKE $1 AND c.lyrics ILIKE $1 THEN 3.0
						WHEN c.title ILIKE $1 AND c.artist ILIKE $1 THEN 2.5
						WHEN c.title ILIKE $1 AND c.lyrics ILIKE $1 THEN 2.5
						WHEN c.artist ILIKE $1 AND c.lyrics ILIKE $1 THEN 2.0
						WHEN c.title ILIKE $1 THEN 2.0
						WHEN c.artist ILIKE $1 THEN 1.5
						WHEN c.lyrics ILIKE $1 THEN 1.0
					END AS score
				FROM content c
				WHERE c.status = 1 AND c.is_deleted = 0
				AND (c.title ILIKE $1 OR c.artist ILIKE $1 OR c.lyrics ILIKE $1)
				ORDER BY score DESC, c.title
				LIMIT $2 OFFSET $3
			`
			args = []interface{}{"%" + keyword + "%", pageSize, (page - 1) * pageSize}
		}

		var total int64
		svcCtx.DB.Raw(countSQL, args[0]).Scan(&total)

		rows, err := svcCtx.DB.Raw(searchSQL, args...).Rows()
		if err != nil {
			logx.Errorf("搜索歌曲失败: %v", err)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "搜索失败",
				"data":    nil,
			})
			return
		}
		defer rows.Close()

		var list []map[string]interface{}
		for rows.Next() {
			var item struct {
				ID          int64
				Title       string
				Artist      string
				CoverURL    string
				Duration    int
				VipLevel    int16
				MatchFields pq.StringArray
				Score       float64
			}
			if err := rows.Scan(&item.ID, &item.Title, &item.Artist, &item.CoverURL, &item.Duration, &item.VipLevel, &item.MatchFields, &item.Score); err != nil {
				continue
			}

			matchFields := make([]string, len(item.MatchFields))
			for i, f := range item.MatchFields {
				matchFields[i] = f
			}

			list = append(list, map[string]interface{}{
				"id":           item.ID,
				"title":        item.Title,
				"artist":       item.Artist,
				"cover_url":    item.CoverURL,
				"duration":     item.Duration,
				"vip_level":    item.VipLevel,
				"match_fields": matchFields,
				"score":        item.Score,
			})
		}

		logx.Infof("搜索歌曲: keyword=%s, type=%s, total=%d, page=%d", keyword, searchType, total, page)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"total":     total,
				"list":      list,
				"page":      page,
				"page_size": pageSize,
			},
		})
	}
}

// downloadCompleteHandler 确认下载完成处理器
// POST /api/v1/content/download/complete
// 必须登录，更新下载记录状态为"已下载"
func downloadCompleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 POST",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		var req struct {
			ContentID int64  `json:"content_id"`
			FileSize  int64  `json:"file_size"`
			LocalPath string `json:"local_path"`
		}
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "请求参数错误: " + err.Error(),
				"data":    nil,
			})
			return
		}

		if req.ContentID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "内容 ID 不能为空",
				"data":    nil,
			})
			return
		}

		var record struct {
			ID        int64
			UserID    int64
			ContentID int64
			Status    int16
		}
		err := svcCtx.DB.Table("user_downloads").
			Select("id, user_id, content_id, status").
			Where("user_id = ? AND content_id = ?", bearerCtx.UserID, req.ContentID).
			First(&record).Error
		if err != nil {
			logx.Errorf("查询下载记录失败: %v", err)
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "下载记录不存在",
				"data":    nil,
			})
			return
		}

		now := time.Now()
		updates := map[string]interface{}{
			"status":        3,
			"download_time": now,
			"updated_at":    now,
		}
		if req.FileSize > 0 {
			updates["file_size"] = req.FileSize
		}
		if req.LocalPath != "" {
			updates["file_url"] = req.LocalPath
		}

		result := svcCtx.DB.Table("user_downloads").
			Where("id = ?", record.ID).
			Updates(updates)
		if result.Error != nil {
			logx.Errorf("更新下载记录失败: %v", result.Error)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "更新下载记录失败",
				"data":    nil,
			})
			return
		}

		svcCtx.DB.Table("content_download_stats").
			Where("content_id = ?", req.ContentID).
			Updates(map[string]interface{}{
				"total_downloads":    gorm.Expr("COALESCE(total_downloads, 0) + 0"),
				"last_download_time": now,
				"updated_at":         now,
			})

		logx.Infof("确认下载完成: userID=%d, contentID=%d, recordID=%d, fileSize=%d",
			bearerCtx.UserID, req.ContentID, record.ID, req.FileSize)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"success":      true,
				"message":      "下载完成",
				"record_id":    record.ID,
				"status":       "completed",
				"completed_at": now.Format("2006-01-02 15:04:05"),
			},
		})
	}
}

func recordDownloadEvent(svcCtx *svc.ServiceContext, userID, contentID int64, title, audioURL string, fileSize int64) {
	svcCtx.DB.Table("content_download_stats").
		Where("content_id = ?", contentID).
		Updates(map[string]interface{}{
			"total_downloads":    gorm.Expr("COALESCE(total_downloads, 0) + 1"),
			"today_downloads":    gorm.Expr("COALESCE(today_downloads, 0) + 1"),
			"week_downloads":     gorm.Expr("COALESCE(week_downloads, 0) + 1"),
			"last_download_time": time.Now(),
			"updated_at":         time.Now(),
		})

	var existingCount int64
	svcCtx.DB.Table("user_downloads").
		Where("user_id = ? AND content_id = ?", userID, contentID).
		Count(&existingCount)

	if existingCount == 0 {
		svcCtx.DB.Table("user_downloads").Create(map[string]interface{}{
			"user_id":       userID,
			"content_id":    contentID,
			"content_title": title,
			"file_url":      audioURL,
			"download_time": time.Now(),
			"status":        2,
			"file_size":     fileSize,
			"created_at":    time.Now(),
			"sync_status":   0,
		})
	} else {
		svcCtx.DB.Table("user_downloads").
			Where("user_id = ? AND content_id = ?", userID, contentID).
			Updates(map[string]interface{}{
				"download_time": time.Now(),
				"status":        2,
				"file_size":     fileSize,
			})
	}
}

// contentLikeHandler 点赞/取消点赞处理器
// POST /api/v1/content/:id/like
// 必须登录，切换点赞状态（已点赞则取消，未点赞则点赞）
func contentLikeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 POST",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 4 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "内容 ID 不能为空",
				"data":    nil,
			})
			return
		}

		contentID, err := strconv.ParseInt(parts[len(parts)-2], 10, 64)
		if err != nil || contentID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "内容 ID 格式错误",
				"data":    nil,
			})
			return
		}

		var content struct {
			ID        int64
			Title     string
			Status    int16
			IsDeleted int16
		}
		err = svcCtx.DB.Table("content").
			Select("id, title, status, is_deleted").
			Where("id = ? AND status = 1 AND is_deleted = 0", contentID).
			First(&content).Error
		if err != nil {
			logx.Errorf("查询内容失败: %v", err)
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "歌曲不存在或已下架",
				"data":    nil,
			})
			return
		}

		var likeRecord struct {
			ID int64
		}
		likeErr := svcCtx.DB.Table("user_likes").
			Select("id").
			Where("user_id = ? AND content_id = ?", bearerCtx.UserID, contentID).
			First(&likeRecord).Error

		isLiked := likeErr == nil && likeRecord.ID > 0

		if isLiked {
			deleteResult := svcCtx.DB.Table("user_likes").
				Where("user_id = ? AND content_id = ?", bearerCtx.UserID, contentID).
				Delete(&struct{}{})
			if deleteResult.Error != nil {
				logx.Errorf("取消点赞失败: %v", deleteResult.Error)
				httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
					"code":    500,
					"message": "取消点赞失败",
					"data":    nil,
				})
				return
			}

			svcCtx.DB.Table("content").
				Where("id = ?", contentID).
				Update("like_count", gorm.Expr("GREATEST(like_count - 1, 0)"))

			logx.Infof("取消点赞: userID=%d, contentID=%d, title=%s", bearerCtx.UserID, contentID, content.Title)

			var likeCount int64
			svcCtx.DB.Table("content").Select("like_count").Where("id = ?", contentID).Scan(&likeCount)

			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{
					"success":    true,
					"message":    "取消点赞成功",
					"liked":      false,
					"like_count": likeCount,
				},
			})
		} else {
			createResult := svcCtx.DB.Table("user_likes").Create(map[string]interface{}{
				"user_id":    bearerCtx.UserID,
				"content_id": contentID,
				"created_at": time.Now(),
			})
			if createResult.Error != nil {
				logx.Errorf("点赞失败: %v", createResult.Error)
				httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
					"code":    500,
					"message": "点赞失败",
					"data":    nil,
				})
				return
			}

			svcCtx.DB.Table("content").
				Where("id = ?", contentID).
				Update("like_count", gorm.Expr("COALESCE(like_count, 0) + 1"))

			logx.Infof("点赞成功: userID=%d, contentID=%d, title=%s", bearerCtx.UserID, contentID, content.Title)

			var likeCount int64
			svcCtx.DB.Table("content").Select("like_count").Where("id = ?", contentID).Scan(&likeCount)

			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{
					"success":    true,
					"message":    "点赞成功",
					"liked":      true,
					"like_count": likeCount,
				},
			})
		}
	}
}

// contentLikeListHandler 点赞列表处理器
// GET /api/v1/content/likes
// 必须登录，查看用户点赞的歌曲列表
func contentLikeListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 GET",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		pageStr := r.URL.Query().Get("page")
		sizeStr := r.URL.Query().Get("page_size")
		page := int32(1)
		pageSize := int32(20)

		if pageStr != "" {
			if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
				page = int32(p)
			}
		}
		if sizeStr != "" {
			if s, err := strconv.ParseInt(sizeStr, 10, 32); err == nil && s > 0 && s <= 50 {
				pageSize = int32(s)
			}
		}

		var total int64
		svcCtx.DB.Table("user_likes").
			Where("user_id = ?", bearerCtx.UserID).
			Count(&total)

		offset := (page - 1) * pageSize

		rows, err := svcCtx.DB.Raw(`
			SELECT
				c.id AS content_id,
				COALESCE(c.title, '') AS title,
				COALESCE(c.artist, '') AS artist,
				COALESCE(c.cover_url, '') AS cover_url,
				ul.created_at AS liked_at,
				COALESCE(c.vip_level, 0) AS vip_level,
				CASE WHEN c.vip_level > 0 THEN true ELSE false END AS is_vip_content
			FROM user_likes ul
			INNER JOIN content c ON ul.content_id = c.id
			WHERE ul.user_id = ? AND c.status = 1 AND c.is_deleted = 0
			ORDER BY ul.created_at DESC
			LIMIT ? OFFSET ?
		`, bearerCtx.UserID, pageSize, offset).Rows()
		if err != nil {
			logx.Errorf("查询点赞列表失败: %v", err)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "查询点赞列表失败",
				"data":    nil,
			})
			return
		}
		defer rows.Close()

		var list []map[string]interface{}
		for rows.Next() {
			var item struct {
				ContentID    int64
				Title        string
				Artist       string
				CoverURL     string
				LikedAt      time.Time
				VipLevel     int16
				IsVipContent bool
			}
			if err := rows.Scan(
				&item.ContentID, &item.Title, &item.Artist,
				&item.CoverURL, &item.LikedAt,
				&item.VipLevel, &item.IsVipContent,
			); err != nil {
				continue
			}
			list = append(list, map[string]interface{}{
				"content_id":     item.ContentID,
				"title":          item.Title,
				"artist":         item.Artist,
				"cover_url":      item.CoverURL,
				"liked_at":       item.LikedAt.Format("2006-01-02 15:04:05"),
				"vip_level":      item.VipLevel,
				"is_vip_content": item.IsVipContent,
			})
		}

		logx.Infof("查询点赞列表: userID=%d, page=%d, size=%d, total=%d",
			bearerCtx.UserID, page, pageSize, total)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"total":     total,
				"list":      list,
				"page":      page,
				"page_size": pageSize,
			},
		})
	}
}

// contentPlaylistCreateHandler 创建歌单处理器
// POST /api/v1/content/playlists
// 必须登录，创建用户歌单
func contentPlaylistCreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 POST",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		var req types.PlaylistCreateReq
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "读取请求体失败",
					"data":    nil,
				})
				return
			}
			json.Unmarshal(body, &req)
		} else {
			r.ParseForm()
			req.Name = r.FormValue("name")
			req.Description = r.FormValue("description")
			req.CoverURL = r.FormValue("cover_url")
		}

		name := strings.TrimSpace(req.Name)
		if name == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌单名称不能为空",
				"data":    nil,
			})
			return
		}

		if len(name) < 1 || len(name) > 100 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌单名称长度必须在1-100个字符之间",
				"data":    nil,
			})
			return
		}

		var playlistCount int64
		svcCtx.DB.Table("playlists").
			Where("user_id = ? AND deleted_at IS NULL", bearerCtx.UserID).
			Count(&playlistCount)

		const maxPlaylists = 100
		if playlistCount >= maxPlaylists {
			httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
				"code":    403,
				"message": fmt.Sprintf("已达到最大歌单数量限制（%d个）", maxPlaylists),
				"data":    nil,
			})
			return
		}

		description := strings.TrimSpace(req.Description)
		coverURL := strings.TrimSpace(req.CoverURL)
		if coverURL == "" {
			coverURL = "/static/default-playlist-cover.png"
		}

		now := time.Now()
		playlist := struct {
			ID          int64
			Name        string
			Description string
			CoverURL    string
			SongCount   int
			CreatedAt   time.Time
		}{
			Name:        name,
			Description: description,
			CoverURL:    coverURL,
			SongCount:   0,
			CreatedAt:   now,
		}

		result := svcCtx.DB.Table("playlists").Create(map[string]interface{}{
			"user_id":     bearerCtx.UserID,
			"name":        name,
			"description": description,
			"cover_url":   coverURL,
			"song_count":  0,
			"is_public":   1,
			"status":      1,
			"created_at":  now,
			"updated_at":  now,
		})

		if result.Error != nil {
			logx.Errorf("创建歌单失败: %v", result.Error)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "创建歌单失败",
				"data":    nil,
			})
			return
		}

		svcCtx.DB.Table("playlists").
			Select("id, name, description, cover_url, song_count, created_at").
			Where("id = ?", result.RowsAffected).
			First(&playlist)

		logx.Infof("创建歌单成功: userID=%d, playlistID=%d, name=%s",
			bearerCtx.UserID, playlist.ID, playlist.Name)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"id":          playlist.ID,
				"name":        playlist.Name,
				"description": playlist.Description,
				"cover_url":   playlist.CoverURL,
				"song_count":  playlist.SongCount,
				"created_at":  playlist.CreatedAt.Format("2006-01-02 15:04:05"),
			},
		})
	}
}

// contentPlaylistAddSongHandler 添加歌曲到歌单处理器
// POST /api/v1/content/playlists/:id/songs
// 必须登录，添加歌曲到指定歌单
func contentPlaylistAddSongHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 POST",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 5 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌单 ID 不能为空",
				"data":    nil,
			})
			return
		}

		playlistID, err := strconv.ParseInt(parts[len(parts)-2], 10, 64)
		if err != nil || playlistID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌单 ID 格式错误",
				"data":    nil,
			})
			return
		}

		var req types.PlaylistAddSongReq
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "读取请求体失败",
					"data":    nil,
				})
				return
			}
			json.Unmarshal(body, &req)
		} else {
			r.ParseForm()
			contentIDStr := r.FormValue("content_id")
			if id, err := strconv.ParseInt(contentIDStr, 10, 64); err == nil {
				req.ContentID = id
			}
		}

		if req.ContentID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌曲 ID 不能为空",
				"data":    nil,
			})
			return
		}

		var playlist struct {
			ID        int64
			UserID    int64
			Name      string
			SongCount int
			IsPublic  int16
			Status    int16
		}
		err = svcCtx.DB.Table("playlists").
			Select("id, user_id, name, song_count, is_public, status").
			Where("id = ? AND status = 1 AND deleted_at IS NULL", playlistID).
			First(&playlist).Error
		if err != nil {
			logx.Errorf("查询歌单失败: %v", err)
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "歌单不存在",
				"data":    nil,
			})
			return
		}

		if playlist.UserID != bearerCtx.UserID && playlist.IsPublic == 0 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
				"code":    403,
				"message": "无权限操作此歌单",
				"data":    nil,
			})
			return
		}

		var song struct {
			ID     int64
			Title  string
			Status int16
		}
		songErr := svcCtx.DB.Table("content").
			Select("id, title, status").
			Where("id = ? AND status = 1", req.ContentID).
			First(&song).Error

		if songErr != nil {
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "歌曲不存在或已下架",
				"data":    nil,
			})
			return
		}

		var existingRecord struct {
			ID int64
		}
		existingErr := svcCtx.DB.Table("playlist_songs").
			Select("id").
			Where("playlist_id = ? AND content_id = ?", playlistID, req.ContentID).
			First(&existingRecord).Error

		if existingErr == nil && existingRecord.ID > 0 {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{
					"success":    false,
					"message":    "歌曲已在歌单中",
					"song_count": playlist.SongCount,
				},
			})
			return
		}

		var maxSortOrder int
		svcCtx.DB.Table("playlist_songs").
			Select("COALESCE(MAX(sort_order), -1)").
			Where("playlist_id = ?", playlistID).
			Scan(&maxSortOrder)

		now := time.Now()
		insertResult := svcCtx.DB.Table("playlist_songs").Create(map[string]interface{}{
			"playlist_id": playlistID,
			"content_id":  req.ContentID,
			"sort_order":  maxSortOrder + 1,
			"created_at":  now,
		})

		if insertResult.Error != nil {
			logx.Errorf("添加歌曲到歌单失败: %v", insertResult.Error)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "添加歌曲到歌单失败",
				"data":    nil,
			})
			return
		}

		svcCtx.DB.Table("playlists").
			Where("id = ?", playlistID).
			Update("song_count", gorm.Expr("COALESCE(song_count, 0) + 1"))

		svcCtx.DB.Table("playlists").
			Where("id = ?", playlistID).
			Update("updated_at", now)

		var newSongCount int
		svcCtx.DB.Table("playlists").Select("song_count").Where("id = ?", playlistID).Scan(&newSongCount)

		logx.Infof("添加歌曲到歌单成功: userID=%d, playlistID=%d, playlistName=%s, contentID=%d, songTitle=%s",
			bearerCtx.UserID, playlistID, playlist.Name, req.ContentID, song.Title)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"success":    true,
				"message":    "添加歌曲到歌单成功",
				"song_count": newSongCount,
			},
		})
	}
}

// contentPlaylistUpdateHandler 更新歌单处理器
// PUT /api/v1/content/playlists/:id
// 必须登录，只有歌单创建者才能修改
func contentPlaylistUpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut && r.Method != http.MethodPatch {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 PUT/PATCH",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		path := strings.Trim(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌单 ID 格式无效",
				"data":    nil,
			})
			return
		}
		playlistIDStr := parts[len(parts)-1]
		playlistID, err := strconv.ParseInt(playlistIDStr, 10, 64)
		if err != nil || playlistID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌单 ID 格式无效",
				"data":    nil,
			})
			return
		}

		var req types.PlaylistUpdateReq
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "读取请求体失败",
					"data":    nil,
				})
				return
			}
			json.Unmarshal(body, &req)
		} else {
			r.ParseForm()
			req.Name = r.FormValue("name")
			req.Description = r.FormValue("description")
			req.CoverURL = r.FormValue("cover_url")
			if v := r.FormValue("is_public"); v != "" {
				val, err := strconv.ParseInt(v, 10, 16)
				if err == nil {
					isPublic := int16(val)
					req.IsPublic = &isPublic
				}
			}
		}

		name := strings.TrimSpace(req.Name)
		description := strings.TrimSpace(req.Description)

		if name != "" {
			if len(name) < 1 || len(name) > 100 {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "歌单名称长度必须在1-100个字符之间",
					"data":    nil,
				})
				return
			}
		}

		if description != "" && len(description) > 500 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌单描述长度不能超过500个字符",
				"data":    nil,
			})
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
			Status      int16
			CreatedAt   time.Time
			UpdatedAt   time.Time
		}
		err = svcCtx.DB.Table("playlists").
			Select("id, user_id, name, description, cover_url, song_count, is_public, status, created_at, updated_at").
			Where("id = ? AND status = 1 AND deleted_at IS NULL", playlistID).
			First(&playlist).Error

		if err != nil {
			logx.Errorf("查询歌单失败: %v", err)
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "歌单不存在",
				"data":    nil,
			})
			return
		}

		if playlist.UserID != bearerCtx.UserID {
			httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
				"code":    403,
				"message": "无权限修改此歌单，只有创建者才能修改",
				"data":    nil,
			})
			return
		}

		hasUpdate := false
		updates := make(map[string]interface{})

		if name != "" && name != playlist.Name {
			updates["name"] = name
			hasUpdate = true
		}

		if description != "" && description != playlist.Description {
			updates["description"] = description
			hasUpdate = true
		}

		coverURL := strings.TrimSpace(req.CoverURL)
		if coverURL != "" && coverURL != playlist.CoverURL {
			oldCoverURL := playlist.CoverURL
			if oldCoverURL != "" && !strings.HasPrefix(oldCoverURL, "http") &&
				!strings.HasPrefix(oldCoverURL, "/static/default") {
				localPath := getLocalFilePath(svcCtx, oldCoverURL)
				if localPath != "" {
					os.Remove(localPath)
				}
			}
			updates["cover_url"] = coverURL
			hasUpdate = true
		}

		if req.IsPublic != nil && *req.IsPublic != playlist.IsPublic {
			if *req.IsPublic == 0 || *req.IsPublic == 1 {
				updates["is_public"] = *req.IsPublic
				hasUpdate = true
			} else {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "可见性设置无效，0-私有，1-公开",
					"data":    nil,
				})
				return
			}
		}

		if !hasUpdate {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 200,
				"data": types.PlaylistUpdateResp{
					ID:          playlist.ID,
					Name:        playlist.Name,
					Description: playlist.Description,
					CoverURL:    playlist.CoverURL,
					SongCount:   playlist.SongCount,
					IsPublic:    playlist.IsPublic,
					UpdatedAt:   playlist.UpdatedAt.Format("2006-01-02 15:04:05"),
				},
			})
			return
		}

		now := time.Now()
		updates["updated_at"] = now

		result := svcCtx.DB.Table("playlists").Where("id = ?", playlistID).Updates(updates)
		if result.Error != nil {
			logx.Errorf("更新歌单失败: %v", result.Error)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "更新歌单失败",
				"data":    nil,
			})
			return
		}

		var updatedPlaylist struct {
			Name        string
			Description string
			CoverURL    string
			IsPublic    int16
			UpdatedAt   time.Time
		}
		svcCtx.DB.Table("playlists").
			Select("name, description, cover_url, is_public, updated_at").
			Where("id = ?", playlistID).
			Scan(&updatedPlaylist)

		if updatedPlaylist.Name != "" {
			playlist.Name = updatedPlaylist.Name
		}
		if updatedPlaylist.Description != "" {
			playlist.Description = updatedPlaylist.Description
		}
		if updatedPlaylist.CoverURL != "" {
			playlist.CoverURL = updatedPlaylist.CoverURL
		}
		if updatedPlaylist.UpdatedAt.IsZero() == false {
			playlist.UpdatedAt = updatedPlaylist.UpdatedAt
		}
		if updatedPlaylist.IsPublic != 0 || req.IsPublic != nil {
			playlist.IsPublic = updatedPlaylist.IsPublic
		}

		logx.Infof("更新歌单成功: userID=%d, playlistID=%d, name=%s",
			bearerCtx.UserID, playlistID, playlist.Name)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": types.PlaylistUpdateResp{
				ID:          playlist.ID,
				Name:        playlist.Name,
				Description: playlist.Description,
				CoverURL:    playlist.CoverURL,
				SongCount:   playlist.SongCount,
				IsPublic:    playlist.IsPublic,
				UpdatedAt:   now.Format("2006-01-02 15:04:05"),
			},
		})
	}
}

// contentPlaylistDeleteHandler 删除歌单处理器
// DELETE /api/v1/content/playlists/:id
// 必须登录，只有歌单创建者才能删除
func contentPlaylistDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 DELETE",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		path := strings.Trim(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌单 ID 格式无效",
				"data":    nil,
			})
			return
		}
		playlistIDStr := parts[len(parts)-1]
		playlistID, err := strconv.ParseInt(playlistIDStr, 10, 64)
		if err != nil || playlistID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌单 ID 格式无效",
				"data":    nil,
			})
			return
		}

		var playlist struct {
			ID        int64
			UserID    int64
			Name      string
			SongCount int
			IsDefault int16
			Status    int16
			DeletedAt *time.Time
			CreatedAt time.Time
		}
		err = svcCtx.DB.Table("playlists").
			Select("id, user_id, name, song_count, is_default, status, deleted_at, created_at").
			Where("id = ?", playlistID).
			First(&playlist).Error

		if err != nil {
			logx.Errorf("查询歌单失败: %v", err)
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "歌单不存在",
				"data":    nil,
			})
			return
		}

		if playlist.DeletedAt != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌单已被删除",
				"data":    nil,
			})
			return
		}

		if playlist.UserID != bearerCtx.UserID {
			httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
				"code":    403,
				"message": "无权限删除此歌单，只有创建者才能删除",
				"data":    nil,
			})
			return
		}

		if playlist.IsDefault == 1 {
			httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
				"code":    403,
				"message": "默认歌单不能删除",
				"data":    nil,
			})
			return
		}

		var songCount int64
		svcCtx.DB.Table("playlist_songs").Where("playlist_id = ?", playlistID).Count(&songCount)

		logx.Infof("删除歌单关联歌曲: playlistID=%d, name=%s, songCount=%d",
			playlistID, playlist.Name, songCount)

		svcCtx.DB.Table("playlist_songs").Where("playlist_id = ?", playlistID).Delete(nil)

		now := time.Now()
		result := svcCtx.DB.Table("playlists").Where("id = ?", playlistID).Updates(map[string]interface{}{
			"status":     2,
			"deleted_at": now,
			"updated_at": now,
		})

		if result.Error != nil {
			logx.Errorf("删除歌单失败: %v", result.Error)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "删除歌单失败",
				"data":    nil,
			})
			return
		}

		logx.Infof("歌单删除成功: userID=%d, playlistID=%d, name=%s, deletedSongs=%d",
			bearerCtx.UserID, playlistID, playlist.Name, songCount)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": types.PlaylistDeleteResp{
				Success: true,
				Message: "歌单删除成功",
			},
		})
	}
}

// contentSubscribeHandler 订阅音频处理器
// @Summary      订阅音频
// @Description  用户订阅歌曲/歌手/专辑，需要登录
// @Tags         订阅管理
// @Accept       json
// @Produce      json
// @Param        id   path      int64  true  "音频ID"
// @Param        body  body      types.SubscribeReq  true  "订阅请求"
// @Success      200  {object}  map[string]interface{}  "成功"
// @Failure      401  {object}  map[string]interface{}  "未登录"
// @Failure      400  {object}  map[string]interface{}  "参数错误"
// @Failure      404  {object}  map[string]interface{}  "音频不存在"
// @Router       /{id}/subscribe [post]
// @Security     BearerAuth
func contentSubscribeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 POST",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		path := strings.Trim(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		var contentID int64
		for i := len(parts) - 1; i >= 0; i-- {
			if id, err := strconv.ParseInt(parts[i], 10, 64); err == nil && id > 0 {
				contentID = id
				break
			}
		}
		if contentID <= 0 {
			var req types.SubscribeReq
			contentType := r.Header.Get("Content-Type")
			if strings.Contains(contentType, "application/json") {
				body, err := io.ReadAll(r.Body)
				if err == nil {
					json.Unmarshal(body, &req)
				}
			} else {
				r.ParseForm()
				if v := r.FormValue("content_id"); v != "" {
					contentID, _ = strconv.ParseInt(v, 10, 64)
				}
			}
			if req.ContentID > 0 {
				contentID = req.ContentID
			}
		}

		if contentID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "音频 ID 不能为空",
				"data":    nil,
			})
			return
		}

		subscribeType := int16(1)
		var req types.SubscribeReq
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &req)
		} else {
			r.ParseForm()
			if v := r.FormValue("type"); v != "" {
				if t, err := strconv.ParseInt(v, 10, 16); err == nil {
					subscribeType = int16(t)
				}
			}
		}
		if req.Type > 0 {
			subscribeType = req.Type
		}

		if subscribeType < 1 || subscribeType > 3 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "订阅类型无效，1-歌曲，2-歌手，3-专辑",
				"data":    nil,
			})
			return
		}

		var content struct {
			ID             int64
			Title          string
			Status         int16
			IsDeleted      int16
			SubscribeCount int64
		}
		err := svcCtx.DB.Table("content").
			Select("id, title, status, is_deleted, subscribe_count").
			Where("id = ?", contentID).
			First(&content).Error

		if err != nil {
			logx.Errorf("查询音频失败: %v", err)
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "音频不存在",
				"data":    nil,
			})
			return
		}

		if content.IsDeleted == 1 || content.Status != 1 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "该音频不可订阅（已下架或已删除）",
				"data":    nil,
			})
			return
		}

		var existingSub struct {
			ID int64
		}
		svcCtx.DB.Table("user_subscriptions").
			Select("id").
			Where("user_id = ? AND target_id = ? AND subscribe_type = ?", bearerCtx.UserID, contentID, subscribeType).
			First(&existingSub)

		if existingSub.ID > 0 {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 200,
				"data": types.SubscribeResp{
					Success:        false,
					Message:        "您已订阅过该内容",
					ContentID:      contentID,
					SubscribeCount: content.SubscribeCount,
				},
			})
			return
		}

		now := time.Now()
		insertResult := svcCtx.DB.Table("user_subscriptions").Create(map[string]interface{}{
			"user_id":        bearerCtx.UserID,
			"subscribe_type": subscribeType,
			"target_id":      contentID,
			"target_name":    content.Title,
			"created_at":     now,
		})

		if insertResult.Error != nil {
			logx.Errorf("保存订阅记录失败: %v", insertResult.Error)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "保存订阅失败",
				"data":    nil,
			})
			return
		}

		svcCtx.DB.Table("content").
			Where("id = ?", contentID).
			Update("subscribe_count", gorm.Expr("COALESCE(subscribe_count, 0) + 1"))

		var newCount int64
		svcCtx.DB.Table("content").Select("subscribe_count").Where("id = ?", contentID).Scan(&newCount)

		logx.Infof("订阅成功: userID=%d, contentID=%d, title=%s, type=%d, newCount=%d",
			bearerCtx.UserID, contentID, content.Title, subscribeType, newCount)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": types.SubscribeResp{
				Success:        true,
				Message:        "订阅成功",
				ContentID:      contentID,
				SubscribeCount: newCount,
			},
		})
	}
}

// contentUnsubscribeHandler 取消订阅音频处理器
// contentUnsubscribeHandler 取消订阅音频处理器
// @Summary      取消订阅音频
// @Description  用户取消订阅歌曲/歌手/专辑，需要登录
// @Tags         订阅管理
// @Accept       json
// @Produce      json
// @Param        id   path      int64  true  "音频ID"
// @Param        body  body      types.UnsubscribeReq  true  "取消订阅请求"
// @Success      200  {object}  map[string]interface{}  "成功"
// @Failure      401  {object}  map[string]interface{}  "未登录"
// @Failure      400  {object}  map[string]interface{}  "参数错误"
// @Router       /{id}/subscribe [delete]
// @Security     BearerAuth
func contentUnsubscribeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 DELETE",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		path := strings.Trim(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		var contentID int64
		for i := len(parts) - 1; i >= 0; i-- {
			if id, err := strconv.ParseInt(parts[i], 10, 64); err == nil && id > 0 {
				contentID = id
				break
			}
		}
		if contentID <= 0 {
			var req types.UnsubscribeReq
			contentType := r.Header.Get("Content-Type")
			if strings.Contains(contentType, "application/json") {
				body, err := io.ReadAll(r.Body)
				if err == nil {
					json.Unmarshal(body, &req)
				}
			} else {
				r.ParseForm()
				if v := r.FormValue("content_id"); v != "" {
					contentID, _ = strconv.ParseInt(v, 10, 64)
				}
			}
			if req.ContentID > 0 {
				contentID = req.ContentID
			}
		}

		if contentID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "音频 ID 不能为空",
				"data":    nil,
			})
			return
		}

		subscribeType := int16(1)
		var req types.UnsubscribeReq
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &req)
		} else {
			r.ParseForm()
			if v := r.FormValue("type"); v != "" {
				if t, err := strconv.ParseInt(v, 10, 16); err == nil {
					subscribeType = int16(t)
				}
			}
		}
		if req.Type > 0 {
			subscribeType = req.Type
		}

		var existingSub struct {
			ID         int64
			TargetName string
		}
		svcCtx.DB.Table("user_subscriptions").
			Select("id, target_name").
			Where("user_id = ? AND target_id = ? AND subscribe_type = ?", bearerCtx.UserID, contentID, subscribeType).
			First(&existingSub)

		if existingSub.ID <= 0 {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 200,
				"data": types.UnsubscribeResp{
					Success:        false,
					Message:        "您未订阅该内容",
					ContentID:      contentID,
					SubscribeCount: 0,
				},
			})
			return
		}

		deleteResult := svcCtx.DB.Table("user_subscriptions").
			Where("id = ?", existingSub.ID).
			Delete(nil)

		if deleteResult.Error != nil {
			logx.Errorf("删除订阅记录失败: %v", deleteResult.Error)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "取消订阅失败",
				"data":    nil,
			})
			return
		}

		svcCtx.DB.Table("content").
			Where("id = ? AND subscribe_count > 0", contentID).
			Update("subscribe_count", gorm.Expr("GREATEST(subscribe_count - 1, 0)"))

		var newCount int64
		svcCtx.DB.Table("content").Select("subscribe_count").Where("id = ?", contentID).Scan(&newCount)

		logx.Infof("取消订阅成功: userID=%d, contentID=%d, targetName=%s, type=%d, newCount=%d",
			bearerCtx.UserID, contentID, existingSub.TargetName, subscribeType, newCount)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": types.UnsubscribeResp{
				Success:        true,
				Message:        "取消订阅成功",
				ContentID:      contentID,
				SubscribeCount: newCount,
			},
		})
	}
}

// contentSubscribeListHandler 订阅列表处理器
// @Summary      获取订阅列表
// @Description  查看用户订阅的音频列表，支持分页和类型筛选，需要登录
// @Tags         订阅管理
// @Accept       json
// @Produce      json
// @Param        page   query     int32  false  "页码"  default(1)
// @Param        page_size  query  int32  false  "每页数量"  default(20)
// @Param        type   query     int16  false  "订阅类型：1-歌曲 2-歌手 3-专辑"
// @Success      200  {object}  map[string]interface{}  "成功"
// @Failure      401  {object}  map[string]interface{}  "未登录"
// @Router       /subscriptions [get]
// @Security     BearerAuth
func contentSubscribeListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 GET",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		pageStr := r.URL.Query().Get("page")
		sizeStr := r.URL.Query().Get("page_size")
		subTypeStr := r.URL.Query().Get("type")

		page := int32(1)
		pageSize := int32(20)

		if pageStr != "" {
			if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
				page = int32(p)
			}
		}
		if sizeStr != "" {
			if s, err := strconv.ParseInt(sizeStr, 10, 32); err == nil && s > 0 && s <= 50 {
				pageSize = int32(s)
			}
		}

		var subscribeType *int16
		if subTypeStr != "" {
			if t, err := strconv.ParseInt(subTypeStr, 10, 16); err == nil && t >= 1 && t <= 3 {
				st := int16(t)
				subscribeType = &st
			}
		}

		query := svcCtx.DB.Table("user_subscriptions").
			Where("user_id = ?", bearerCtx.UserID)

		if subscribeType != nil {
			query = query.Where("subscribe_type = ?", *subscribeType)
		}

		var total int64
		query.Count(&total)

		offset := (page - 1) * pageSize

		type SubRecord struct {
			ID            int64
			TargetID      int64
			SubscribeType int16
			TargetName    string
			CreatedAt     time.Time
		}

		rows, err := query.
			Select("id, target_id, subscribe_type, target_name, created_at").
			Order("created_at DESC").
			Limit(int(pageSize)).
			Offset(int(offset)).
			Rows()

		if err != nil {
			logx.Errorf("查询订阅记录失败: %v", err)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "查询订阅记录失败",
				"data":    nil,
			})
			return
		}
		defer rows.Close()

		var targetIDs []int64
		var records []SubRecord

		for rows.Next() {
			var record SubRecord
			if err := rows.Scan(&record.ID, &record.TargetID, &record.SubscribeType, &record.TargetName, &record.CreatedAt); err != nil {
				continue
			}
			targetIDs = append(targetIDs, record.TargetID)
			records = append(records, record)
		}

		contentMap := make(map[int64]struct {
			Title    string
			Artist   string
			CoverURL string
			Duration int
			VipLevel int16
		})

		if len(targetIDs) > 0 {
			type ContentInfo struct {
				ID       int64
				Title    string
				Artist   string
				CoverURL string
				Duration int
				VipLevel int16
			}

			var contents []ContentInfo
			svcCtx.DB.Table("content").
				Select("id, title, artist, cover_url, duration_sec, vip_level").
				Where("id IN ?", targetIDs).
				Find(&contents)

			for _, c := range contents {
				contentMap[c.ID] = struct {
					Title    string
					Artist   string
					CoverURL string
					Duration int
					VipLevel int16
				}{
					Title:    c.Title,
					Artist:   c.Artist,
					CoverURL: c.CoverURL,
					Duration: c.Duration,
					VipLevel: c.VipLevel,
				}
			}
		}

		list := make([]types.SubscribeListItem, 0, len(records))
		for _, record := range records {
			info, ok := contentMap[record.TargetID]
			if !ok {
				info = struct {
					Title    string
					Artist   string
					CoverURL string
					Duration int
					VipLevel int16
				}{
					Title:    record.TargetName,
					Artist:   "",
					CoverURL: "",
					Duration: 0,
					VipLevel: 0,
				}
			}

			list = append(list, types.SubscribeListItem{
				ID:            record.ID,
				ContentID:     record.TargetID,
				Title:         info.Title,
				Artist:        info.Artist,
				CoverURL:      info.CoverURL,
				Duration:      info.Duration,
				VipLevel:      info.VipLevel,
				SubscribeType: record.SubscribeType,
				SubscribedAt:  record.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}

		logx.Infof("查询订阅列表: userID=%d, type=%v, page=%d, size=%d, total=%d",
			bearerCtx.UserID, subscribeType, page, pageSize, total)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": types.SubscribeListResp{
				Total:    total,
				List:     list,
				Page:     page,
				PageSize: pageSize,
			},
		})
	}
}

// contentArtistDetailHandler 歌手详情处理器
// @Summary      获取歌手详情
// @Description  查看歌手详细信息，包含热门歌曲、专辑列表等
// @Tags         歌手管理
// @Accept       json
// @Produce      json
// @Param        id   path      int64  true  "歌手ID"
// @Success      200  {object}  map[string]interface{}  "成功"
// @Failure      400  {object}  map[string]interface{}  "参数错误"
// @Failure      404  {object}  map[string]interface{}  "歌手不存在"
// @Router       /artists/{id} [get]
func contentArtistDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 GET",
				"data":    nil,
			})
			return
		}

		path := strings.Trim(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		var artistID int64
		for i := len(parts) - 1; i >= 0; i-- {
			if id, err := strconv.ParseInt(parts[i], 10, 64); err == nil && id > 0 {
				artistID = id
				break
			}
		}
		if artistID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌手 ID 不能为空",
				"data":    nil,
			})
			return
		}

		var artist struct {
			ID        int64
			Name      string
			AvatarURL string
			Bio       string
			FanCount  int64
			SongCount int64
			Status    int16
			CreatedAt time.Time
		}
		err := svcCtx.DB.Table("artists").
			Select("id, name, avatar_url, bio, fan_count, song_count, status, created_at").
			Where("id = ?", artistID).
			First(&artist).Error

		if err != nil {
			logx.Errorf("查询歌手失败: %v", err)
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "歌手不存在",
				"data":    nil,
			})
			return
		}

		if artist.Status != 1 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "该歌手已下架或被禁用",
				"data":    nil,
			})
			return
		}

		type HotSong struct {
			ID        int64
			Title     string
			CoverURL  string
			Duration  int
			PlayCount int64
			VipLevel  int16
		}

		var hotSongs []HotSong
		svcCtx.DB.Table("content").
			Select("id, title, cover_url, duration_sec, play_count, vip_level").
			Where("artist_id = ? AND status = 1 AND is_deleted = 0", artistID).
			Order("play_count DESC").
			Limit(10).
			Find(&hotSongs)

		hotSongList := make([]types.ArtistHotSong, 0, len(hotSongs))
		for _, song := range hotSongs {
			hotSongList = append(hotSongList, types.ArtistHotSong{
				ID:        song.ID,
				Title:     song.Title,
				CoverURL:  song.CoverURL,
				Duration:  song.Duration,
				PlayCount: song.PlayCount,
				VipLevel:  song.VipLevel,
			})
		}

		type AlbumInfo struct {
			ID          int64
			Name        string
			CoverURL    string
			SongCount   int
			PublishedAt time.Time
		}

		var albums []AlbumInfo
		svcCtx.DB.Table("albums").
			Select("id, name, cover_url, song_count, published_at").
			Where("artist_id = ? AND status = 1", artistID).
			Order("published_at DESC").
			Limit(10).
			Find(&albums)

		albumList := make([]types.ArtistAlbumItem, 0, len(albums))
		for _, album := range albums {
			albumList = append(albumList, types.ArtistAlbumItem{
				ID:          album.ID,
				Name:        album.Name,
				CoverURL:    album.CoverURL,
				SongCount:   album.SongCount,
				PublishedAt: album.PublishedAt.Format("2006-01-02"),
			})
		}

		var totalPlays int64
		svcCtx.DB.Table("content").
			Where("artist_id = ? AND is_deleted = 0", artistID).
			Select("COALESCE(SUM(play_count), 0)").
			Scan(&totalPlays)

		var albumCount int64
		svcCtx.DB.Table("albums").Where("artist_id = ? AND status = 1", artistID).Count(&albumCount)

		logx.Infof("查询歌手详情: artistID=%d, name=%s, hotSongs=%d, albums=%d",
			artistID, artist.Name, len(hotSongList), len(albumList))

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": types.ArtistDetailResp{
				ID:         artist.ID,
				Name:       artist.Name,
				AvatarURL:  artist.AvatarURL,
				Bio:        artist.Bio,
				FanCount:   artist.FanCount,
				SongCount:  artist.SongCount,
				AlbumCount: albumCount,
				TotalPlays: totalPlays,
				HotSongs:   hotSongList,
				Albums:     albumList,
			},
		})
	}
}

// contentSubscribeNotifyHandler 订阅通知处理器
// @Summary      发送订阅通知
// @Description  内容发布时触发通知给订阅用户（内部接口或管理员调用），需要登录
// @Tags         通知管理
// @Accept       json
// @Produce      json
// @Param        id   path      int64  true  "内容ID"
// @Param        body  body      types.SubscribeNotifyReq  true  "通知请求"
// @Success      200  {object}  map[string]interface{}  "成功"
// @Failure      401  {object}  map[string]interface{}  "未登录"
// @Failure      400  {object}  map[string]interface{}  "参数错误"
// @Router       /{id}/notify [post]
// @Security     BearerAuth
func contentSubscribeNotifyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 POST",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		path := strings.Trim(r.URL.Path, "/")
		parts := strings.Split(path, "/")
		var contentID int64
		for i := len(parts) - 1; i >= 0; i-- {
			if id, err := strconv.ParseInt(parts[i], 10, 64); err == nil && id > 0 {
				contentID = id
				break
			}
		}

		var req types.SubscribeNotifyReq
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				logx.Errorf("读取请求体失败: %v", err)
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "读取请求失败",
					"data":    nil,
				})
				return
			}
			json.Unmarshal(body, &req)
			if req.ContentID <= 0 && contentID > 0 {
				req.ContentID = contentID
			}
		} else {
			r.ParseForm()
			req.ContentID = contentID
			req.Title = r.FormValue("title")
			if v := r.FormValue("notify_type"); v != "" {
				if t, err := strconv.ParseInt(v, 10, 16); err == nil {
					req.NotifyType = int16(t)
				}
			}
			req.JumpURL = r.FormValue("jump_url")
		}

		if req.ContentID <= 0 || req.Title == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "内容ID和标题不能为空",
				"data":    nil,
			})
			return
		}

		if req.NotifyType < 1 || req.NotifyType > 3 {
			req.NotifyType = 1
		}

		type Subscriber struct {
			UserID   int64
			TargetID int64
		}

		var subscribers []Subscriber
		svcCtx.DB.Table("user_subscriptions").
			Select("user_id, target_id").
			Where("target_id = ? AND subscribe_type = ?", req.ContentID, 1).
			Find(&subscribers)

		if len(subscribers) == 0 {
			logx.Infof("无订阅者，跳过通知: contentID=%d", req.ContentID)
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 200,
				"data": types.SubscribeNotifyResp{
					Success:      true,
					Message:      "暂无订阅用户",
					TotalSent:    0,
					SuccessCount: 0,
					FailCount:    0,
					NotifyLogID:  0,
				},
			})
			return
		}

		now := time.Now()
		notifyLogResult := svcCtx.DB.Table("subscribe_notify_logs").Create(map[string]interface{}{
			"content_id":   req.ContentID,
			"title":        req.Title,
			"notify_type":  req.NotifyType,
			"jump_url":     req.JumpURL,
			"total_count":  len(subscribers),
			"triggered_by": bearerCtx.UserID,
			"status":       0,
			"created_at":   now,
		})

		var notifyLogID int64
		if notifyLogResult.Error == nil {
			svcCtx.DB.Table("subscribe_notify_logs").Select("id").Where("created_at = ?", now).Order("id DESC").First(&notifyLogID)
		}

		successCount := int64(0)
		failCount := int64(0)

		for _, sub := range subscribers {
			insertErr := svcCtx.DB.Table("user_notifications").Create(map[string]interface{}{
				"user_id":       sub.UserID,
				"content_id":    req.ContentID,
				"title":         req.Title,
				"notify_type":   req.NotifyType,
				"jump_url":      req.JumpURL,
				"is_read":       0,
				"notify_log_id": notifyLogID,
				"created_at":    now,
			}).Error

			status := int16(1)
			errorMsg := ""
			if insertErr != nil {
				status = 2
				errorMsg = insertErr.Error()
				failCount++
				logx.Errorf("生成通知失败: userID=%d, contentID=%d, error=%v",
					sub.UserID, req.ContentID, insertErr)
			} else {
				successCount++
			}

			svcCtx.DB.Table("subscribe_notify_details").Create(map[string]interface{}{
				"log_id":     notifyLogID,
				"user_id":    sub.UserID,
				"content_id": req.ContentID,
				"status":     status,
				"error_msg":  errorMsg,
				"sent_at":    now,
				"created_at": now,
			})
		}

		updateStatus := int16(1)
		if failCount > 0 && successCount == 0 {
			updateStatus = 2
		} else if failCount > 0 {
			updateStatus = 3
		}

		svcCtx.DB.Table("subscribe_notify_logs").
			Where("id = ?", notifyLogID).
			Updates(map[string]interface{}{
				"success_count": successCount,
				"fail_count":    failCount,
				"status":        updateStatus,
				"updated_at":    now,
			})

		logx.Infof("订阅通知完成: logID=%d, contentID=%d, title=%s, total=%d, success=%d, fail=%d",
			notifyLogID, req.ContentID, req.Title, len(subscribers), successCount, failCount)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": types.SubscribeNotifyResp{
				Success:      true,
				Message:      fmt.Sprintf("通知发送完成，成功%d条，失败%d条", successCount, failCount),
				TotalSent:    int64(len(subscribers)),
				SuccessCount: successCount,
				FailCount:    failCount,
				NotifyLogID:  notifyLogID,
			},
		})
	}
}

// contentDeleteHandler 删除歌曲处理器
// @Summary      删除歌曲
// @Description  删除歌曲，管理员或上传者才能删除，需要登录
// @Tags         内容管理
// @Accept       json
// @Produce      json
// @Param        id   path      int64  true  "歌曲ID"
// @Success      200  {object}  map[string]interface{}  "成功"
// @Failure      401  {object}  map[string]interface{}  "未登录"
// @Failure      403  {object}  map[string]interface{}  "无权限"
// @Failure      404  {object}  map[string]interface{}  "歌曲不存在"
// @Router       /{id} [delete]
// @Security     BearerAuth
func contentDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"message": "仅支持 DELETE",
				"data":    nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "请先登录",
				"data":    nil,
			})
			return
		}

		contentIDStr := r.URL.Query().Get("content_id")
		if contentIDStr == "" {
			path := r.URL.Path
			parts := strings.Split(strings.Trim(path, "/"), "/")
			if len(parts) > 0 {
				contentIDStr = parts[len(parts)-1]
			}
		}

		contentID, err := strconv.ParseInt(contentIDStr, 10, 64)
		if err != nil || contentID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌曲 ID 格式无效",
				"data":    nil,
			})
			return
		}

		var req types.ContentDeleteReq
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, err := io.ReadAll(r.Body)
			if err == nil {
				json.Unmarshal(body, &req)
			}
		}
		if req.ContentID > 0 {
			contentID = req.ContentID
		}

		var content struct {
			ID         int64
			Title      string
			AudioURL   string
			CoverURL   string
			IsDeleted  int16
			UploaderID *int64
		}
		err = svcCtx.DB.Table("content").
			Select("id, title, audio_url, cover_url, is_deleted, uploader_id").
			Where("id = ?", contentID).
			First(&content).Error

		if err != nil {
			logx.Errorf("查询歌曲失败: %v", err)
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": "歌曲不存在",
				"data":    nil,
			})
			return
		}

		if content.IsDeleted == 1 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "歌曲已被删除",
				"data":    nil,
			})
			return
		}

		var isAdmin bool
		var adminID int64
		svcCtx.DB.Table("sys_admin").Select("id").Where("id = ?", bearerCtx.UserID).Scan(&adminID)
		if adminID > 0 {
			isAdmin = true
		}

		isUploader := content.UploaderID != nil && *content.UploaderID == bearerCtx.UserID

		if !isAdmin && !isUploader {
			httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
				"code":    403,
				"message": "无权限删除此歌曲",
				"data":    nil,
			})
			return
		}

		var likeCount int64
		svcCtx.DB.Table("user_likes").Where("content_id = ?", contentID).Count(&likeCount)

		var downloadCount int64
		svcCtx.DB.Table("user_downloads").Where("content_id = ?", contentID).Count(&downloadCount)

		var playCount int64
		svcCtx.DB.Table("play_history").Where("content_id = ?", contentID).Count(&playCount)

		var playlistCount int64
		svcCtx.DB.Table("playlist_songs").Where("content_id = ?", contentID).Count(&playlistCount)

		logx.Infof("删除歌曲关联数据: contentID=%d, title=%s, likes=%d, downloads=%d, plays=%d, playlists=%d",
			contentID, content.Title, likeCount, downloadCount, playCount, playlistCount)

		if content.AudioURL != "" {
			localPath := getLocalFilePath(svcCtx, content.AudioURL)
			if localPath != "" {
				if err := os.Remove(localPath); err == nil {
					logx.Infof("删除音频文件成功: %s", localPath)
				} else {
					logx.Errorf("删除音频文件失败: %s, error: %v", localPath, err)
				}
			}
		}

		if content.CoverURL != "" && !strings.HasPrefix(content.CoverURL, "http") {
			localPath := getLocalFilePath(svcCtx, content.CoverURL)
			if localPath != "" {
				if err := os.Remove(localPath); err == nil {
					logx.Infof("删除封面文件成功: %s", localPath)
				} else {
					logx.Errorf("删除封面文件失败: %s, error: %v", localPath, err)
				}
			}
		}

		svcCtx.DB.Table("user_likes").Where("content_id = ?", contentID).Delete(nil)
		svcCtx.DB.Table("user_downloads").Where("content_id = ?", contentID).Delete(nil)
		svcCtx.DB.Table("play_history").Where("content_id = ?", contentID).Delete(nil)
		svcCtx.DB.Table("playlist_songs").Where("content_id = ?", contentID).Delete(nil)

		now := time.Now()
		result := svcCtx.DB.Table("content").Where("id = ?", contentID).Updates(map[string]interface{}{
			"is_deleted": 1,
			"status":     2,
			"updated_at": now,
		})

		if result.Error != nil {
			logx.Errorf("删除歌曲失败: %v", result.Error)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "删除歌曲失败",
				"data":    nil,
			})
			return
		}

		reason := req.Reason
		if reason == "" {
			if isAdmin {
				reason = "管理员删除"
			} else {
				reason = "上传者删除"
			}
		}

		logx.Infof("歌曲删除成功: contentID=%d, title=%s, reason=%s, operator=%d, isAdmin=%v",
			contentID, content.Title, reason, bearerCtx.UserID, isAdmin)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"data": map[string]interface{}{
				"success":    true,
				"message":    "歌曲删除成功",
				"content_id": contentID,
				"title":      content.Title,
				"deleted_at": now.Format("2006-01-02 15:04:05"),
			},
		})
	}
}
