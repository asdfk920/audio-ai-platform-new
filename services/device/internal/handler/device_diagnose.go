package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// deviceDiagnoseHandler 设备远程诊断处理器
// POST /api/device/diagnose
// 用途：用户通过 App 发起设备远程诊断，后端通过 MQTT 下发诊断指令
func deviceDiagnoseHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.DeviceDiagnoseReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "请求体格式错误",
				"data": nil,
			})
			return
		}

		userID, ok := jwt.GetUserIdFromContext(r.Context())
		if !ok {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code": 401,
				"msg":  "用户身份验证失败",
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceDiagnoseLogic(r.Context(), svcCtx)
		resp, err := l.DeviceDiagnose(&req, userID)
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
			"msg":  "诊断任务已下发",
			"data": resp,
		})
	})
}

// DeviceDiagnoseHandler 设备远程诊断
// @Summary      设备远程诊断
// @Description  用户通过 App 发起设备远程诊断
// @Tags         设备管理
// @Accept       json
// @Produce      json
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /api/v1/device/diagnose [post]
// @Security     BearerAuth
func DeviceDiagnoseHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return deviceDiagnoseHandler(svcCtx)
}
