package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// DeviceDownloadHandler 设备下载接口HTTP处理器（完整流程）
// POST /api/v1/device/download
// 用户点击下载按钮，创建记录并下发指令到设备（端口8002）
func DeviceDownloadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.DeviceDownloadReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "请求体格式错误",
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceDownloadLogic(r.Context(), svcCtx)
		resp, err := l.Download(&req)

		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 9001,
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
	})
}

// DeviceDownloadCallbackHandler 设备下载回调处理器（设备→设备服务）
// POST /api/v1/device/download/callback
// 设备下载完成后上报结果，或内部调用更新状态
func DeviceDownloadCallbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.DeviceDownloadCallbackReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "请求体格式错误",
				"data": nil,
			})
			return
		}

		l := logic.NewDeviceDownloadLogic(r.Context(), svcCtx)
		resp, err := l.HandleCallback(&req)

		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 9001,
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
}

// DeviceDownloadStatusHandler 查询设备下载状态处理器
// GET /api/v1/device/download/status?content_id=xxx
func DeviceDownloadStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 GET",
				"data": nil,
			})
			return
		}

		contentIDStr := r.URL.Query().Get("content_id")
		if contentIDStr == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "缺少 content_id 参数",
				"data": nil,
			})
			return
		}

		contentID, parseErr := strconv.ParseInt(contentIDStr, 10, 64)
		if parseErr != nil || contentID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "content_id 必须为正整数",
				"data": nil,
			})
			return
		}

		userID, _ := jwt.GetUserIdFromContext(r.Context())

		l := logic.NewDeviceDownloadLogic(r.Context(), svcCtx)
		resp, err := l.GetDownloadStatus(contentID, userID)

		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 9001,
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
	})
}

// DeviceDownloadListHandler 设备下载历史列表处理器
// GET /api/v1/device/downloads
func DeviceDownloadListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 GET",
				"data": nil,
			})
			return
		}

		var req types.DeviceDownloadListReq

		pageStr := r.URL.Query().Get("page")
		sizeStr := r.URL.Query().Get("page_size")

		if pageStr != "" {
			if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil {
				req.Page = int32(p)
			}
		}
		if sizeStr != "" {
			if s, err := strconv.ParseInt(sizeStr, 10, 32); err == nil {
				req.PageSize = int32(s)
			}
		}

		sn := r.URL.Query().Get("sn")
		status := r.URL.Query().Get("status")

		if sn != "" {
			req.DeviceSN = sn
		}
		if status != "" {
			req.Status = status
		}

		userID, _ := jwt.GetUserIdFromContext(r.Context())

		l := logic.NewDeviceDownloadLogic(r.Context(), svcCtx)
		resp, err := l.GetDownloadList(&req, userID)

		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 9001,
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
	})
}
