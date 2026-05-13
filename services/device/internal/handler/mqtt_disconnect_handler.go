package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

// MqttDisconnectHandler MQTT 设备断开连接处理器
// POST /mqtt/disconnect
// 用途：EMQX Broker 在设备断开连接时调用，更新设备离线状态
// 触发时机：设备主动断开或异常断开时
func MqttDisconnectHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"Method Not Allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			ClientID string `json:"clientid"` // 设备客户端ID（= SN）
			Reason   string `json:"reason"`   // 断开原因
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logx.Errorf("MQTT断开事件：解析请求失败：%v", err)
			writeMqttDisconnectResult(w, false, "请求参数格式错误")
			return
		}

		logx.Infof("MQTT断开事件：clientid=%s, reason=%s", req.ClientID, req.Reason)

		l := logic.NewMqttAuthLogic(svcCtx)
		err := l.HandleDisconnect(req.ClientID)

		if err != nil {
			logx.Errorf("更新设备离线状态失败：clientid=%s, err=%v", req.ClientID, err)
			writeMqttDisconnectResult(w, false, err.Error())
			return
		}

		logx.Infof("MQTT断开处理成功：clientid=%s", req.ClientID)
		writeMqttDisconnectResult(w, true, "")
	}
}

// writeMqttDisconnectResult 写入断开事件处理结果
func writeMqttDisconnectResult(w http.ResponseWriter, success bool, errorMsg string) {
	resp := map[string]interface{}{
		"result": "ok",
	}

	if !success && errorMsg != "" {
		resp["error"] = errorMsg
		resp["result"] = "error"
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
