package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// devicePlaybackStatusHandler 设备播放状态查询处理器
// GET /api/device/status/playback
// 用途：用户通过 App 查询设备当前播放状态（需 JWT 鉴权）
// 流程：
//   1. 接收 GET 请求，解析 Authorization 头获取用户 Token
//   2. 从 Query 参数获取 sn 设备序列号
//   3. 校验 Token 和参数格式
//   4. 验证用户权限（查询 user_device_bind 绑定表）
//   5. 查询设备在线状态（Redis + MySQL）
//   6. 查询播放状态（Redis 缓存优先，MySQL 数据库备选）
//   7. 组装播放状态响应数据
//   8. 返回播放状态信息
func devicePlaybackStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求方法
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 GET",
				"data": nil,
			})
			return
		}

		// 从 Query 参数获取 sn
		sn := r.URL.Query().Get("sn")
		if sn == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "设备序列号不能为空",
				"data": nil,
			})
			return
		}

		// 构造请求对象
		req := types.DevicePlaybackStatusReq{
			Sn: sn,
		}

		// 调用业务逻辑
		l := logic.NewDevicePlaybackStatusLogic(r.Context(), svcCtx)
		resp, err := l.DevicePlaybackStatus(&req)
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
			"msg":  "查询成功",
			"data": resp,
		})
	})
}