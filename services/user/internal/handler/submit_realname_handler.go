// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// SubmitRealNameHandler 提交实名认证处理器
// POST /api/v1/user/realname/submit
// 用途：用户提交实名认证信息（姓名、身份证号、证件照片等），等待后台审核
func SubmitRealNameHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// #region agent log
		// debug-mode log (no secrets / no PII)
		func() {
			type payload struct {
				SessionId     string         `json:"sessionId"`
				RunId         string         `json:"runId"`
				HypothesisId  string         `json:"hypothesisId"`
				Location      string         `json:"location"`
				Message       string         `json:"message"`
				Data          map[string]any `json:"data,omitempty"`
				TimestampUnix int64          `json:"timestamp"`
			}
			p := payload{
				SessionId:    "046601",
				RunId:        "pre-fix",
				HypothesisId: "H1_handler_hit",
				Location:     "submit_realname_handler.go:SubmitRealNameHandler",
				Message:      "handler entered",
				Data: map[string]any{
					"method": r.Method,
					"path":   r.URL.Path,
				},
				TimestampUnix: time.Now().UnixMilli(),
			}
			b, _ := json.Marshal(p)
			f, err := os.OpenFile("C:\\Users\\Lenovo\\Desktop\\audio-ai-platform\\debug-046601.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err == nil {
				_, _ = f.Write(append(b, '\n'))
				_ = f.Close()
			}
		}()
		// #endregion

		var req types.RealNameSubmitReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		clientIP := r.RemoteAddr
		userAgent := r.UserAgent()

		l := logic.NewSubmitRealNameLogic(r.Context(), svcCtx)
		resp, err := l.Submit(&req, clientIP, userAgent)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
