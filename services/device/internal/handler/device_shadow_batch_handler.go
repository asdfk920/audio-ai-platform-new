package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// deviceShadowBatchHandler 批量更新设备影子（Redis Lua 原子 CAS + DB 事务，全成功或全失败）
// POST /api/device/shadow/batch
func deviceShadowBatchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.DeviceShadowBatchReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "请求参数格式错误",
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceShadowBatchLogic(r.Context(), svcCtx)
		resp, err := l.BatchUpdate(&req)
		if err != nil {
			httpx.Error(w, err)
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"msg":  "批量更新成功",
			"data": resp,
		})
	}
}

// DeviceShadowBatchHandler 批量更新设备影子
func DeviceShadowBatchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return deviceShadowBatchHandler(svcCtx)
}
