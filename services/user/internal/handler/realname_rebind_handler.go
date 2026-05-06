// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// @RealnameRebind 实名换绑
// @Summary      实名换绑
// @Description  实名认证用户更换绑定的邮箱或手机号
// @Tags         账户绑定
// @Accept       json
// @Produce      json
// @Param        body  body      errorx.Response  true  "请求参数"
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/rebind/realname [put]
// @Security     BearerAuth
// RealnameRebindHandler 实名认证换绑处理器
// POST /api/v1/user/realname/rebind
// 用途：用户实名认证信息换绑（更换身份证信息）
func RealnameRebindHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RealnameRebindReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewRealnameRebindLogic(r.Context(), svcCtx)
		_, err := l.RealnameRebind(&req)
		httpx.ErrorCtx(r.Context(), w, err)
	}
}
