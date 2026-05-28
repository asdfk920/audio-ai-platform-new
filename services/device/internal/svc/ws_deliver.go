package svc

// WsDeliverResult 设备 WebSocket 投递结果（本机直写 / Redis relay）
type WsDeliverResult struct {
	LocalWriteOk   bool // 本进程 deviceConnMap 直写成功
	RelayPublishOk bool // 已发布到 Redis Pub/Sub，由在线实例消费
}

// WsDeliverFunc 向设备 WS 投递 JSON（deviceKey 常为 device_id 字符串，sn 为备用键）
type WsDeliverFunc func(deviceKey, sn string, payload interface{}) (WsDeliverResult, error)
