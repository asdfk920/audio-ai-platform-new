package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/middleware/auth"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
)

// TaskDeleteHandler 任务删除处理
// @Summary 删除历史分离任务
// @Description 根据任务ID删除音频分离任务及其关联的音轨数据，只能删除自己的任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param task_id path string false "任务标识符 (UUID格式)"
// @Param id query int false "主键ID (数字)"
// @Success 200 {object} logic.TaskDeleteResp
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 403 {object} map[string]string "无权访问"
// @Failure 404 {object} map[string]string "任务不存在"
// @Router /api/v1/inference/tasks/{task_id} [delete]
func TaskDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := getTaskIDFromPath(r)
		id := getIDFromRequest(r)

		if taskID == "" && id == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "任务 ID 或主键 ID 不能为空",
				"hint":  "请提供 task_id (如: test_001) 或 id (如: 2)",
				"examples": []string{
					"DELETE /api/v1/inference/tasks/test_001",
					"DELETE /api/v1/inference/tasks?id=2",
					"DELETE /api/v1/inference/tasks?task_id=test_001&id=2",
				},
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

		taskDeleteLogic := logic.NewTaskDeleteLogic(svcCtx)

		resp, err := taskDeleteLogic.DeleteTask(&logic.TaskDeleteReq{
			TaskID: taskID,
			ID:     id,
		}, userID)

		if err != nil {
			w.Header().Set("Content-Type", "application/json")

			errorMsg := err.Error()

			switch errorMsg {
			case "任务不存在或无权访问":
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": errorMsg,
					"hint":  "请检查任务ID是否正确，或者该任务不属于您",
				})
				return
			case "任务不存在或已被删除":
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": errorMsg,
					"hint":  "该任务可能已被删除，请刷新任务列表",
				})
				return
			default:
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": errorMsg,
				})
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}

// getIDFromRequest 从请求中提取主键 ID
// 支持多种格式（按优先级）：
//  1. URL 路径参数 /api/v1/inference/tasks/{数字}
//  2. 查询参数 ?id=2
//  3. 查询参数 ?ID=2
func getIDFromRequest(r *http.Request) int64 {
	pathParts := splitPath(r.URL.Path)

	if len(pathParts) >= 4 && pathParts[3] != "" {
		lastPart := pathParts[len(pathParts)-1]
		if lastPart != "tasks" {
			if id, err := strconv.ParseInt(lastPart, 10, 64); err == nil {
				return id
			}
		}
	}

	if idStr := r.URL.Query().Get("id"); idStr != "" {
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			return id
		}
	}

	if idStr := r.URL.Query().Get("ID"); idStr != "" {
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			return id
		}
	}

	return 0
}
