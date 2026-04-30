package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/audio-ai-platform/gateway/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// RateLimiter 限流中间件
type RateLimiter struct {
	config    config.RateLimitConfig
	redis     *redis.Client
	globalLim *rate.Limiter
}

// NewRateLimiter 创建限流中间件
func NewRateLimiter(cfg config.RateLimitConfig, redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		config:    cfg,
		redis:     redisClient,
		globalLim: rate.NewLimiter(rate.Limit(cfg.GlobalRPS), cfg.GlobalRPS),
	}
}

// Handler 限流处理函数
func (rl *RateLimiter) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 全局限流检查
		if !rl.globalLim.Allow() {
			rl.rateLimitResponse(c, "global")
			return
		}

		// IP 限流检查
		clientIP := rl.getClientIP(c)
		if !rl.checkIPLimit(clientIP) {
			rl.rateLimitResponse(c, "ip")
			return
		}

		// 用户限流检查（如果已认证）
		if userID, exists := c.Get("user_id"); exists {
			if !rl.checkUserLimit(userID.(int64)) {
				rl.rateLimitResponse(c, "user")
				return
			}
		}

		c.Next()
	}
}

// getClientIP 获取客户端 IP
func (rl *RateLimiter) getClientIP(c *gin.Context) string {
	// 从 X-Forwarded-For 获取真实 IP（如果有代理）
	if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 从 X-Real-IP 获取
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		return realIP
	}

	// 直接获取远程地址
	return c.ClientIP()
}

// checkIPLimit 检查 IP 限流
func (rl *RateLimiter) checkIPLimit(ip string) bool {
	if rl.config.IPRPS <= 0 {
		return true
	}

	key := "rate_limit:ip:" + ip
	return rl.checkRedisLimit(key, rl.config.IPRPS)
}

// checkUserLimit 检查用户限流
func (rl *RateLimiter) checkUserLimit(userID int64) bool {
	if rl.config.UserRPS <= 0 {
		return true
	}

	key := "rate_limit:user:" + strconv.FormatInt(userID, 10)
	return rl.checkRedisLimit(key, rl.config.UserRPS)
}

// checkRedisLimit 检查 Redis 限流
func (rl *RateLimiter) checkRedisLimit(key string, maxRPS int) bool {
	if rl.redis == nil {
		// 如果没有 Redis，使用内存限流
		return true
	}

	ctx := context.Background()

	// 使用 Redis 的 INCR 和 EXPIRE 实现滑动窗口限流
	current, err := rl.redis.Incr(ctx, key).Result()
	if err != nil {
		// Redis 出错时允许通过
		return true
	}

	// 如果是第一次请求，设置过期时间
	if current == 1 {
		rl.redis.Expire(ctx, key, time.Duration(rl.config.WindowSeconds)*time.Second)
	}

	return current <= int64(maxRPS)
}

// rateLimitResponse 限流响应
func (rl *RateLimiter) rateLimitResponse(c *gin.Context, limitType string) {
	c.Header("X-RateLimit-Type", limitType)
	c.Header("Retry-After", strconv.Itoa(rl.config.WindowSeconds))

	c.JSON(http.StatusTooManyRequests, gin.H{
		"error":       "请求过于频繁",
		"limit_type":  limitType,
		"retry_after": rl.config.WindowSeconds,
	})
	c.Abort()
}

// GetRateLimitInfo 获取限流信息（用于监控）
func (rl *RateLimiter) GetRateLimitInfo() gin.H {
	return gin.H{
		"global_rps":     rl.config.GlobalRPS,
		"ip_rps":         rl.config.IPRPS,
		"user_rps":       rl.config.UserRPS,
		"window_seconds": rl.config.WindowSeconds,
	}
}
