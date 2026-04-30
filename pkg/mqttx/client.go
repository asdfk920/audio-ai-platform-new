package mqttx

import (
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client 薄封装；Broker 未配置时 NewClient 返回 (nil, nil)，方法均为 no-op。
type Client struct {
	c    mqtt.Client
	mu   sync.RWMutex
	subs map[string]subscription
}

type subscription struct {
	qos     byte
	handler mqtt.MessageHandler
}

// NewClient 建立连接；Broker 为空返回 (nil, nil)。
func NewClient(cfg Config) (*Client, error) {
	b := strings.TrimSpace(cfg.Broker)
	if b == "" {
		return nil, nil
	}
	cid := strings.TrimSpace(cfg.ClientID)
	if cid == "" {
		cid = "audio-ai-platform"
	}
	wrapper := &Client{
		subs: make(map[string]subscription),
	}
	opts := mqtt.NewClientOptions().AddBroker(b).SetClientID(cid)
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		wrapper.resubscribeAll(client)
	})

	c := mqtt.NewClient(opts)
	wrapper.c = c
	token := c.Connect()
	token.WaitTimeout(15 * time.Second)
	if token.Error() != nil {
		return nil, token.Error()
	}
	return wrapper, nil
}

// Disconnect 断开；nil Client 安全。
func (c *Client) Disconnect() {
	if c == nil || c.c == nil {
		return
	}
	c.c.Disconnect(250)
}

// Publish QoS0 发布；nil Client 为 no-op。
func (c *Client) Publish(topic string, qos byte, retained bool, payload []byte) error {
	if c == nil || c.c == nil || topic == "" {
		return nil
	}
	t := c.c.Publish(topic, qos, retained, payload)
	t.WaitTimeout(5 * time.Second)
	return t.Error()
}

// Subscribe 订阅；handler 在库内异步回调；nil Client 为 no-op。
func (c *Client) Subscribe(topic string, qos byte, handler mqtt.MessageHandler) error {
	if c == nil || c.c == nil || topic == "" {
		return nil
	}
	c.mu.Lock()
	c.subs[topic] = subscription{
		qos:     qos,
		handler: handler,
	}
	c.mu.Unlock()
	t := c.c.Subscribe(topic, qos, handler)
	t.WaitTimeout(5 * time.Second)
	return t.Error()
}

func (c *Client) resubscribeAll(client mqtt.Client) {
	if c == nil || client == nil {
		return
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for topic, sub := range c.subs {
		if strings.TrimSpace(topic) == "" {
			continue
		}
		token := client.Subscribe(topic, sub.qos, sub.handler)
		token.WaitTimeout(5 * time.Second)
	}
}
