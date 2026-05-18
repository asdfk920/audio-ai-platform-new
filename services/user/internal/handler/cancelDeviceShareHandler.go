package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func cancelDeviceShareHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeviceShareCancelReq

		// 不使用 httpx.Parse：go-zero 在 Content-Length==0 或非典型 Content-Type 时会跳过 JSON，
		// 导致 sn/share_id 解析不到；ParseForm 等步骤也可能返回非 CodeError，最终落到 9004。
		const maxBody = 1 << 20
		raw, errRead := io.ReadAll(io.LimitReader(r.Body, maxBody))
		if errRead != nil {
			httpx.ErrorCtx(r.Context(), w, errorx.NewCodeError(errorx.CodeInvalidParam, "读取请求体失败"))
			return
		}
		if len(strings.TrimSpace(string(raw))) > 0 {
			if err := json.Unmarshal(raw, &req); err != nil {
				httpx.ErrorCtx(r.Context(), w, errorx.NewCodeError(errorx.CodeInvalidParam, "请求体不是合法 JSON"))
				return
			}
		}

		l := logic.NewCancelDeviceShareLogic(r.Context(), svcCtx)
		err := l.CancelDeviceShare(&req)
		if err != nil {
			var ce *errorx.CodeError
			if !errors.As(err, &ce) {
				err = errorx.NewCodeError(errorx.CodeInternalError, "系统繁忙，请稍后重试")
			}
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, map[string]interface{}{
			"code": 0,
			"msg":  "用户撤销设备成功",
		})
	}
}
