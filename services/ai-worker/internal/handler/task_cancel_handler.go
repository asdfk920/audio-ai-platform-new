package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/middleware/auth"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
)

// TaskCancelHandler 任务取消处理
// @Summary 取消分离任务
// @Description 取消正在进行的音频分离任务（仅限 pending 或 processing 状态），通过 WebSocket 实时通知用户
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param task_id path string false "任务标识符 (UUID格式)"
// @Param id query int false "主键ID (数字)"
// @Success 200 {object} logic.TaskCancelResp
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 403 {object} map[string]string "无权访问或状态不允许"
// @Failure 404 {object} map[string]string "任务不存在"
// @Router /api/v1/inference/tasks/{task_id}/cancel [post]
func TaskCancelHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
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
					"POST /api/v1/inference/tasks/test_001/cancel",
					"POST /api/v1/inference/tasks/cancel?id=2",
					"POST /api/v1/inference/tasks/cancel?task_id=test_001&id=2",
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

		taskCancelLogic := logic.NewTaskCancelLogic(svcCtx)

		resp, err := taskCancelLogic.CancelTask(&logic.TaskCancelReq{
			TaskID: taskID,
			ID:     id,
		}, userID)

		if err != nil {
			w.Header().Set("Content-Type", "application/json")

			errorMsg := err.Error()

			switch {
			case errorMsg == "任务不存在或无权访问":
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": errorMsg,
					"hint":  "请检查任务ID是否正确，或者该任务不属于您",
				})
				return

			case errorMsg == "无法取消已完成的任务":
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": errorMsg,
					"hint":  "已完成的任务无法取消，如需清理请使用删除接口 DELETE /api/v1/inference/tasks/{id}",
				})
				return

			case errorMsg == "无法取消已失败的任务（建议使用删除功能）":
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": errorMsg,
					"hint":  "失败的任务无法取消，请使用删除接口清理：DELETE /api/v1/inference/tasks/{id}",
				})
				return

			case containsAny(errorMsg, []string{"只能取消", "无权取消"}):
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": errorMsg,
					"hint":  "只能取消待处理(pending)或处理中(processing)状态的任务",
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

// containsAny 检查字符串是否包含任一子串
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(substr) > 0 && containsString(s, substr) {
			return true
		}
	}
	return false
}

// containsString 检查字符串是否包含子串
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr, 0))
}

func containsSubstringHelper(s, substr string, pos int) bool {
	if pos+len(substr) > len(s) {
		return false
	}
	if s[pos:pos+len(substr)] == substr {
		return true
	}
	return containsSubstringHelper(s, substr, pos+1)
}
