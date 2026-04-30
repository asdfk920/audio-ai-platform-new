package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// devicePlayPlaylistHandler 设备播放歌单指令处理器
// POST /api/device/cmd/play_playlist
// 用途：用户通过 App 向设备下发播放歌单指令（需 JWT 鉴权）
// 流程：
//   1. 接收 POST 请求，解析 Authorization 头获取用户 Token
//   2. 解析请求体 JSON 数据，获取 sn、action、params 参数
//   3. 校验 Token 和参数格式
//   4. 验证用户权限（查询 user_device_bind 绑定表）
//   5. 查询歌单信息（user_playlists 表）
//   6. 查询歌曲列表（playlist_songs 表）
//   7. 检查设备在线状态（Redis + MySQL）
//   8. 在线时通过 MQTT 立即下发，离线时缓存指令
//   9. 记录命令日志并返回响应
func devicePlayPlaylistHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求方法
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		// 解析请求体
		var req types.DevicePlayPlaylistReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "请求体格式错误",
				"data": nil,
			})
			return
		}

		// 调用业务逻辑
		l := logic.NewDevicePlayPlaylistLogic(r.Context(), svcCtx)
		resp, err := l.DevicePlayPlaylist(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  err.Error(),
				"data": nil,
			})
			return
		}

		// 返回成功响应
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"msg":  "操作成功",
			"data": resp,
		})
	})
}