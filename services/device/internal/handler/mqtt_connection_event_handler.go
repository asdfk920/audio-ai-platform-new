package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

func MqttConnectionEventHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"Method Not Allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		var req logic.MqttConnectionEventReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logx.Errorf("MQTT连接事件：解析请求失败: %v", err)
			writeMqttEventResult(w, false, "请求参数格式错误")
			return
		}

		logx.Infof("MQTT连接事件: action=%s, clientid=%s, reason=%s",
			req.Action, req.ClientID, req.Reason)

		l := logic.NewMqttConnectionEventLogic(svcCtx)
		err := l.HandleEvent(&req)

		if err != nil {
			logx.Errorf("MQTT连接事件处理失败: action=%s, clientid=%s, err=%v",
				req.Action, req.ClientID, err)
			writeMqttEventResult(w, false, err.Error())
			return
		}

		logx.Infof("MQTT连接事件处理成功: action=%s, clientid=%s",
			req.Action, req.ClientID)
		writeMqttEventResult(w, true, "")
	}
}

func writeMqttEventResult(w http.ResponseWriter, success bool, errorMsg string) {
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
