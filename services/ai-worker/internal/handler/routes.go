// Package handler 音频上传 API 路由注册
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/middleware/auth"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
	"github.com/zeromicro/go-zero/rest"
)

// healthCheckHandler 健康检查处理
func healthCheckHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"message": "服务运行正常",
		})
	}
}

// RegisterHandlers 注册 ai-worker 全部路由
func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	// 公开接口（无需认证）
	publicRoutes := []rest.Route{
		{
			// 健康检查
			// GET /api/v1/health
			Method:  http.MethodGet,
			Path:    "/api/v1/health",
			Handler: healthCheckHandler(serverCtx),
		},
	}

	server.AddRoutes(publicRoutes)

	// JWT 保护的接口（需要登录）
	jwtProtected := []rest.Route{
		{
			// 发起音频分离任务（HTTP 版本）
			// POST /api/v1/audio/separate
			Method:  http.MethodPost,
			Path:    "/audio/separate",
			Handler: AudioSeparateHandler(serverCtx),
		},
		{
			// 查询分离任务列表
			// GET /api/v1/inference/tasks
			Method:  http.MethodGet,
			Path:    "/inference/tasks",
			Handler: TaskListHandler(serverCtx),
		},
	}

	server.AddRoutes(
		rest.WithMiddleware(
			auth.Middleware(serverCtx.Config.Auth.AccessSecret),
			jwtProtected...,
		),
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/api/v1"),
	)

	// WebSocket 接口（自行处理 Token 认证，独立注册）
	wsRoutes := []rest.Route{
		{
			// WebSocket 音轨分离
			// GET /api/v1/audio/separate/ws
			// 用途：通过 WebSocket 长连接实现音轨分离
			Method:  http.MethodGet,
			Path:    "/api/v1/audio/separate/ws",
			Handler: AudioSeparateWSHandler(serverCtx),
		},
	}

	server.AddRoutes(wsRoutes)
}
