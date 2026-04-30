package reg

import (
	"sync"
	"time"
)

// IPLimiter 固定窗口内每 IP 最大请求次数（进程内；多实例需网关或 Redis）。
type IPLimiter struct {
	window time.Duration
	max    int
	m      sync.Map // string -> *ipEntry
}

type ipEntry struct {
	mu      sync.Mutex
	count   int
	resetAt time.Time
}

// NewIPLimiter max<=0 表示关闭限流（Allow 恒 true）。
func NewIPLimiter(window time.Duration, max int) *IPLimiter {
	if window <= 0 {
		window = time.Minute
	}
	return &IPLimiter{window: window, max: max}
}

// Allow 在窗口内递增计数，超过 max 返回 false。
func (l *IPLimiter) Allow(key string) bool {
	if l == nil || l.max <= 0 {
		return true
	}
	if key == "" {
		key = "unknown"
	}
	v, _ := l.m.LoadOrStore(key, &ipEntry{})
	e := v.(*ipEntry)
	e.mu.Lock()
	defer e.mu.Unlock()
	now := time.Now()
	if now.After(e.resetAt) {
		e.count = 0
		e.resetAt = now.Add(l.window)
	}
	if e.count >= l.max {
		return false
	}
	e.count++
	return true
}
