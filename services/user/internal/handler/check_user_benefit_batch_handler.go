package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/ctxuser"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/member/benefitcheck"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// CheckUserBenefitBatchHandler 批量用户权益校验处理器
// POST /api/v1/user/benefit/check_batch
// 用途：批量校验用户是否拥有多项权益，支持频率限制
func CheckUserBenefitBatchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid := ctxuser.ParseUserID(r.Context())
		if uid <= 0 {
			httpx.ErrorCtx(r.Context(), w, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录"))
			return
		}
		if err := benefitcheck.EnsureUserRateLimit(r.Context(), svcCtx.Config, uid); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		var req types.CheckUserBenefitBatchReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := logic.NewCheckUserBenefitBatchLogic(r.Context(), svcCtx)
		resp, err := l.CheckUserBenefitBatch(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.WriteJsonCtx(r.Context(), w, http.StatusOK, errorx.SuccessData("校验成功", resp))
	}
}
