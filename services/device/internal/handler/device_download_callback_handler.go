package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/device/internal/deviceauthsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/pkg/ip"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// deviceDownloadCallbackHandler 设备歌曲下载结果回调。
// POST /api/device/download/callback
//
// 设备下载成功/失败后回调：
// {
//   "task_id": "...",
//   "sn": "AUSP2605000002Y2",
//   "status": "success|fail|failed",
//   "error_msg": "失败原因",
//   "local_path": "/sdcard/music/xxx.mp3",
//   "file_size": 123,
//   "duration_ms": 4567
// }
//
// 认证：Authorization: Bearer <device_token>，或 sn + X-Device-Secret。
func deviceDownloadCallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	auth := deviceauthsvc.New(svcCtx)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.WSDownloadResultReport
		dec := json.NewDecoder(r.Body)
		dec.UseNumber()
		if err := dec.Decode(&req); err != nil {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "请求体不是合法 JSON"))
			return
		}
		req.Sn = strings.ToUpper(strings.TrimSpace(req.Sn))
		req.TaskID = strings.TrimSpace(req.TaskID)
		if req.TaskID == "" {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "task_id 必填"))
			return
		}

		clientIP := ip.ClientIPTrusted(r, svcCtx.RegisterTrustedNets)
		ctx := r.Context()
		ctx = deviceauthsvc.WithClientIP(ctx, clientIP)
		if tok := deviceauthsvc.ExtractBearerToken(r.Header.Get("Authorization")); tok != "" {
			ctx = deviceauthsvc.WithBearerToken(ctx, tok)
		}

		secret := strings.TrimSpace(r.Header.Get(headerDeviceSecret))
		if deviceauthsvc.BearerTokenFromContext(ctx) == "" && (req.Sn == "" || secret == "") {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, errorx.Error(errorx.CodeTokenInvalid, "请使用设备 Bearer Token，或提供 sn + X-Device-Secret"))
			return
		}

		principal, err := auth.AuthenticateRequest(ctx, req.Sn, secret, clientIP)
		if err != nil {
			writeAuthErr(w, r, err)
			return
		}
		if req.Sn != "" && !strings.EqualFold(principal.DeviceSN, req.Sn) {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "sn 与设备凭证不匹配"))
			return
		}
		if req.DeviceID > 0 && principal.DeviceID != req.DeviceID {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, "device_id 与设备凭证不匹配"))
			return
		}
		req.Sn = strings.TrimSpace(principal.DeviceSN)
		req.DeviceID = principal.DeviceID

		auth.TouchAuthSuccess(ctx, principal, clientIP, "")

		payload, err := logic.ProcessDownloadFinishReport(ctx, svcCtx, req, req.Sn)
		if err != nil {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusBadRequest, errorx.Error(errorx.CodeInvalidParam, err.Error()))
			return
		}

		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"msg":  "下载结果回调接收成功",
			"data": payload,
		})
	}
}
