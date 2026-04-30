package mqttx

// Config MQTT 客户端（设备上报、OTA 进度、影子同步等）。Broker 为空则未启用。
type Config struct {
	Broker   string // 例如 tcp://localhost:1883
	ClientID string
	Username string
	Password string
}
