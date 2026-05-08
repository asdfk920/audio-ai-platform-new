package model

import (
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// PerformanceMonitor 性能监控器
// 监控推理延迟、吞吐量等指标
type PerformanceMonitor struct {
	mu sync.RWMutex

	// 延迟统计
	latencies      []time.Duration
	maxLatencies   int
	totalInference int64

	// 吞吐量统计
	inferencesPerSecond float64
	lastResetTime       time.Time

	// 延迟目标
	targetLatencyMs int64 // 目标延迟（毫秒），默认 100ms

	// 回调函数
	onLatencyExceeded func(latency time.Duration)
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		latencies:       make([]time.Duration, 0, 1000),
		maxLatencies:    1000,
		targetLatencyMs: 100, // 目标：单路延迟<100ms
		lastResetTime:   time.Now(),
	}
}

// RecordLatency 记录推理延迟
func (pm *PerformanceMonitor) RecordLatency(latency time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.latencies = append(pm.latencies, latency)
	pm.totalInference++

	// 限制数组大小
	if len(pm.latencies) > pm.maxLatencies {
		pm.latencies = pm.latencies[1:]
	}

	// 检查是否超过目标延迟
	if latency.Milliseconds() > pm.targetLatencyMs {
		logx.Errorf("推理延迟超过目标：%dms > %dms",
			latency.Milliseconds(), pm.targetLatencyMs)

		if pm.onLatencyExceeded != nil {
			pm.onLatencyExceeded(latency)
		}
	}

	// 计算吞吐量
	elapsed := time.Since(pm.lastResetTime).Seconds()
	if elapsed > 0 {
		pm.inferencesPerSecond = float64(pm.totalInference) / elapsed
	}
}

// GetAverageLatency 获取平均延迟
func (pm *PerformanceMonitor) GetAverageLatency() time.Duration {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.latencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, lat := range pm.latencies {
		total += lat
	}

	return total / time.Duration(len(pm.latencies))
}

// GetP50Latency 获取 P50 延迟（中位数）
func (pm *PerformanceMonitor) GetP50Latency() time.Duration {
	return pm.getPercentile(50)
}

// GetP95Latency 获取 P95 延迟
func (pm *PerformanceMonitor) GetP95Latency() time.Duration {
	return pm.getPercentile(95)
}

// GetP99Latency 获取 P99 延迟
func (pm *PerformanceMonitor) GetP99Latency() time.Duration {
	return pm.getPercentile(99)
}

// getPercentile 获取百分位数延迟
func (pm *PerformanceMonitor) getPercentile(percentile int) time.Duration {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if len(pm.latencies) == 0 {
		return 0
	}

	// 复制并排序
	sorted := make([]time.Duration, len(pm.latencies))
	copy(sorted, pm.latencies)

	// 简单排序（实际应该用更高效的算法）
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// 计算百分位索引
	idx := (len(sorted) * percentile) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}

// GetInferencesPerSecond 获取每秒推理次数
func (pm *PerformanceMonitor) GetInferencesPerSecond() float64 {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.inferencesPerSecond
}

// GetStats 获取统计信息
func (pm *PerformanceMonitor) GetStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	avgLatency := time.Duration(0)
	if len(pm.latencies) > 0 {
		var total time.Duration
		for _, lat := range pm.latencies {
			total += lat
		}
		avgLatency = total / time.Duration(len(pm.latencies))
	}

	return map[string]interface{}{
		"avg_latency_ms":     avgLatency.Milliseconds(),
		"p50_latency_ms":     pm.getPercentile(50).Milliseconds(),
		"p95_latency_ms":     pm.getPercentile(95).Milliseconds(),
		"p99_latency_ms":     pm.getPercentile(99).Milliseconds(),
		"inferences_per_sec": pm.inferencesPerSecond,
		"total_inferences":   pm.totalInference,
		"target_latency_ms":  pm.targetLatencyMs,
		"meets_target":       avgLatency.Milliseconds() <= pm.targetLatencyMs,
	}
}

// Reset 重置统计
func (pm *PerformanceMonitor) Reset() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.latencies = pm.latencies[:0]
	pm.totalInference = 0
	pm.inferencesPerSecond = 0
	pm.lastResetTime = time.Now()
}

// SetTargetLatency 设置目标延迟
func (pm *PerformanceMonitor) SetTargetLatency(ms int64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.targetLatencyMs = ms
}

// SetOnLatencyExceeded 设置延迟超标回调
func (pm *PerformanceMonitor) SetOnLatencyExceeded(callback func(time.Duration)) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.onLatencyExceeded = callback
}

// LatencyOptimizer 延迟优化器
type LatencyOptimizer struct {
	mu           sync.RWMutex
	currentBatch int
	optimalBatch int
	minLatency   time.Duration
}

// NewLatencyOptimizer 创建延迟优化器
func NewLatencyOptimizer() *LatencyOptimizer {
	return &LatencyOptimizer{
		currentBatch: 1,
		optimalBatch: 1,
		minLatency:   time.Hour, // 初始化为很大的值
	}
}

// OptimizeBatchSize 优化批量大小
func (lo *LatencyOptimizer) OptimizeBatchSize(batchSize int, latency time.Duration) int {
	lo.mu.Lock()
	defer lo.mu.Unlock()

	if latency < lo.minLatency {
		lo.minLatency = latency
		lo.optimalBatch = batchSize
	}

	return lo.optimalBatch
}

// GetOptimalBatchSize 获取最优批量大小
func (lo *LatencyOptimizer) GetOptimalBatchSize() int {
	lo.mu.RLock()
	defer lo.mu.RUnlock()
	return lo.optimalBatch
}
