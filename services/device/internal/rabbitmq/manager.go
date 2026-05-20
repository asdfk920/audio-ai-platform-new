package rabbitmq

import (
	"context"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/device/internal/config"
)

const (
	DefaultExchange       = "device.cmd.exchange"
	DefaultQueueName      = "device.cmd.queue"
	DefaultRetryQueueName = "device.cmd.retry.queue"
	DefaultDLQQueueName   = "device.cmd.dlq.queue"
	DefaultPrefetchCount  = 10
	DefaultMaxConcurrent  = 100
	DefaultMaxDeviceConc  = 3
	DefaultTimeoutSec     = 30
	DefaultMaxRetry       = 3
	DefaultRetryDelayMs   = 1000
	DefaultReconnectSec   = 5
)

type Manager struct {
	config        config.RabbitMQ
	conn          *amqp.Connection
	channel       *amqp.Channel
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	reconnecting  bool
	reconnectMu   sync.Mutex
	done          chan struct{}
	onConnected   func()
	onDisconnected func()
}

func NewManager(ctx context.Context, cfg config.RabbitMQ) *Manager {
	ctx, cancel := context.WithCancel(ctx)
	mgr := &Manager{
		config: setDefaults(cfg),
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	return mgr
}

func (m *Manager) SetOnConnected(fn func()) { m.onConnected = fn }
func (m *Manager) SetOnDisconnected(fn func()) { m.onDisconnected = fn }

func setDefaults(cfg config.RabbitMQ) config.RabbitMQ {
	if cfg.Exchange == "" { cfg.Exchange = DefaultExchange }
	if cfg.QueueName == "" { cfg.QueueName = DefaultQueueName }
	if cfg.RetryQueueName == "" { cfg.RetryQueueName = DefaultRetryQueueName }
	if cfg.DLQQueueName == "" { cfg.DLQQueueName = DefaultDLQQueueName }
	if cfg.PrefetchCount <= 0 { cfg.PrefetchCount = DefaultPrefetchCount }
	if cfg.MaxConcurrent <= 0 { cfg.MaxConcurrent = DefaultMaxConcurrent }
	if cfg.MaxDeviceConcurrent <= 0 { cfg.MaxDeviceConcurrent = DefaultMaxDeviceConc }
	if cfg.DefaultTimeout <= 0 { cfg.DefaultTimeout = DefaultTimeoutSec }
	if cfg.MaxRetry <= 0 { cfg.MaxRetry = DefaultMaxRetry }
	if cfg.RetryDelayMs <= 0 { cfg.RetryDelayMs = DefaultRetryDelayMs }
	if cfg.ReconnectIntervalSec <= 0 { cfg.ReconnectIntervalSec = DefaultReconnectSec }
	return cfg
}

func (m *Manager) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	url := m.config.URL
	if url == "" {
		return fmt.Errorf("RabbitMQ URL is required")
	}

	var err error
	m.conn, err = amqp.Dial(url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	m.channel, err = m.conn.Channel()
	if err != nil {
		m.conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	if err := m.setupTopology(); err != nil {
		m.channel.Close()
		m.conn.Close()
		return fmt.Errorf("failed to setup topology: %w", err)
	}

	logx.Infof("[RabbitMQ] Connected and topology ready: exchange=%s, queue=%s",
		m.config.Exchange, m.config.QueueName)

	go m.monitorConnection()

	if m.onConnected != nil {
		m.onConnected()
	}
	return nil
}

func (m *Manager) setupTopology() error {
	err := m.channel.ExchangeDeclare(
		m.config.Exchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	args := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": m.config.DLQQueueName,
	}
	if m.config.DLQEnable {
		_, err = m.channel.QueueDeclare(
			m.config.DLQQueueName,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("declare DLQ: %w", err)
		}
	}

	_, err = m.channel.QueueDeclare(
		m.config.QueueName,
		true,
		false,
		false,
		false,
		args,
	)
	if err != nil {
		return fmt.Errorf("declare main queue: %w", err)
	}

	retryArgs := amqp.Table{
		"x-message-ttl":             int32(m.config.RetryDelayMs),
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": m.config.QueueName,
	}
	_, err = m.channel.QueueDeclare(
		m.config.RetryQueueName,
		true,
		false,
		false,
		false,
		retryArgs,
	)
	if err != nil {
		return fmt.Errorf("declare retry queue: %w", err)
	}

	err = m.channel.QueueBind(m.config.QueueName, "cmd.*", m.config.Exchange, false, nil)
	if err != nil {
		return fmt.Errorf("bind main queue: %w", err)
	}
	err = m.channel.QueueBind(m.config.RetryQueueName, "cmd.retry.*", m.config.Exchange, false, nil)
	if err != nil {
		return fmt.Errorf("bind retry queue: %w", err)
	}

	err = m.channel.Qos(m.config.PrefetchCount, 0, false)
	if err != nil {
		return fmt.Errorf("set QoS prefetch=%d: %w", m.config.PrefetchCount, err)
	}

	return nil
}

func (m *Manager) monitorConnection() {
	closeCh := make(chan *amqp.Error)
	m.mu.RLock()
	if m.conn != nil {
		closeCh = m.conn.NotifyClose(make(chan *amqp.Error, 1))
	}
	m.mu.RUnlock()

	select {
	case <-closeCh:
		logx.Infof("[RabbitMQ] Connection lost, reconnecting...")
		if m.onDisconnected != nil {
			m.onDisconnected()
		}
		m.reconnectLoop()
	case <-m.ctx.Done():
		logx.Infof("[RabbitMQ] Shutdown requested")
	case <-m.done:
		return
	}
}

func (m *Manager) reconnectLoop() {
	m.reconnectMu.Lock()
	if m.reconnecting {
		m.reconnectMu.Unlock()
		return
	}
	m.reconnecting = true
	m.reconnectMu.Unlock()

	defer func() {
		m.reconnectMu.Lock()
		m.reconnecting = false
		m.reconnectMu.Unlock()
	}()

	ticker := time.NewTicker(time.Duration(m.config.ReconnectIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-m.done:
			return
		case <-ticker.C:
			logx.Infof("[RabbitMQ] Attempting to reconnect...")
			if err := m.Connect(); err != nil {
				logx.Errorf("[RabbitMQ] Reconnect failed: %v", err)
				continue
			}
			logx.Infof("[RabbitMQ] Reconnected successfully")
			return
		}
	}
}

func (m *Manager) GetChannel() (*amqp.Channel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.channel == nil || m.channel.IsClosed() {
		return nil, fmt.Errorf("channel not available")
	}
	return m.channel, nil
}

func (m *Manager) Close() error {
	select {
	case <-m.done:
		return nil
	default:
		close(m.done)
		m.cancel()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	if m.channel != nil && !m.channel.IsClosed() {
		if err := m.channel.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close channel: %w", err))
		}
	}
	if m.conn != nil && !m.conn.IsClosed() {
		if err := m.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close connection: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %+v", errs)
	}
	return nil
}

func (m *Manager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conn != nil && !m.conn.IsClosed()
}
