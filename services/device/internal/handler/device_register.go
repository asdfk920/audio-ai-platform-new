package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
)

// deviceRegisterHandler 设备注册处理器
// POST /api/device/register
// 安全设计：
// - 设备携带SN + 时间戳 + 签名发起注册
// - 签名由设备密钥对SN+时间戳进行HMAC-SHA256计算
// - 全程不传输密钥明文，防止泄露
// - 时间戳校验防止重放攻击
func deviceRegisterHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		var req types.DeviceRegisterReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewDeviceRegisterLogic(r.Context(), svcCtx)
		resp, err := l.DeviceRegister(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
