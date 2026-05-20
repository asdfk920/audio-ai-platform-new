package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zeromicro/go-zero/core/logx"
)

type CommandMessage struct {
	MessageID      string                 `json:"message_id"`
	InstructionID  int64                  `json:"instruction_id"`
	DeviceID       int64                  `json:"device_id"`
	DeviceSN       string                 `json:"device_sn"`
	UserID         int64                  `json:"user_id"`
	CommandCode    string                 `json:"command_code"`
	CommandType    string                 `json:"command_type"`
	Params         map[string]interface{} `json:"params"`
	Priority       int                    `json:"priority"`
	RetryCount     int                    `json:"retry_count"`
	MaxRetry       int                    `json:"max_retry"`
	TimeoutSeconds int                    `json:"timeout_seconds"`
	CreatedAt      time.Time              `json:"created_at"`
	ExpiresAt      time.Time              `json:"expires_at"`
	Operator       string                 `json:"operator"`
	Reason         string                 `json:"reason"`
}

func (m *Manager) PublishCommand(ctx context.Context, msg *CommandMessage) error {
	if !m.IsConnected() {
		return fmt.Errorf("RabbitMQ not connected")
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal command message: %w", err)
	}

	ch, err := m.GetChannel()
	if err != nil {
		return fmt.Errorf("get channel: %w", err)
	}

	routingKey := fmt.Sprintf("cmd.%s", msg.DeviceSN)

	err = ch.PublishWithContext(ctx,
		m.config.Exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    msg.MessageID,
			Timestamp:    time.Now(),
			Body:         body,
			Headers: amqp.Table{
				"instruction_id":  msg.InstructionID,
				"device_sn":       msg.DeviceSN,
				"command_code":    msg.CommandCode,
				"retry_count":     msg.RetryCount,
				"priority":        msg.Priority,
				"created_at":      msg.CreatedAt.Unix(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("publish command: %w", err)
	}

	logx.Infof("[RabbitMQ] Command published: message_id=%s, instruction_id=%d, device_sn=%s, cmd=%s, routing_key=%s",
		msg.MessageID, msg.InstructionID, msg.DeviceSN, msg.CommandCode, routingKey)

	return nil
}

func (m *Manager) PublishRetry(ctx context.Context, msg *CommandMessage, delayMs int) error {
	if !m.IsConnected() {
		return fmt.Errorf("RabbitMQ not connected")
	}

	msg.RetryCount++
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal retry message: %w", err)
	}

	ch, err := m.GetChannel()
	if err != nil {
		return fmt.Errorf("get channel: %w", err)
	}

	routingKey := fmt.Sprintf("cmd.retry.%s", msg.DeviceSN)

	err = ch.PublishWithContext(ctx,
		m.config.Exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    msg.MessageID,
			Timestamp:    time.Now(),
			Expiration:   fmt.Sprintf("%d", delayMs),
			Body:         body,
			Headers: amqp.Table{
				"instruction_id":  msg.InstructionID,
				"device_sn":       msg.DeviceSN,
				"command_code":    msg.CommandCode,
				"retry_count":     msg.RetryCount,
				"is_retry":        true,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("publish retry: %w", err)
	}

	logx.Infof("[RabbitMQ] Retry published: message_id=%s, instruction_id=%d, retry_count=%d/%d, delay=%dms",
		msg.MessageID, msg.InstructionID, msg.RetryCount, msg.MaxRetry, delayMs)

	return nil
}
