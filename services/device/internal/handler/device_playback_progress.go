package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// devicePlaybackProgressHandler 设备播放进度查询处理器
// GET /api/device/status/progress
// 用途：用户通过 App 查询设备当前播放进度（需 JWT 鉴权）
// 流程：
//   1. 接收 GET 请求，解析 Authorization 头获取用户 Token
//   2. 从 Query 参数获取 sn 设备序列号
//   3. 校验 Token 和参数格式（SN 必须为 16 位字母数字组合）
//   4. 验证用户权限（查询 user_device_bind 绑定表）
//   5. 查询设备在线状态（Redis + MySQL）
//   6. 查询播放进度数据（Redis 缓存优先，键名 device:progress:{sn}）
//   7. Redis 未命中时查询 MySQL 数据库 device_playback_status 表
//   8. 计算进度百分比和剩余时间
//   9. 组装播放进度响应数据
//  10. 返回播放进度信息（current_time, duration, percentage, remaining_time）
func devicePlaybackProgressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 GET",
				"data": nil,
			})
			return
		}

		sn := r.URL.Query().Get("sn")
		if sn == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "设备序列号不能为空",
				"data": nil,
			})
			return
		}

		req := types.DevicePlaybackProgressReq{
			Sn: sn,
		}

		l := logic.NewDevicePlaybackProgressLogic(r.Context(), svcCtx)
		resp, err := l.DevicePlaybackProgress(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  err.Error(),
				"data": nil,
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"msg":  "查询成功",
			"data": resp,
		})
	})
}