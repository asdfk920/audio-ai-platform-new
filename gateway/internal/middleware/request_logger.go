package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/audio-ai-platform/gateway/internal/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLogger 请求日志中间件
type RequestLogger struct {
	logger *zap.Logger
}

// NewRequestLogger 创建请求日志中间件
func NewRequestLogger(cfg config.LogConfig) (*RequestLogger, error) {
	var zapConfig zap.Config
	
	if cfg.Format == "json" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}
	
	// 设置日志级别
	switch cfg.Level {
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	
	logger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}
	
	return &RequestLogger{
		logger: logger,
	}, nil
}

// Handler 请求日志处理函数
func (rl *RequestLogger) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录开始时间
		start := time.Now()
		
		// 读取请求体（用于记录请求参数）
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}
		
		// 创建响应写入器来捕获响应
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw
		
		// 处理请求
		c.Next()
		
		// 计算处理时间
		duration := time.Since(start)
		
		// 构建日志字段
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
			zap.Int("response_size", c.Writer.Size()),
		}
		
		// 添加服务名称（如果有）
		if serviceName, exists := c.Get("service_name"); exists {
			fields = append(fields, zap.String("service", serviceName.(string)))
		}
		
		// 添加用户信息（如果已认证）
		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, zap.Int64("user_id", userID.(int64)))
		}
		
		if username, exists := c.Get("username"); exists {
			fields = append(fields, zap.String("username", username.(string)))
		}
		
		// 记录请求体（仅记录小请求）
		if len(requestBody) > 0 && len(requestBody) < 1024 {
			var jsonBody map[string]interface{}
			if err := json.Unmarshal(requestBody, &jsonBody); err == nil {
				fields = append(fields, zap.Any("request_body", jsonBody))
			} else {
				fields = append(fields, zap.String("request_body", string(requestBody)))
			}
		}
		
		// 记录响应体（仅记录小响应）
		if blw.body.Len() > 0 && blw.body.Len() < 1024 {
			var jsonResponse map[string]interface{}
			if err := json.Unmarshal(blw.body.Bytes(), &jsonResponse); err == nil {
				fields = append(fields, zap.Any("response_body", jsonResponse))
			}
		}
		
		// 根据状态码选择日志级别
		status := c.Writer.Status()
		switch {
		case status >= 500:
			rl.logger.Error("请求处理错误", fields...)
		case status >= 400:
			rl.logger.Warn("客户端错误", fields...)
		default:
			rl.logger.Info("请求处理完成", fields...)
		}
	}
}

// bodyLogWriter 用于捕获响应体的写入器
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Close 关闭日志记录器
func (rl *RequestLogger) Close() error {
	return rl.logger.Sync()
}

// GetLogger 获取底层日志记录器
func (rl *RequestLogger) GetLogger() *zap.Logger {
	return rl.logger
}