package handler

import (
	"encoding/json"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/logic"
	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"
)

// MqttAuthHandler EMQX MQTT 设备认证处理器
// POST /mqtt/auth
// 用途：MQTT Broker (EMQX) 调用此接口验证设备连接身份
// 触发时机：设备发起 MQTT CONNECT 连接时
// 调用方：EMQX Broker 的 HTTP 认证插件
//
// 请求格式（EMQX 标准）:
//
//	{
//	  "clientid": "设备SN编号",
//	  "username": "设备SN编号",
//	  "password": "设备专属密钥"
//	}
//
// 响应格式（EMQX 标准）:
// 成功: {"result": "allow"}
// 失败: {"result": "deny", "error_msg": "原因说明"}
func MqttAuthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			writeMqttAuthResult(w, false, "仅支持 POST 方法")
			return
		}

		var req struct {
			ClientID string `json:"clientid"` // 设备客户端ID（= SN）
			Username string `json:"username"` // 用户名（= SN）
			Password string `json:"password"` // 密码（= 设备密钥）
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logx.Errorf("MQTT认证：解析请求失败：%v", err)
			writeMqttAuthResult(w, false, "请求参数格式错误")
			return
		}

		logx.Infof("MQTT认证请求：clientid=%s, username=%s", req.ClientID, req.Username)

		l := logic.NewMqttAuthLogic(svcCtx)
		result, errMsg := l.Authenticate(req.ClientID, req.Username, req.Password)

		if result {
			logx.Infof("MQTT认证成功：clientid=%s", req.ClientID)
			writeMqttAuthResult(w, true, "")
		} else {
			logx.Errorf("MQTT认证失败：clientid=%s, 原因=%s", req.ClientID, errMsg)
			writeMqttAuthResult(w, false, errMsg)
		}
	}
}

// writeMqttAuthResult 写入 EMQX 标准格式的认证结果
func writeMqttAuthResult(w http.ResponseWriter, allow bool, errorMsg string) {
	resp := map[string]interface{}{
		"result": "deny",
	}

	if allow {
		resp["result"] = "allow"
	} else if errorMsg != "" {
		resp["error_msg"] = errorMsg
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
