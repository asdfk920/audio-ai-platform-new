package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// contentDetailHandler 内容详情处理器
// GET /api/v1/content/:id
// 必须登录：需要用户 Token 验证权限和点赞状态
func contentDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 GET",
				"data": nil,
			})
			return
		}

		// 从 URL 路径提取内容 ID
		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 3 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "缺少内容 ID 参数",
				"data": nil,
			})
			return
		}

		contentID, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
		if err != nil || contentID <= 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "内容 ID 格式无效",
				"data": nil,
			})
			return
		}

		// 解析 Authorization Header 获取用户信息（必须登录）
		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code": 401,
				"msg":  "请先登录",
				"data": nil,
			})
			return
		}

		l := logic.NewContentDetailLogic(r.Context(), svcCtx)
		resp, err := l.ContentDetail(contentID, bearerCtx.UserID)
		if err != nil {
			if strings.Contains(err.Error(), "不存在") || strings.Contains(err.Error(), "尚未开放") || strings.Contains(err.Error(), "已过期") {
				httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
					"code": 404,
					"msg":  err.Error(),
					"data": nil,
				})
			} else {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code": 400,
					"msg":  err.Error(),
					"data": nil,
				})
			}
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"msg":  "获取成功",
			"data": resp,
		})
	}
}
