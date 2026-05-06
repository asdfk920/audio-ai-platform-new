package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// @ListFamilyMembers 成员列表
// @Summary      成员列表
// @Description  查询当前家庭组的所有成员信息
// @Tags         家庭管理
// @Accept       json
// @Produce      json
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/family/member/list [get]
// @Security     BearerAuth
// ListFamilyMembersHandler 查询家庭成员列表处理器
// GET /api/v1/user/family/members/list
// 用途：查询当前家庭的成员列表
func ListFamilyMembersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := logic.NewListFamilyMembersLogic(r.Context(), svcCtx).ListFamilyMembers()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.Success(resp))
	}
}
