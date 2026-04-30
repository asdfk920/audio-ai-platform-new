// Package status 设备状态查询 Prometheus 指标（与 shadow 解耦）。
package status

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	statusRedisHit = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "device",
		Subsystem: "status_query",
		Name:      "redis_hit_total",
		Help:      "设备状态查询命中 Redis 影子次数",
	})
	statusRedisMiss = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "device",
		Subsystem: "status_query",
		Name:      "redis_miss_total",
		Help:      "设备状态查询 Redis 无有效影子次数",
	})
	statusDBFallback = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "device",
		Subsystem: "status_query",
		Name:      "db_fallback_total",
		Help:      "设备状态查询因 Redis 异常走 DB 降级次数",
	})
	statusSeed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "device",
		Subsystem: "status_query",
		Name:      "seed_shadow_total",
		Help:      "设备状态查询触发影子回种次数",
	})
	statusNegHit = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "device",
		Subsystem: "status_query",
		Name:      "neg_cache_hit_total",
		Help:      "设备状态查询命中负缓存（无绑定）次数",
	})
)

func IncRedisHit()    { statusRedisHit.Inc() }
func IncRedisMiss()   { statusRedisMiss.Inc() }
func IncDBFallback()  { statusDBFallback.Inc() }
func IncSeed()        { statusSeed.Inc() }
func IncNegCacheHit() { statusNegHit.Inc() }
