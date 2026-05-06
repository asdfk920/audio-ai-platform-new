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

// BindContactHandler 绑定联系方式处理器
// @Summary      绑定联系方式
// @Description  为当前登录用户绑定邮箱或手机号
// @Tags         账户绑定
// @Accept       json
// @Produce      json
// @Param        body  body      errorx.Response  true  "绑定联系方式请求"
// @Success      200  {object}  errorx.Response  "绑定成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/bind [put]
// @Security     BearerAuth
// POST /api/v1/user/contact/bind
// 用途：用户绑定手机号或邮箱，需验证码验证
func BindContactHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.BindContactReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewBindContactLogic(r.Context(), svcCtx)
		resp, err := l.BindContact(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
