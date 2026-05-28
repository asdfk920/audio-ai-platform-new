package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/rest/httpx"

	shadowv2 "github.com/jacklau/audio-ai-platform/services/device/internal/device/shadowv2"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

type AdminShadowHandler struct {
	svcCtx *svc.ServiceContext
}

func NewAdminShadowHandler(svcCtx *svc.ServiceContext) *AdminShadowHandler {
	return &AdminShadowHandler{
		svcCtx: svcCtx,
	}
}

// GetDeviceShadowDetail 获取设备影子完整详情（Admin 后台专用）
// GET /api/admin/device/shadow/detail?device_sn=xxx
func (h *AdminShadowHandler) GetDeviceShadowDetail(w http.ResponseWriter, r *http.Request) {
	deviceSN := r.URL.Query().Get("device_sn")
	if deviceSN == "" {
		httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"success": false,
			"message": "device_sn 参数不能为空",
			"data":    nil,
		})
		return
	}

	deviceSN = trimSpace(deviceSN)

	store := h.svcCtx.GetRedisShadowStore()
	if store == nil {
		httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"success": false,
			"message": "Redis 存储服务未初始化",
			"data":    nil,
		})
		return
	}

	shadow, err := store.GetShadow(r.Context(), deviceSN)
	if err != nil {
		if err.Error() == "shadow not found" {
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"success": false,
				"message": "设备影子不存在",
				"data": map[string]interface{}{
					"device_sn": deviceSN,
					"hint":      "设备可能尚未上线或影子未被初始化",
				},
			})
			return
		}

		httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"success": false,
			"message": "查询设备影子失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	responseData := buildAdminShadowResponse(shadow)

	httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"success": true,
		"message": "查询成功",
		"data":    responseData,
	})
}

// ListDeviceShadows 批量查询设备影子列表（Admin 后台专用）
// POST /api/admin/device/shadow/list
// Request body: {"device_sns": ["sn1", "sn2", ...]}
func (h *AdminShadowHandler) ListDeviceShadows(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceSNs []string `json:"device_sns"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"success": false,
			"message": "请求参数格式错误",
			"data":    nil,
		})
		return
	}

	if len(req.DeviceSNs) == 0 {
		httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"success": false,
			"message": "device_sns 列表不能为空",
			"data":    nil,
		})
		return
	}

	if len(req.DeviceSNs) > 50 {
		httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"success": false,
			"message": "单次查询数量不能超过 50 个设备",
			"data":    nil,
		})
		return
	}

	store := h.svcCtx.GetRedisShadowStore()
	if store == nil {
		httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"success": false,
			"message": "Redis 存储服务未初始化",
			"data":    nil,
		})
		return
	}

	results := make([]map[string]interface{}, 0)
	notFound := make([]string, 0)

	for _, sn := range req.DeviceSNs {
		sn = trimSpace(sn)
		if sn == "" {
			continue
		}

		shadow, err := store.GetShadow(r.Context(), sn)
		if err != nil {
			notFound = append(notFound, sn)
			results = append(results, map[string]interface{}{
				"device_sn": sn,
				"found":     false,
			})
		} else {
			results = append(results, buildAdminShadowResponse(shadow))
		}
	}

	httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"success": true,
		"message": "查询成功",
		"data": map[string]interface{}{
			"total":       len(results),
			"found_count": len(results) - len(notFound),
			"not_found":   notFound,
			"items":       results,
		},
	})
}

// GetDeviceShadowStats 获取设备影子统计信息（Admin 后台专用）
// GET /api/admin/device/shadow/stats
func (h *AdminShadowHandler) GetDeviceShadowStats(w http.ResponseWriter, r *http.Request) {
	store := h.svcCtx.GetRedisShadowStore()
	if store == nil {
		httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"success": false,
			"message": "Redis 存储服务未初始化",
			"data":    nil,
		})
		return
	}

	ctx := r.Context()

	onlineCount, _ := store.CountByStatus(ctx, shadowv2.StatusOnline)
	offlineCount, _ := store.CountByStatus(ctx, shadowv2.StatusOffline)
	abnormalCount, _ := store.CountByStatus(ctx, shadowv2.StatusAbnormal)
	totalCount, _ := store.CountTotal(ctx)

	stats := map[string]interface{}{
		"total_devices":    totalCount,
		"online_devices":   onlineCount,
		"offline_devices":  offlineCount,
		"abnormal_devices": abnormalCount,
		"online_rate":      calculateRate(onlineCount, totalCount),
		"timestamp":        getCurrentTimestamp(),
	}

	httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"success": true,
		"message": "查询成功",
		"data":    stats,
	})
}

// DeleteDeviceShadow 删除设备影子（Admin 后台专用，谨慎使用）
// DELETE /api/admin/device/shadow?device_sn=xxx
func (h *AdminShadowHandler) DeleteDeviceShadow(w http.ResponseWriter, r *http.Request) {
	deviceSN := r.URL.Query().Get("device_sn")
	if deviceSN == "" {
		httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
			"code":    400,
			"success": false,
			"message": "device_sn 参数不能为空",
			"data":    nil,
		})
		return
	}

	deviceSN = trimSpace(deviceSN)

	store := h.svcCtx.GetRedisShadowStore()
	if store == nil {
		httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"success": false,
			"message": "Redis 存储服务未初始化",
			"data":    nil,
		})
		return
	}

	err := store.DeleteShadow(r.Context(), deviceSN)
	if err != nil {
		httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
			"code":    500,
			"success": false,
			"message": "删除失败: " + err.Error(),
			"data":    nil,
		})
		return
	}

	httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
		"code":    200,
		"success": true,
		"message": "删除成功",
		"data": map[string]interface{}{
			"device_sn": deviceSN,
			"deleted":   true,
		},
	})
}

// buildAdminShadowResponse 构建管理员专用的影子响应数据
func buildAdminShadowResponse(shadow *shadowv2.DeviceShadow) map[string]interface{} {
	return map[string]interface{}{
		"device_sn":             shadow.DeviceSN,
		"found":                 true,
		"reported":              shadow.Reported,
		"desired":               shadow.Desired,
		"version":               shadow.Version,
		"status":                string(shadow.Status),
		"status_text":           getStatusText(shadow.Status),
		"update_time":           shadow.UpdateTime,
		"update_time_formatted": formatTimestamp(shadow.UpdateTime),
		"metadata":              map[string]interface{}{},
		"is_online":             shadow.Status == shadowv2.StatusOnline,
		"reported_fields_count": len(shadow.Reported),
		"desired_fields_count":  len(shadow.Desired),
	}
}

// getStatusText 获取状态文本描述
func getStatusText(status shadowv2.DeviceStatus) string {
	switch status {
	case shadowv2.StatusOnline:
		return "在线"
	case shadowv2.StatusOffline:
		return "离线"
	case shadowv2.StatusAbnormal:
		return "异常"
	default:
		return "未知"
	}
}

// calculateRate 计算百分比
func calculateRate(count, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(count) / float64(total) * 100
}

// formatTimestamp 格式化时间戳
func formatTimestamp(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}
	// 简单的 ISO 格式化，实际项目中可以使用更完善的格式化
	return strconv.FormatInt(timestamp, 10)
}

// getCurrentTimestamp 获取当前时间戳
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// trimSpace 去除空格
func trimSpace(s string) string {
	return strings.TrimSpace(s)
}
