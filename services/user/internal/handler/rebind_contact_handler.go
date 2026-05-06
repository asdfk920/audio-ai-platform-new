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

// @RebindContact 换绑联系方式
// @Summary      换绑联系方式
// @Description  更换已绑定的邮箱或手机号
// @Tags         账户绑定
// @Accept       json
// @Produce      json
// @Param        body  body      errorx.Response  true  "请求参数"
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/rebind [put]
// @Security     BearerAuth
// RebindContactHandler 换绑联系方式处理器
// POST /api/v1/user/contact/rebind
// 用途：用户换绑手机号或邮箱，需旧验证码和新验证码双重验证
func RebindContactHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RebindContactReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewRebindContactLogic(r.Context(), svcCtx)
		resp, err := l.RebindContact(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
