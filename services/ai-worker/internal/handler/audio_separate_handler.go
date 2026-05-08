package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/middleware/auth"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
)

// AudioSeparateHandler 音频分离处理
// @Summary 发起音频分离任务
// @Description 根据 content_id 查询音频 URL 并发起 AI 分离任务
// @Tags 音频分离
// @Accept json
// @Produce json
// @Param data body logic.AudioSeparateReq true "分离请求"
// @Success 200 {object} logic.AudioSeparateResp
// @Router /api/v1/audio/separate [post]
func AudioSeparateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req logic.AudioSeparateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("请求参数错误：%v", err), http.StatusBadRequest)
			return
		}

		// 从 Context 获取用户 ID（由 JWT 中间件注入）
		userID := auth.GetUserIDFromContext(r.Context())
		if userID == 0 {
			http.Error(w, "用户未登录", http.StatusUnauthorized)
			return
		}

		// 创建 logic 实例
		audioLogic := logic.NewAudioSeparateLogic(svcCtx)

		// 调用 logic 处理分离
		resp, err := audioLogic.SeparateAudio(&req, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 返回 JSON 响应
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
