package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// diagnosisCommandHandler 诊断指令下发处理器
// POST /api/device/diagnosis/command
// 用途：用户通过App/后台发起设备日志收集等诊断指令，通过WebSocket下发给设备
func diagnosisCommandHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  fmt.Sprintf("仅支持 POST 请求，当前方法: %s。请使用 POST 方法访问: POST /api/v1/device/diagnosis/command", r.Method),
				"data": map[string]interface{}{
					"allowed_methods": []string{"POST"},
					"current_method":  r.Method,
					"correct_url":     "/api/v1/device/diagnosis/command",
					"usage_hint":      "请在Apifox/Postman中将请求方法改为POST，并确保Body格式为JSON",
				},
			})
			return
		}

		var req types.DiagnosisCommandReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "请求体格式错误: " + err.Error(),
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

		l := logic.NewDiagnosisCommandLogic(r.Context(), svcCtx)
		resp, err := l.SendDiagnosisCommand(&req, userID)
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
			"msg":  "诊断指令已下发",
			"data": resp,
		})
	})
}

// DiagnosisCommandHandler 诊断指令下发
// @Summary      诊断指令下发
// @Description  用户通过App/后台发起设备日志收集等诊断指令，通过WebSocket下发给在线设备
// @Tags         设备诊断
// @Accept       json
// @Produce      json
// @Param        request body types.DiagnosisCommandReq true "诊断指令请求"
// @Success      200  {object}  errorx.Response{data=types.DiagnosisCommandResp}  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /api/v1/device/diagnosis/command [post]
// @Security     BearerAuth
func DiagnosisCommandHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return diagnosisCommandHandler(svcCtx)
}
