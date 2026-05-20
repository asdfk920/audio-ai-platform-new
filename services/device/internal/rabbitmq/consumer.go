package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zeromicro/go-zero/core/logx"
)

type CommandHandler func(ctx context.Context, msg *CommandMessage) error
type StatusUpdater func(instructionID int64, status string, errorMsg string)

type Consumer struct {
	mgr           *Manager
	handler       CommandHandler
	statusUpdater StatusUpdater
	concurrency   *ConcurrencyController
	rateLimiter   *RateLimiter
	workerCount   int
}

func NewConsumer(mgr *Manager, handler CommandHandler, statusUpdater StatusUpdater) *Consumer {
	return &Consumer{
		mgr:           mgr,
		handler:       handler,
		statusUpdater: statusUpdater,
		concurrency: NewConcurrencyController(
			mgr.config.MaxConcurrent,
			mgr.config.MaxDeviceConcurrent,
		),
		rateLimiter: NewRateLimiter(50, time.Second),
		workerCount: mgr.config.PrefetchCount,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	logx.Infof("[Consumer] Starting command consumer with %d workers...", c.workerCount)

	ch, err := c.mgr.GetChannel()
	if err != nil {
		return fmt.Errorf("get channel: %w", err)
	}

	msgs, err := ch.Consume(
		c.mgr.config.QueueName,
		fmt.Sprintf("consumer-%d", time.Now().UnixNano()),
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("start consume: %w", err)
	}

	for i := 0; i < c.workerCount; i++ {
		go c.worker(ctx, msgs, i)
	}

	retryMsgs, err := ch.Consume(
		c.mgr.config.RetryQueueName,
		fmt.Sprintf("retry-consumer-%d", time.Now().UnixNano()),
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logx.Errorf("[Consumer] Failed to start retry consumer: %v", err)
	} else {
		go c.worker(ctx, retryMsgs, c.workerCount+1)
	}

	logx.Infof("[Consumer] ✅ Command consumer started successfully (workers=%d)", c.workerCount*2)

	select {
	case <-ctx.Done():
		logx.Infof("[Consumer] Stopping consumer...")
		return ctx.Err()
	}
}

func (c *Consumer) worker(ctx context.Context, msgs <-chan amqp.Delivery, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-msgs:
			if !ok {
				logx.Infof("[Consumer-Worker%d] Channel closed", workerID)
				return
			}
			c.processMessage(ctx, msg, workerID)
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, delivery amqp.Delivery, workerID int) {
	startTime := time.Now()

	var cmdMsg CommandMessage
	if err := json.Unmarshal(delivery.Body, &cmdMsg); err != nil {
		logx.Errorf("[Consumer-Worker%d] Failed to unmarshal message: %v", workerID, err)
		c.nack(delivery)
		return
	}

	if cmdMsg.ExpiresAt.Before(time.Now()) {
		logx.Infof("[Consumer-Worker%d] Message expired: message_id=%s, instruction_id=%d",
			workerID, cmdMsg.MessageID, cmdMsg.InstructionID)
		c.updateStatus(cmdMsg.InstructionID, "timeout", "message expired")
		c.ack(delivery)
		return
	}

	logx.Infof("\n[Consumer-Worker%d] Processing command:", workerID)
	logx.Infof("  Message ID:     %s", cmdMsg.MessageID)
	logx.Infof("  Instruction ID: %d", cmdMsg.InstructionID)
	logx.Infof("  Device SN:      %s", cmdMsg.DeviceSN)
	logx.Infof("  Command Code:   %s", cmdMsg.CommandCode)
	logx.Infof("  Retry Count:    %d / %d", cmdMsg.RetryCount, cmdMsg.MaxRetry)
	logx.Infof("  Priority:       %d", cmdMsg.Priority)
	logx.Infof("  Created At:     %s", cmdMsg.CreatedAt.Format(time.RFC3339))

	if !c.rateLimiter.Allow() {
		logx.Infof("[Consumer-Worker%d] Rate limited, will retry later: message_id=%s",
			workerID, cmdMsg.MessageID)
		c.retryLater(&cmdMsg, delivery)
		return
	}

	acquireCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := c.concurrency.Acquire(acquireCtx, cmdMsg.DeviceSN); err != nil {
		logx.Infof("[Consumer-Worker%d] Concurrency limit reached: device_sn=%s, global=%d/%d, device=%d/%d",
			workerID, cmdMsg.DeviceSN,
			c.concurrency.GlobalUsage(), c.mgr.config.MaxConcurrent,
			c.concurrency.DeviceUsage(cmdMsg.DeviceSN), c.mgr.config.MaxDeviceConcurrent)
		c.retryLater(&cmdMsg, delivery)
		return
	}
	defer c.concurrency.Release(cmdMsg.DeviceSN)

	timeoutSec := c.mgr.config.DefaultTimeout
	if cmdMsg.TimeoutSeconds > 0 && cmdMsg.TimeoutSeconds < timeoutSec {
		timeoutSec = cmdMsg.TimeoutSeconds
	}

	execCtx, execCancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer execCancel()

	err := c.handler(execCtx, &cmdMsg)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		c.handleFailure(&cmdMsg, delivery, err, workerID)
	} else {
		c.handleSuccess(&cmdMsg, delivery, duration, workerID)
	}
}

