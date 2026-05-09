package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/middleware/auth"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
)

// TaskDetailHandler 任务详情处理
// @Summary 查询单个分离任务详情
// @Description 根据任务ID查询音频分离任务的详细信息，包括进度、状态、音轨列表等，只能查询自己的任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param task_id path string true "任务ID"
// @Success 200 {object} logic.TaskDetailResp
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 403 {object} map[string]string "无权访问"
// @Failure 404 {object} map[string]string "任务不存在"
// @Router /api/v1/inference/tasks/{task_id} [get]
func TaskDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := getTaskIDFromPath(r)

		if taskID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "任务 ID 不能为空",
			})
			return
		}

		userID := auth.GetUserIDFromContext(r.Context())
		if userID == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "用户未登录或 Token 无效",
			})
			return
		}

		taskLogic := logic.NewTaskDetailLogic(svcCtx)

		resp, err := taskLogic.GetTaskDetail(&logic.TaskDetailReq{
			TaskID: taskID,
		}, userID)

		if err != nil {
			w.Header().Set("Content-Type", "application/json")

			if err.Error() == "任务不存在或无权访问" {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{
					"error": err.Error(),
					"hint":  "请检查任务ID是否正确，或者该任务不属于您",
				})
				return
			}

			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

// getTaskIDFromPath 从 URL 路径或查询参数中提取任务ID
// 支持三种格式（按优先级）：
//  1. /api/v1/inference/tasks/{task_id}        （RESTful 路径参数）
//  2. /api/v1/inference/tasks?task_id={task_id}  （查询参数）
//  3. /api/v1/inference/tasks?TASK_ID={task_id}   （兼容大写）
func getTaskIDFromPath(r *http.Request) string {
	pathParts := splitPath(r.URL.Path)

	if len(pathParts) >= 4 && pathParts[3] != "" {
		lastPart := pathParts[len(pathParts)-1]
		if lastPart != "tasks" {
			return lastPart
		}
	}

	if taskID := r.URL.Query().Get("task_id"); taskID != "" {
		return taskID
	}

	if taskID := r.URL.Query().Get("TASK_ID"); taskID != "" {
		return taskID
	}

	return ""
}

// splitPath 分割 URL 路径
func splitPath(path string) []string {
	if len(path) == 0 || path[0] != '/' {
		return []string{}
	}

	var parts []string
	start := 1

	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			parts = append(parts, path[start:i])
			start = i + 1
		}
	}

	if start < len(path) {
		parts = append(parts, path[start:])
	}

	return parts
}
