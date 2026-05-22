package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// POST /api/v1/song/download
// 歌曲下载：创建 download_song 指令并通过 WebSocket 下发到设备
// 用户通过前端页面点击下载按钮，后端校验权限后下发指令
func SongDownloadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.SongDownloadReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "请求体格式错误",
				"data": nil,
			})
			return
		}

		l := logic.NewSongDownloadLogic(r.Context(), svcCtx)
		resp, err := l.SongDownload(&req)

		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 9001,
				"msg":  err.Error(),
				"data": nil,
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"msg":  "操作成功",
			"data": resp,
		})
	})
}
