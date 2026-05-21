package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/httpresp"
	"github.com/jacklau/audio-ai-platform/services/content/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/content/internal/pkg/util/auth"
	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
)

// contentDetailHandler 内容详情处理器
// GET /api/v1/content/:id
// 必须登录：需要用户 Token 验证权限和点赞状态
func contentDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpresp.Write(w, http.StatusMethodNotAllowed, httpresp.MsgMethodNotAllowed+`：仅支持 GET`, nil)
			return
		}

		// 从 URL 路径提取内容 ID
		path := r.URL.Path
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) < 3 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `缺少内容 ID 参数`), nil)
			return
		}

		contentID, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
		if err != nil || contentID <= 0 {
			httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, `内容 ID 格式无效`), nil)
			return
		}

		// 解析 Authorization Header 获取用户信息（必须登录）
		bearerCtx := auth.ParseBearer(r, svcCtx.Config.Auth.AccessSecret)
		if bearerCtx.UserID <= 0 {
			httpresp.Write(w, http.StatusUnauthorized, httpresp.MsgUnauthorized, nil)
			return
		}

		l := logic.NewContentDetailLogic(r.Context(), svcCtx)
		resp, err := l.ContentDetail(contentID, bearerCtx.UserID)
		if err != nil {
			if strings.Contains(err.Error(), "不存在") || strings.Contains(err.Error(), "尚未开放") || strings.Contains(err.Error(), "已过期") {
				httpresp.Write(w, http.StatusNotFound, httpresp.WithDetail(httpresp.MsgNotFound, err.Error()), nil)
			} else {
				httpresp.Write(w, http.StatusBadRequest, httpresp.WithDetail(httpresp.MsgBadRequest, err.Error()), nil)
			}
			return
		}

		httpresp.WriteSuccess(w, resp)
	}
}
