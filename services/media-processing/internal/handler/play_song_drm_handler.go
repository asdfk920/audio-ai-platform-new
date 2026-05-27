package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/types"
)

// PlaySongWithDRMHandler 播放/下载歌曲（DRM透传版本）
// GET /api/v1/song/play?song_id=xxx&quality=hq
// POST /api/v1/song/play
//
// 功能：原样透传第三方加密音频流+DRM信息给前端/设备
// 合规原则：不解密、不破解、不转码、不存储明文，只做透传
//
// 请求参数：
//   - song_id: 歌曲ID（必填）
//   - quality: 音质（可选，standard/hq/hires，默认hq）
//
// 响应示例：
//
//	{
//	  "success": true,
//	  "message": "获取成功（DRM保护，原样透传）",
//	  "song_id": 12345,
//	  "song_name": "夜曲",
//	  "stream_url": "https://cdn.example.com/music/encrypted.mp3",
//	  "drm_info": {
//	    "drm_type": "widevine",
//	    "content_id": "uuid-song-123",
//	    "license_url": "https://license.example.com/widevine"
//	  },
//	  "copyright": "©2026 唱片公司",
//	  "encrypted": true,
//	  "quality": "hq"
//	}
func PlaySongWithDRMHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logx.Infof("====================================")
		logx.Infof("[DRM Play Song Handler] 收到播放/下载请求")
		logx.Infof("   Method: %s", r.Method)
		logx.Infof("   URL: %s", r.URL.String())
		logx.Infof("   RemoteAddr: %s", r.RemoteAddr)
		logx.Infof("====================================")

		var req types.PlaySongWithDRMReq

		if r.Method == http.MethodGet {
			songIDStr := r.URL.Query().Get("song_id")
			if songIDStr == "" {
				http.Error(w, `{"success":false,"message":"歌曲ID不能为空"}`, http.StatusBadRequest)
				return
			}
			if songID, err := strconv.ParseInt(songIDStr, 10, 64); err == nil {
				req.SongID = songID
			} else {
				http.Error(w, `{"success":false,"message":"歌曲ID格式错误"}`, http.StatusBadRequest)
				return
			}
			req.Quality = r.URL.Query().Get("quality")
			req.Token = r.URL.Query().Get("token")
		} else if r.Method == http.MethodPost {
			contentType := r.Header.Get("Content-Type")
			if strings.Contains(contentType, "application/json") {
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, `{"success":false,"message":"请求体格式错误"}`, http.StatusBadRequest)
					return
				}
			} else {
				r.ParseForm()
				if songIDStr := r.FormValue("song_id"); songIDStr != "" {
					if songID, err := strconv.ParseInt(songIDStr, 10, 64); err == nil {
						req.SongID = songID
					}
				}
				req.Quality = r.FormValue("quality")
				req.Token = r.FormValue("token")
			}
		}
		if strings.TrimSpace(req.Token) == "" {
			req.Token = strings.TrimSpace(r.Header.Get("Authorization"))
		}

		l := logic.NewPlaySongWithDRMLogic(r.Context(), svcCtx)
		resp, err := l.PlaySong(&req)

		if err != nil {
			logx.Errorf("[DRM Play Song Handler] 处理失败: %v", err)

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"success":false,"message":"` + escapeJSON(err.Error()) + `"}`))
			return
		}

		logx.Infof("[DRM Play Song Handler] ✅ 处理成功: song_id=%d, encrypted=%v", resp.SongID, resp.Encrypted)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-DRM-Protected", "true")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}
