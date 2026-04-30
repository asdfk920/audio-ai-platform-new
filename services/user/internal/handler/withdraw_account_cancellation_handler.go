package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"

	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/httpmeta"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// WithdrawAccountCancellationHandler 撤销注销申请处理器
// POST /api/v1/user/account/cancellation/withdraw
// 用途：用户在注销冷静期内撤销注销申请，恢复账号正常使用
func WithdrawAccountCancellationHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 创建逻辑处理实例
		l := logic.NewWithdrawAccountCancellationLogic(r.Context(), svcCtx)

		// 获取客户端 IP 地址（用于日志记录）
		clientIP := httpmeta.ClientIP(r)

		// 执行撤销注销申请逻辑
		resp, err := l.Withdraw(clientIP)
		if err != nil {
			// 撤销失败，返回错误
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 返回成功响应
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
