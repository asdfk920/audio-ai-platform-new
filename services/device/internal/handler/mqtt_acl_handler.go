package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

func MqttACLHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			writeMqttACLResult(w, "deny")
			return
		}

		var req logic.MqttACLReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logx.Errorf("MQTT ACL: 解析请求失败: %v", err)
			writeMqttACLResult(w, "deny")
			return
		}

		l := logic.NewMqttACLLogic(svcCtx)
		resp, err := l.CheckPermission(&req)
		if err != nil {
			logx.Errorf("MQTT ACL: 鉴权失败 clientid=%s, topic=%s, action=%s, err=%v",
				req.ClientID, req.Topic, req.Action, err)
			writeMqttACLResult(w, "deny")
			return
		}

		writeMqttACLResult(w, resp.Result)
	}
}

func writeMqttACLResult(w http.ResponseWriter, result string) {
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"result": result,
	})
}
