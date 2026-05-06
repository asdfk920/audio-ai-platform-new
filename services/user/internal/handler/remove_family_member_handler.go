package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// @RemoveFamilyMember 移除成员
// @Summary      移除成员
// @Description  从家庭组中移除成员
// @Tags         家庭管理
// @Accept       json
// @Produce      json
// @Param        body  body      errorx.Response  true  "请求参数"
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/family/member/remove [post]
// @Security     BearerAuth
// RemoveFamilyMemberHandler 移除家庭成员处理器
// POST /api/v1/user/family/member/remove
// 用途：家庭管理员移除指定家庭成员
func RemoveFamilyMemberHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.FamilyMemberRemoveReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		if err := logic.NewRemoveFamilyMemberLogic(r.Context(), svcCtx).RemoveFamilyMember(&req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.SuccessMsg("移除家庭成员成功"))
	}
}
