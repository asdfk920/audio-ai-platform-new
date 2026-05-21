package logic

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// Redis Pub/Sub 通道：任意实例收到 HTTP 下发的指令后，若本机无 WS 连接，则发布到此通道，
// 由实际持有 WebSocket 的实例消费并 WriteJSON（解决多副本 / 网关未粘滞会话时的「Redis 在线但本机 map 为空」）。
const wsRelayPubSubChannel = "audio_platform:device:ws_push"

var wsRelayRedis *redis.Client

// SetWsRelayRedis 在主程序启动时注入，与设备业务 Redis 复用同一 Client。
func SetWsRelayRedis(c *redis.Client) {
	wsRelayRedis = c
}

func relayPublishWsPush(ctx context.Context, deviceIDStr string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	env := struct {
		DeviceID string          `json:"device_id"`
		Payload  json.RawMessage `json:"payload"`
	}{
		DeviceID: deviceIDStr,
		Payload:  payload,
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return wsRelayRedis.Publish(ctx, wsRelayPubSubChannel, raw).Err()
}

// StartWsRelaySubscriber 在每个 device 实例上启动；收到消息后仅在本机 deviceConnMap 命中时投递。
func StartWsRelaySubscriber(ctx context.Context) {
	if wsRelayRedis == nil {
		return
	}

	sub := wsRelayRedis.Subscribe(ctx, wsRelayPubSubChannel)
	go func() {
		<-ctx.Done()
		_ = sub.Close()
	}()

	logx.Infof("ws relay: subscriber ready channel=%s", wsRelayPubSubChannel)

	for msg := range sub.Channel() {
		if msg == nil {
			continue
		}

		var envelope struct {
			DeviceID string          `json:"device_id"`
			Payload  json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal([]byte(msg.Payload), &envelope); err != nil {
			logx.Errorf("ws relay: unmarshal envelope: %v", err)
			continue
		}
		if envelope.DeviceID == "" {
			continue
		}

		var body map[string]interface{}
		if err := json.Unmarshal(envelope.Payload, &body); err != nil {
			logx.Errorf("ws relay: unmarshal payload device_id=%s: %v", envelope.DeviceID, err)
			continue
		}

		lock.RLock()
		dc, ok := deviceConnMap[envelope.DeviceID]
		lock.RUnlock()
		if !ok {
			continue
		}

		if err := dc.WriteJSON(body); err != nil {
			logx.Errorf("ws relay: WriteJSON failed device_id=%s: %v", envelope.DeviceID, err)
		} else {
			logx.Infof("ws relay: delivered to local WS device_id=%s", envelope.DeviceID)
		}
	}

	logx.Infof("ws relay: subscriber exited")
}
