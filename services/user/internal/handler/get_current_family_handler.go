package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// @GetCurrentFamily 查询家庭组
// @Summary      查询家庭组
// @Description  查询当前用户所属的家庭组详情
// @Tags         家庭管理
// @Accept       json
// @Produce      json
// @Success      200  {object}  errorx.Response  "成功"
// @Failure      400  {object}  errorx.Response  "参数错误"
// @Failure      401  {object}  errorx.Response  "未登录"
// @Failure      500  {object}  errorx.Response  "服务器错误"
// @Router       /user/family/current [get]
// @Security     BearerAuth
// GetCurrentFamilyHandler 获取当前家庭信息处理器
// GET /api/v1/user/family/current
// 用途：查询当前用户所在的家庭信息
func GetCurrentFamilyHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := logic.NewGetCurrentFamilyLogic(r.Context(), svcCtx).GetCurrentFamily()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.Success(resp))
	}
}
