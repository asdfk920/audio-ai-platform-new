// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// @GetRealNameStatus 查询实名状态
// @Summary      查询实名状态
// @Description  查询当前用户的实名认证状态
// @Tags         实名认证
// @Accept       json
// @Produce      json
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/realname/status [get]
// @Security     BearerAuth
// GetRealNameStatusHandler 获取实名认证状态处理器
// GET /api/v1/user/realname/status
// 用途：查询当前用户的实名认证状态（未认证/审核中/已通过/已拒绝）
func GetRealNameStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewGetRealNameStatusLogic(r.Context(), svcCtx)
		resp, err := l.Get()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
