package handler

import (
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
			"user_id":        userID,
			"content_id":     contentID,
			"content_title":  title,
			"file_url":       audioURL,
			"download_time":  time.Now(),
			"status":         2,
			"file_size":      fileSize,
			"created_at":     time.Now(),
			"sync_status":    0,
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
