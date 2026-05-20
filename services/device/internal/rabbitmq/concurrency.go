package rabbitmq

import (
	"context"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConcurrencyController struct {
	globalSem    chan struct{}
	deviceSems   map[string]chan struct{}
	deviceSemMu sync.RWMutex
	maxGlobal    int
	maxPerDevice int
}

func NewConcurrencyController(maxGlobal, maxPerDevice int) *ConcurrencyController {
	if maxGlobal <= 0 {
		maxGlobal = DefaultMaxConcurrent
	}
	if maxPerDevice <= 0 {
		maxPerDevice = DefaultMaxDeviceConc
	}
	return &ConcurrencyController{
		globalSem:    make(chan struct{}, maxGlobal),
		deviceSems:   make(map[string]chan struct{}),
		maxGlobal:    maxGlobal,
		maxPerDevice: maxPerDevice,
	}
}

func (c *ConcurrencyController) Acquire(ctx context.Context, deviceSN string) error {
	select {
	case c.globalSem <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}

	if err := c.acquireDevice(ctx, deviceSN); err != nil {
		c.releaseGlobal()
		return err
	}
	return nil
}

func (c *ConcurrencyController) acquireDevice(ctx context.Context, deviceSN string) error {
	c.deviceSemMu.Lock()
	sem, ok := c.deviceSems[deviceSN]
	if !ok {
		sem = make(chan struct{}, c.maxPerDevice)
		c.deviceSems[deviceSN] = sem
	}
	c.deviceSemMu.Unlock()

	select {
	case sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *ConcurrencyController) Release(deviceSN string) {
	c.releaseDevice(deviceSN)
	c.releaseGlobal()
}

func (c *ConcurrencyController) releaseDevice(deviceSN string) {
	c.deviceSemMu.RLock()
	sem, ok := c.deviceSems[deviceSN]
	c.deviceSemMu.RUnlock()

	if ok {
		select {
		case <-sem:
		default:
		}
	}
}

func (c *ConcurrencyController) releaseGlobal() {
	select {
	case <-c.globalSem:
	default:
	}
}

func (c *ConcurrencyController) GlobalUsage() int {
	return len(c.globalSem)
}

func (c *ConcurrencyController) DeviceUsage(deviceSN string) int {
	c.deviceSemMu.RLock()
	defer c.deviceSemMu.RUnlock()
	if sem, ok := c.deviceSems[deviceSN]; ok {
		return len(sem)
	}
	return 0
}

type RateLimiter struct {
	tokens       chan struct{}
	refillRate   time.Duration
	refillTicker *time.Ticker
	stopChan     chan struct{}
	mu           sync.Mutex
}

func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	if rate <= 0 {
		rate = 100
	}
	if interval <= 0 {
		interval = time.Second
	}
	rl := &RateLimiter{
		tokens:     make(chan struct{}, rate),
		refillRate: interval,
		stopChan:   make(chan struct{}),
	}
	for i := 0; i < rate; i++ {
		rl.tokens <- struct{}{}
	}
	go rl.refillLoop()
	return rl
}

func (rl *RateLimiter) refillLoop() {
	rl.refillTicker = time.NewTicker(rl.refillRate)
	defer rl.refillTicker.Stop()

	for {
		select {
		case <-rl.refillTicker.C:
			select {
			case rl.tokens <- struct{}{}:
			default:
			}
		case <-rl.stopChan:
			return
		}
	}
}

func (rl *RateLimiter) Allow() bool {
	select {
	case <-rl.tokens:
		return true
	default:
		logx.Infof("[RateLimiter] Rate limit exceeded")
		return false
	}
}

func (rl *RateLimiter) Wait(ctx context.Context) bool {
	select {
	case <-rl.tokens:
		return true
	case <-ctx.Done():
		return false
	}
}

func (rl *RateLimiter) Stop() {
	close(rl.stopChan)
}
