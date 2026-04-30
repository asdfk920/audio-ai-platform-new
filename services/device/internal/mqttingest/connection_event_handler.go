package mqttingest

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jacklau/audio-ai-platform/services/device/internal/config"
	"github.com/jacklau/audio-ai-platform/services/device/internal/device/shadow"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

// ConnectionEventHandler MQTT 设备连接/断开事件处理器
// 订阅 MQTT Broker 的系统主题，监听设备连接和断开事件，自动更新设备在线状态
type ConnectionEventHandler struct {
	Log logx.Logger
	DB  *sql.DB
	Rdb *redis.Client
	Cfg config.Config
}

// NewConnectionEventHandler 创建连接事件处理器
func NewConnectionEventHandler(db *sql.DB, rdb *redis.Client, c config.Config) *ConnectionEventHandler {
	return &ConnectionEventHandler{
		Log: logx.WithContext(context.Background()),
		DB:  db,
		Rdb: rdb,
		Cfg: c,
	}
}

// OnConnect 处理设备连接事件
// 当设备连接到 MQTT Broker 时触发，将设备标记为在线
func (h *ConnectionEventHandler) OnConnect(_ mqtt.Client, m mqtt.Message) {
	if h == nil || m == nil {
		return
	}

	sn := parseSNFromConnectionTopic(m.Topic())
	if sn == "" {
		h.Log.Infof("mqtt connect event ignore topic=%q bad sn", m.Topic())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.setDeviceOnline(ctx, sn); err != nil {
		h.Log.Errorf("mqtt connect event sn=%s set online failed: %v", sn, err)
		return
	}

	h.Log.Infof("mqtt connect event sn=%s device online", sn)
}

// OnDisconnect 处理设备断开连接事件
// 当设备断开 MQTT 连接时触发，将设备标记为离线
func (h *ConnectionEventHandler) OnDisconnect(_ mqtt.Client, m mqtt.Message) {
	if h == nil || m == nil {
		return
	}

	sn := parseSNFromConnectionTopic(m.Topic())
	if sn == "" {
		h.Log.Infof("mqtt disconnect event ignore topic=%q bad sn", m.Topic())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.setDeviceOffline(ctx, sn); err != nil {
		h.Log.Errorf("mqtt disconnect event sn=%s set offline failed: %v", sn, err)
		return
	}

	h.Log.Infof("mqtt disconnect event sn=%s device offline", sn)
}

// setDeviceOnline 将设备标记为在线
// 更新 Redis 在线键和影子 Hash
func (h *ConnectionEventHandler) setDeviceOnline(ctx context.Context, sn string) error {
	if h.Rdb == nil {
		return nil
	}

	ttl := time.Duration(h.Cfg.DeviceShadow.HeartbeatTTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 300 * time.Second
	}

	okKey := shadow.OnlineKey(sn)
	sk := shadow.ShadowKey(sn)

	pipe := h.Rdb.Pipeline()
	pipe.Set(ctx, okKey, "1", ttl)
	pipe.HSet(ctx, sk, map[string]interface{}{
		shadow.FOnline:       "1",
		shadow.FLastActiveMs: time.Now().UnixMilli(),
	})
	pipe.Expire(ctx, sk, ttl)

	if h.Cfg.DeviceShadow.EnableOnlineSet {
		pipe.SAdd(ctx, shadow.KeyOnlineAll, sn)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// setDeviceOffline 将设备标记为离线
// 更新 Redis 在线键和影子 Hash
func (h *ConnectionEventHandler) setDeviceOffline(ctx context.Context, sn string) error {
	if h.Rdb == nil {
		return nil
	}

	ttl := time.Duration(h.Cfg.DeviceShadow.HeartbeatTTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 300 * time.Second
	}

	okKey := shadow.OnlineKey(sn)
	sk := shadow.ShadowKey(sn)

	pipe := h.Rdb.Pipeline()
	pipe.Set(ctx, okKey, "0", ttl)
	pipe.HSet(ctx, sk, map[string]interface{}{
		shadow.FOnline:       "0",
		shadow.FLastActiveMs: time.Now().UnixMilli(),
	})
	pipe.Expire(ctx, sk, ttl)

	if h.Cfg.DeviceShadow.EnableOnlineSet {
		pipe.SRem(ctx, shadow.KeyOnlineAll, sn)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// parseSNFromConnectionTopic 从连接事件主题中提取设备 SN
// 支持主题格式：
//   - $SYS/brokers/{node}/clients/{clientid}/connected
//   - $SYS/brokers/{node}/clients/{clientid}/disconnected
//   - $events/client_connected
//   - $events/client_disconnected
func parseSNFromConnectionTopic(topic string) string {
	topic = strings.TrimSpace(topic)

	// EMQX 格式: $SYS/brokers/{node}/clients/{clientid}/connected
	if strings.HasPrefix(topic, "$SYS/brokers/") {
		parts := strings.Split(topic, "/")
		if len(parts) >= 5 {
			return strings.TrimSpace(parts[4])
		}
	}

	// EMQX 5.x 格式: $events/client_connected / $events/client_disconnected
	// 消息体中包含 clientid
	if strings.HasPrefix(topic, "$events/client_") {
		return "" // SN 需要从消息体中解析
	}

	return ""
}

// ParseSNFromEventPayload 从连接事件消息体中解析 SN
// EMQX 5.x $events 主题的消息体格式
func ParseSNFromEventPayload(payload []byte) string {
	var event struct {
		ClientID string `json:"clientid"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return ""
	}
	return strings.TrimSpace(event.ClientID)
}
