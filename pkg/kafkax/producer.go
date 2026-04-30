package kafkax

import (
	"context"
	"strings"

	"github.com/segmentio/kafka-go"
)

// Producer 轻量封装，供各服务发布领域事件（设备状态、审计、OTA 等）。
type Producer struct {
	w *kafka.Writer
}

// NewProducer Brokers 为空时返回 (nil, nil)，调用方按未启用消息总线处理。
func NewProducer(cfg Config) (*Producer, error) {
	var addrs []string
	for _, b := range cfg.Brokers {
		b = strings.TrimSpace(b)
		if b != "" {
			addrs = append(addrs, b)
		}
	}
	if len(addrs) == 0 {
		return nil, nil
	}
	return &Producer{
		w: &kafka.Writer{
			Addr:                   kafka.TCP(addrs...),
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
		},
	}, nil
}

// Close 释放连接；nil Producer 安全。
func (p *Producer) Close() error {
	if p == nil || p.w == nil {
		return nil
	}
	return p.w.Close()
}

// Publish 写入指定主题；nil Producer 或未配置时为 no-op。
func (p *Producer) Publish(ctx context.Context, topic string, key, value []byte) error {
	if p == nil || p.w == nil || topic == "" {
		return nil
	}
	return p.w.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	})
}
