package handler

import (
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

// DeviceShadowDetailHandler 设备影子详情查询（用于详情页展示）
// GET /api/device/shadow/detail?device_sn=xxx
//
// 用途：在设备详情页的"状态上报"标签中展示完整的设备影子信息
//
// 特点：
//  - 返回格式化的属性列表，便于前端表格展示
//  - 包含属性标签、类型、分类等元数据
//  - 标记与期望状态不同的属性
//  - 包含版本信息和时间线数据
//
// 适用场景：
//  - Admin 后台设备详情弹窗
//  - 用户 App 设备详情页面
//  - 运维监控大屏
func DeviceShadowDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpx.WriteJson(w, http.StatusMethodNotAllowed, map[string]interface{}{
				"code":    405,
				"success": false,
				"message": "仅支持 GET 方法",
				"data":    nil,
			})
			return
		}

		deviceSN := r.URL.Query().Get("device_sn")
		if deviceSN == "" {
			deviceSN = r.URL.Query().Get("sn") // 兼容旧参数名
		}

		if strings.TrimSpace(deviceSN) == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"success": false,
				"message": "device_sn 参数不能为空",
				"data": map[string]interface{}{
					"hint": "请提供要查询的设备序列号",
				},
			})
			return
		}

		l := logic.NewDeviceShadowDetailLogic(r.Context(), svcCtx)
		resp, err := l.GetDeviceShadowDetail(deviceSN)
		if err != nil {
			httpx.Error(w, err)
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"success": true,
			"message": "查询成功",
			"data":    resp,
		})
	})
}
