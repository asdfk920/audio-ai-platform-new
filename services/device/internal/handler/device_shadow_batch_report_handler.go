package handler

import (
	"io"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

// deviceShadowBatchReportHandler 网关批量上报子设备影子（仅 reported）
// POST /api/device/shadow/batch-report
func deviceShadowBatchReportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": errorx.CodeInvalidParam,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		raw, err := io.ReadAll(r.Body)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": errorx.CodeInvalidParam,
				"msg":  "读取请求体失败",
				"data": nil,
			})
			return
		}

		req, err := logic.ParseAndValidateBatchReportBody(raw)
		if err != nil {
			httpx.Error(w, err)
			return
		}

		l := logic.NewDeviceShadowBatchReportLogic(r.Context(), svcCtx)
		resp, err := l.BatchReport(req)
		if err != nil {
			httpx.Error(w, err)
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": errorx.CodeSuccess,
			"msg":  "success",
			"data": resp,
		})
	}
}

// DeviceShadowBatchReportHandler 设备端批量影子上报
func DeviceShadowBatchReportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return deviceShadowBatchReportHandler(svcCtx)
}
