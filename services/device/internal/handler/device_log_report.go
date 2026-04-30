package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/device/internal/deviceauthsvc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/pkg/ip"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
	"github.com/jacklau/audio-ai-platform/services/device/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const headerDeviceToken = "X-Device-Token"

// deviceLogReportHandler 设备日志上报处理器
// POST /api/device/log
// 用途：设备通过 HTTP POST 请求上报运行日志到云端
// 认证：通过 X-Device-Token 请求头进行设备身份认证
func deviceLogReportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code": 405,
				"msg":  "仅支持 POST",
				"data": nil,
			})
			return
		}

		// 提取设备 Token
		deviceToken := strings.TrimSpace(r.Header.Get(headerDeviceToken))
		if deviceToken == "" {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code": 401,
				"msg":  "缺少设备认证 Token",
				"data": nil,
			})
			return
		}

		// 解析请求体
		var req types.DeviceLogReportReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "请求参数格式错误",
				"data": nil,
			})
			return
		}

		// 校验 SN 格式
		sn := strings.ToUpper(strings.TrimSpace(req.Sn))
		if sn == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  "设备序列号不能为空",
				"data": nil,
			})
			return
		}

		// 验证设备 Token（查询数据库验证 SN + Token 是否匹配）
		ctx := r.Context()
		deviceInfo, err := svcCtx.DeviceRegister.VerifyToken(ctx, sn, deviceToken)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code": 500,
				"msg":  "设备认证失败",
				"data": nil,
			})
			return
		}
		if deviceInfo == nil {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code": 401,
				"msg":  "设备认证失败：Token 无效或设备未注册",
				"data": nil,
			})
			return
		}

		// 获取客户端 IP
		clientIP := ip.ClientIPTrusted(r, svcCtx.RegisterTrustedNets)
		ctx = deviceauthsvc.WithClientIP(ctx, clientIP)

		// 处理日志上报
		l := logic.NewDeviceLogReportLogic(ctx, svcCtx)
		resp, err := l.DeviceLogReport(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code": 400,
				"msg":  err.Error(),
				"data": nil,
			})
			return
		}

		// 更新设备在线状态
		_ = svcCtx.DeviceRegister.UpdateOnlineStatus(ctx, sn)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200,
			"msg":  "日志接收成功",
			"data": resp,
		})
	}
}
