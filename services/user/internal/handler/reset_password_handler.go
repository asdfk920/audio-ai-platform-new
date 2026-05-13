// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// ResetPasswordHandler 重置密码处理器
// @Summary      重置密码
// @Description  忘记密码时，通过邮箱/手机验证码重置密码
// @Tags         密码管理
// @Accept       json
// @Produce      json
// @Param        body  body      errorx.Response  true  "重置密码请求"
// @Success      200  {object}  errorx.Response  "重置成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/password/reset [post]
// POST /api/v1/user/password/reset
// 用途：用户忘记旧密码时，通过验证码重置密码
func ResetPasswordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ResetPasswordReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewResetPasswordLogic(r.Context(), svcCtx)
		resp, err := l.ResetPassword(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
