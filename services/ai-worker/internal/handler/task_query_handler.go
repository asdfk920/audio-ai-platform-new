package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/middleware/auth"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
)

// TaskListHandler 任务列表处理
// @Summary 查询分离任务列表
// @Description 查询用户的音频分离任务列表，支持分页和状态筛选，只能查询自己的任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param status query string false "状态筛选"
// @Success 200 {object} logic.TaskListResp
// @Router /api/v1/inference/tasks [get]
func TaskListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从 Query 参数获取分页和筛选参数
		page := 1
		pageSize := 20
		status := r.URL.Query().Get("status")

		if p := r.URL.Query().Get("page"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil {
				page = parsed
			}
		}

		if ps := r.URL.Query().Get("page_size"); ps != "" {
			if parsed, err := strconv.Atoi(ps); err == nil {
				pageSize = parsed
			}
		}

		// 从 Context 获取用户 ID（由 JWT 中间件注入）
		userID := auth.GetUserIDFromContext(r.Context())
		if userID == 0 {
			http.Error(w, "用户未登录", http.StatusUnauthorized)
			return
		}

		// 创建 logic 实例
		taskLogic := logic.NewTaskListLogic(svcCtx)

		// 调用 logic 查询任务列表
		resp, err := taskLogic.GetTaskList(&logic.TaskListReq{
			Page:     page,
			PageSize: pageSize,
			Status:   status,
		}, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 返回 JSON 响应
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
