package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// RegisterHandler 用户注册处理器
// @Summary      用户注册
// @Description  新用户注册，支持邮箱或手机号注册，需先发验证码
// @Tags         用户认证
// @Accept       json
// @Produce      json
// @Param        body  body      errorx.Response  true  "注册请求"
// @Success      200  {object}  errorx.Response  "注册成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      409  {object}  errorx.Response  "账号已存在"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/register [post]
func RegisterHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RegisterReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewRegisterLogic(r.Context(), svcCtx)
		resp, err := l.Register(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
