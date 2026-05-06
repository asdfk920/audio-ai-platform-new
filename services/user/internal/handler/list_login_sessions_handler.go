package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// @listLoginSessions 会话列表
// @Summary      会话列表
// @Description  查询当前用户的所有登录会话信息
// @Tags         登录会话
// @Accept       json
// @Produce      json
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/login/sessions [get]
// listLoginSessionsHandler 查询登录会话列表处理器
// GET /api/v1/user/session/list
// 用途：查询当前用户的所有登录会话（设备列表）
func listLoginSessionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewListLoginSessionsLogic(r.Context(), svcCtx)
		resp, err := l.ListLoginSessions()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
