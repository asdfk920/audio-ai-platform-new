package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// DeviceDownloadSongHandler 设备下载歌曲处理器
// POST /api/device/cmd/download_song
//
// 功能：用户通过前端/APP 发起下载，后端向目标设备推送下载指令。
// 流程：参数校验 → 用户鉴权 → 设备与绑定校验 →（按需）内容服务解析 play_url → 创建指令 → WebSocket/队列下发。
//
// 请求体必填：sn、content_id。
// 未传 source_url 时须在设备配置 Content.BaseURL，且请求携带与用户一致的 Authorization（Bearer JWT），以便代调内容服务校验 can_download 并获取 play_url。
func DeviceDownloadSongHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		logxContext := r.Context()

		logx.Infof("====================================")
		logx.Infof("[Device Download Handler] 收到下载歌曲请求")
		logx.Infof("   Method: %s", r.Method)
		logx.Infof("   URL: %s", r.URL.String())
		logx.Infof("   RemoteAddr: %s", r.RemoteAddr)
		logx.Infof("====================================")

		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.DeviceDownloadSongReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  fmt.Sprintf("请求体格式错误: %v", err),
				"data": nil,
			})
			return
		}

		authz := strings.TrimSpace(r.Header.Get("Authorization"))

		l := logic.NewDeviceDownloadSongLogic(logxContext, svcCtx)
		resp, err := l.DownloadSong(&req, authz)

		if err != nil {
			logx.Errorf("[Device Download Handler] 处理失败: %v", err)

			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 4001,
				"msg":  err.Error(),
				"data": nil,
			})
			return
		}

		logx.Infof("[Device Download Handler] ✅ 处理成功: task_id=%s, status=%s", resp.TaskID, resp.Status)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"msg":  "操作成功",
			"data": resp,
		})
	})
}
