package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// @kickLoginSession 吊销会话
// @Summary      吊销会话
// @Description  强制吊销指定登录会话，使该会话立即失效
// @Tags         登录会话
// @Accept       json
// @Produce      json
// @Param        body  body      errorx.Response  true  "请求参数"
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/login/sessions/kick [post]
// kickLoginSessionHandler 踢出登录会话处理器
// POST /api/v1/user/session/kick
// 用途：踢出指定登录会话（强制下线其他设备）
func kickLoginSessionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.KickLoginSessionReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := logic.NewKickLoginSessionLogic(r.Context(), svcCtx)
		resp, err := l.KickLoginSession(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
