package mqttadmin

import mqtt "github.com/eclipse/paho.mqtt.golang"

// Client 返回全局 MQTT 客户端；未配置 broker 时为 nil，业务侧已做 nil 判断。
func Client() mqtt.Client {
	return globalClient
}

var globalClient mqtt.Client

// SetClient 供启动阶段注入（可选）。
func SetClient(c mqtt.Client) {
	globalClient = c
}
