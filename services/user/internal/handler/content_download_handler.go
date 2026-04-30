package handler

import (
	"net/http"
	"strconv"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// contentDownloadHandler 内容流式下载处理器
// GET /api/v1/user/content/download/stream
// 用途：流式传输私有格式内容，支持边下边播和异步保存本地
func contentDownloadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
				"msg":  "content_id 参数必填",
				"data": nil,
			})
			return
		}

		contentID, err := strconv.ParseInt(contentIDStr, 10, 64)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "content_id 格式无效",
				"data": nil,
			})
			return
		}

		userID := ctxuser.ParseUserID(r.Context())
		if userID == 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code": 401,
				"msg":  "未登录",
				"data": nil,
			})
			return
		}

		l := logic.NewContentDownloadLogic(r.Context(), svcCtx)
		err = l.DownloadContent(userID, contentID, w, r)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code": 500,
				"msg":  err.Error(),
				"data": nil,
			})
			return
		}
	}
}
