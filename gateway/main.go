package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/audio-ai-platform/gateway/internal/config"
	"github.com/audio-ai-platform/gateway/internal/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化 Redis 客户端
	var redisClient *redis.Client
	if cfg.Redis.Addr != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})

		// 测试 Redis 连接
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Printf("警告: Redis 连接失败: %v", err)
			redisClient = nil
		} else {
			log.Println("Redis 连接成功")
		}
	}

	// 初始化日志中间件
	logger, err := middleware.NewRequestLogger(cfg.Log)
	if err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Close()

	// 设置 Gin 模式
	if cfg.Log.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建 Gin 引擎
	r := gin.New()

	// 使用日志中间件
	r.Use(logger.Handler())

	// 配置 CORS
	corsConfig := cors.Config{
		AllowOrigins:     cfg.CORS.AllowOrigins,
		AllowMethods:     cfg.CORS.AllowMethods,
		AllowHeaders:     cfg.CORS.AllowHeaders,
		ExposeHeaders:    cfg.CORS.ExposeHeaders,
		MaxAge:           time.Duration(cfg.CORS.MaxAge) * time.Second,
		AllowCredentials: cfg.CORS.AllowCredentials,
	}
	r.Use(cors.New(corsConfig))

	// 初始化 JWT 认证中间件
	jwtAuth := middleware.NewJWTAuth(cfg.JWT)
	r.Use(jwtAuth.Handler())

	// 初始化限流中间件
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimit, redisClient)
	r.Use(rateLimiter.Handler())

	// 初始化路由代理
	routeProxy := middleware.NewRouteProxy(cfg.Routes)

	// 健康检查路由
	r.GET("/health", routeProxy.HealthCheck())

	// 监控路由
	r.GET("/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "running",
			"timestamp":   time.Now().Unix(),
			"rate_limits": rateLimiter.GetRateLimitInfo(),
			"routes":      routeProxy.GetRoutesInfo(),
		})
	})

	// 所有其他请求走路由代理
	r.Any("/*path", routeProxy.Handler())

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler: r,
	}

	// 启动服务器
	go func() {
		logger.GetLogger().Info("网关服务启动",
			zap.String("address", srv.Addr),
			zap.String("name", cfg.Name),
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.GetLogger().Error("服务器启动失败",
				zap.Error(err),
			)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.GetLogger().Info("正在关闭网关服务...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.GetLogger().Error("服务器关闭失败",
			zap.Error(err),
		)
	}

	logger.GetLogger().Info("网关服务已关闭")
}
