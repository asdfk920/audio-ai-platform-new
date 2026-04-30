package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// deviceShadowQueryHandler 设备影子查询处理器
// GET /api/device/shadow?sn=xxx
// 用途：用户查询指定设备的最新状态数据（需 JWT 鉴权）
func deviceShadowQueryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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
				"msg":  "sn 参数不能为空",
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceShadowQueryLogic(r.Context(), svcCtx)
		resp, err := l.DeviceShadowQuery(sn)
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
