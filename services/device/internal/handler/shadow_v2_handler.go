package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	shadowv2 "github.com/jacklau/audio-ai-platform/services/device/internal/device/shadowv2"
	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

type ShadowV2Handler struct {
	svcCtx   *svc.ServiceContext
	notifier *shadowv2.ShadowNotifier
}

func NewShadowV2Handler(svcCtx *svc.ServiceContext) *ShadowV2Handler {
	notifier := shadowv2.NewShadowNotifier(func(deviceSN string, data interface{}) {
		logic.NotifyDeviceStatusChange(deviceSN, data)
	})

	return &ShadowV2Handler{
		svcCtx:   svcCtx,
		notifier: notifier,
	}
}

func (h *ShadowV2Handler) InitShadow(w http.ResponseWriter, r *http.Request) {
	var req shadowv2.InitShadowReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
			"code": 400,
			"msg":  "请求参数格式错误",
			"data": nil,
		})
		return
	}

	logic := shadowv2.NewShadowLogic(r.Context(), h.svcCtx)
	resp, err := logic.InitDeviceShadow(req.DeviceSN)
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
		"msg":  "操作成功",
		"data": resp,
	})
}

func (h *ShadowV2Handler) UpdateReported(w http.ResponseWriter, r *http.Request) {
	var req shadowv2.UpdateReportedReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
			"code": 400,
			"msg":  "请求参数格式错误",
			"data": nil,
		})
		return
	}

	logic := shadowv2.NewShadowLogicWithNotifier(r.Context(), h.svcCtx, h.notifier)
	resp, err := logic.UpdateReportedState(&req)
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
		"msg":  "上报成功",
		"data": resp,
	})
}

func (h *ShadowV2Handler) UpdateDesired(w http.ResponseWriter, r *http.Request) {
	var req shadowv2.UpdateDesiredReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
			"code": 400,
			"msg":  "请求参数格式错误",
			"data": nil,
		})
		return
	}

	logic := shadowv2.NewShadowLogicWithNotifier(r.Context(), h.svcCtx, h.notifier)
	resp, err := logic.UpdateDesiredState(&req)
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
		"msg":  "下发成功",
		"data": resp,
	})
}

func (h *ShadowV2Handler) QueryShadow(w http.ResponseWriter, r *http.Request) {
	deviceSN := r.URL.Query().Get("device_sn")
	if deviceSN == "" {
		httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
			"code": 400,
			"msg":  "device_sn 参数不能为空",
			"data": nil,
		})
		return
	}

	logic := shadowv2.NewShadowLogic(r.Context(), h.svcCtx)
	resp, err := logic.QueryShadow(deviceSN)
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
		"msg":  "查询成功",
		"data": resp,
	})
}

func (h *ShadowV2Handler) QueryReportedOnly(w http.ResponseWriter, r *http.Request) {
	deviceSN := r.URL.Query().Get("device_sn")
	if deviceSN == "" {
		httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
			"code": 400,
			"msg":  "device_sn 参数不能为空",
			"data": nil,
		})
		return
	}

	logic := shadowv2.NewShadowLogic(r.Context(), h.svcCtx)
	reported, version, err := logic.QueryReportedOnly(deviceSN)
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
		"msg":  "查询成功",
		"data": map[string]interface{}{
			"device_sn": deviceSN,
			"reported":  reported,
			"version":   version,
		},
	})
}

func (h *ShadowV2Handler) QueryVersionOnly(w http.ResponseWriter, r *http.Request) {
	deviceSN := r.URL.Query().Get("device_sn")
	if deviceSN == "" {
		httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
			"code": 400,
			"msg":  "device_sn 参数不能为空",
			"data": nil,
		})
		return
	}

	logic := shadowv2.NewShadowLogic(r.Context(), h.svcCtx)
	version, err := logic.QueryVersionOnly(deviceSN)
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
		"msg":  "查询成功",
		"data": map[string]interface{}{
			"device_sn": deviceSN,
			"version":   version,
		},
	})
}

func (h *ShadowV2Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	var req shadowv2.UpdateStatusReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
			"code": 400,
			"msg":  "请求参数格式错误",
			"data": nil,
		})
		return
	}

	logic := shadowv2.NewShadowLogicWithNotifier(r.Context(), h.svcCtx, h.notifier)
	resp, err := logic.UpdateOnlineStatus(&req)
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
		"msg":  "更新成功",
		"data": resp,
	})
}

func (h *ShadowV2Handler) CASUpdate(w http.ResponseWriter, r *http.Request) {
	var req shadowv2.CASUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
			"code": 400,
			"msg":  "请求参数格式错误",
			"data": nil,
		})
		return
	}

	logic := shadowv2.NewShadowLogic(r.Context(), h.svcCtx)
	resp, err := logic.CASUpdateWithCheck(&req)
	if err != nil {
		httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
			"code": 400,
			"msg":  err.Error(),
			"data": nil,
		})
		return
	}

	statusCode := http.StatusOK
	if !resp.Success {
		statusCode = http.StatusConflict
	}

	httpx.WriteJson(w, statusCode, map[string]interface{}{
		"code": 200,
		"msg":  resp.Message,
		"data": resp,
	})
}
