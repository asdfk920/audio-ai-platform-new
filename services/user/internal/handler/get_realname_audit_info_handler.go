// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// @GetRealNameAuditInfo 查询审核详情
// @Summary      查询审核详情
// @Description  查询实名认证的审核详情，包括拒绝原因等
// @Tags         实名认证
// @Accept       json
// @Produce      json
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/realname/audit [get]
// @Security     BearerAuth
// GetRealNameAuditInfoHandler 获取实名认证审核信息处理器
// GET /api/v1/user/realname/audit-info
// 用途：查询当前用户实名认证的审核详细信息（拒绝原因等）
func GetRealNameAuditInfoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewGetRealNameAuditInfoLogic(r.Context(), svcCtx)
		resp, err := l.Get()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
