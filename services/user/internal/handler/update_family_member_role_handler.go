package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// @UpdateFamilyMemberRole 更新成员角色
// @Summary      更新成员角色
// @Description  修改家庭成员的角色
// @Tags         家庭管理
// @Accept       json
// @Produce      json
// @Param        body  body      errorx.Response  true  "请求参数"
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/family/member/role/update [post]
// @Security     BearerAuth
// UpdateFamilyMemberRoleHandler 更新家庭成员角色处理器
// POST /api/v1/user/family/member/role/update
// 用途：家庭管理员更新指定成员的角色（普通成员/管理员）
func UpdateFamilyMemberRoleHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.FamilyMemberRoleUpdateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		if err := logic.NewUpdateFamilyMemberRoleLogic(r.Context(), svcCtx).UpdateFamilyMemberRole(&req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.SuccessMsg("成员角色更新成功"))
	}
}
