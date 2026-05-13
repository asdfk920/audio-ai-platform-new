// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
)

// SendVerifyCodeHandler 发送验证码处理器
// @Summary      发送验证码
// @Description  发送验证码到邮箱或手机，用于注册、登录、找回密码等场景
// @Tags         验证码
// @Accept       json
// @Produce      json
// @Param        body  body      errorx.Response  true  "发送验证码请求"
// @Success      200  {object}  errorx.Response  "发送成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      429  {object}  errorx.Response  "发送过于频繁"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/verify/send [post]
// POST /api/v1/user/verify-code/send
// 用途：向用户邮箱或手机发送验证码，1 分钟有效，1 分钟最多 3 条，超过提示 3 分钟后重试
func SendVerifyCodeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SendVerifyCodeReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewSendVerifyCodeLogic(r.Context(), svcCtx)
		resp, err := l.SendVerifyCode(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.Success(resp))
		}
	}
}
