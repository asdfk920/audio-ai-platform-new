package middleware

import (
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogMiddleware struct{}

func NewLogMiddleware() *LogMiddleware {
	return &LogMiddleware{}
}

func (m *LogMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 记录请求信息
		logx.Infof("Request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// 执行下一个处理器
		next(w, r)

		// 记录响应时间
		duration := time.Since(start)
		logx.Infof("Response: %s %s took %v", r.Method, r.URL.Path, duration)
	}
}
