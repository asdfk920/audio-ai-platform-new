package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/content/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// contentDownloadDeleteHandler 删除下载记录处理器
// DELETE /api/v1/content/downloads
// 必须登录：只能删除自己的下载记录
func contentDownloadDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 DELETE",
				"data": nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code": 401,
				"msg":  "请先登录",
				"data": nil,
			})
			return
		}

		var req types.DownloadDeleteReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "JSON 解析失败",
				"data": nil,
			})
			return
		}

		l := logic.NewContentDownloadLogic(r.Context(), svcCtx)
		err := l.DeleteDownloadRecords(&req, bearerCtx.UserID)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  err.Error(),
				"data": nil,
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "删除成功",
			"data":    nil,
		})
	}
}

// contentDownloadListHandler 下载历史列表处理器
// GET /api/v1/content/downloads
// 必须登录：查看自己的下载历史
func contentDownloadListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 GET",
				"data": nil,
			})
			return
		}

		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code": 401,
				"msg":  "请先登录",
				"data": nil,
			})
			return
		}

		var req types.DownloadListReq
		pageStr := r.URL.Query().Get("page")
		sizeStr := r.URL.Query().Get("size")
		if pageStr != "" {
			if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil {
				req.Page = int32(p)
			}
		}
		if sizeStr != "" {
			if s, err := strconv.ParseInt(sizeStr, 10, 32); err == nil {
				req.Size = int32(s)
			}
		}

		l := logic.NewContentDownloadLogic(r.Context(), svcCtx)
		resp, err := l.GetDownloadList(&req, bearerCtx.UserID)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  err.Error(),
				"data": nil,
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "success",
			"data":    resp,
		})
	}
}