func (c *Consumer) handleSuccess(msg *CommandMessage, delivery amqp.Delivery, durationMs int64, workerID int) {
	c.ack(delivery)
	c.updateStatus(msg.InstructionID, "success", "")

	logx.Infof("\n[Consumer-Worker%d] ✅ Command executed successfully!", workerID)
	logx.Infof("  Message ID:      %s", msg.MessageID)
	logx.Infof("  Instruction ID:  %d", msg.InstructionID)
	logx.Infof("  Device SN:       %s", msg.DeviceSN)
	logx.Infof("  Duration:        %dms", durationMs)
	logx.Infof("  Global Usage:    %d / %d", c.concurrency.GlobalUsage(), c.mgr.config.MaxConcurrent)
	logx.Infof("  Device Usage:    %d / %d", c.concurrency.DeviceUsage(msg.DeviceSN), c.mgr.config.MaxDeviceConcurrent)
}

func (c *Consumer) handleFailure(msg *CommandMessage, delivery amqp.Delivery, err error, workerID int) {
	logx.Errorf("[Consumer-Worker%d] ❌ Command failed: message_id=%s, instruction_id=%d, error=%v",
		workerID, msg.MessageID, msg.InstructionID, err)

	if msg.RetryCount < msg.MaxRetry {
		delayMs := c.calculateBackoffDelay(msg.RetryCount)
		logx.Infof("[Consumer-Worker%d] Scheduling retry: attempt=%d/%d, delay=%dms",
			workerID, msg.RetryCount+1, msg.MaxRetry, delayMs)

		publishErr := c.mgr.PublishRetry(context.Background(), msg, delayMs)
		if publishErr != nil {
			logx.Errorf("[Consumer-Worker%d] Failed to publish retry: %v", workerID, publishErr)
			c.updateStatus(msg.InstructionID, "failed", fmt.Sprintf("retry failed: %v", publishErr))
			c.nack(delivery)
		} else {
			c.updateStatus(msg.InstructionID, "retrying", fmt.Sprintf("attempt %d/%d, next in %dms",
				msg.RetryCount+1, msg.MaxRetry, delayMs))
			c.ack(delivery)
		}
	} else {
		logx.Errorf("[Consumer-Worker%d] Max retries exceeded, sending to DLQ: message_id=%s",
			workerID, msg.MessageID)
		c.updateStatus(msg.InstructionID, "failed", fmt.Sprintf("max retries (%d) exceeded: %v",
			msg.MaxRetry, err))
		c.nack(delivery)
	}
}

func (c *Consumer) calculateBackoffDelay(retryCount int) int {
	baseDelay := c.mgr.config.RetryDelayMs
	delay := float64(baseDelay) * math.Pow(2, float64(retryCount))
	maxDelay := 30000

	if int(delay) > maxDelay {
		return maxDelay
	}
	return int(delay)
}

func (c *Consumer) retryLater(msg *CommandMessage, delivery amqp.Delivery) {
	if msg.RetryCount < msg.MaxRetry {
		delayMs := 500
		publishErr := c.mgr.PublishRetry(context.Background(), msg, delayMs)
		if publishErr == nil {
			c.ack(delivery)
			return
		}
	}
	c.nack(delivery)
}

func (c *Consumer) ack(delivery amqp.Delivery) {
	if err := delivery.Ack(false); err != nil {
		logx.Errorf("[Consumer] Ack failed: %v", err)
	}
}

func (c *Consumer) nack(delivery amqp.Delivery) {
	if err := delivery.Nack(false, false); err != nil {
		logx.Errorf("[Consumer] Nack failed: %v", err)
	}
}

func (c *Consumer) updateStatus(instructionID int64, status, errorMsg string) {
	if c.statusUpdater != nil {
		go c.statusUpdater(instructionID, status, errorMsg)
	}
}

var ErrCommandTimeout = errors.New("command execution timeout")
