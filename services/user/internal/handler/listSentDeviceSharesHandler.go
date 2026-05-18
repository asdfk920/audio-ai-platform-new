// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"errors"
	"net/http"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 查询我发出的共享
func listSentDeviceSharesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewListSentDeviceSharesLogic(r.Context(), svcCtx)
		resp, err := l.ListSentDeviceShares()
		if err != nil {
			var ce *errorx.CodeError
			if !errors.As(err, &ce) {
				err = errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
			}
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
