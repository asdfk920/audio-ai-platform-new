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
		{
			// ⭐ WebSocket 音轨分离（公开接口，自行处理认证）
			// GET /api/v1/audio/separate/ws
			// 重要：必须在这里注册，否则 go-zero 会返回 404！
			Method:  http.MethodGet,
			Path:    "/api/v1/audio/separate/ws",
			Handler: AudioSeparateWSHandler(serverCtx),
		},
	}

	server.AddRoutes(publicRoutes)

	// JWT 保护的接口（需要登录）
	jwtProtected := []rest.Route{
		{
			// ⭐ 上传音频文件到 OSS
			// POST /api/v1/audio/upload
			// 用户上传音频文件到对象存储，返回 URL 用于音轨分离
			Method:  http.MethodPost,
			Path:    "/audio/upload",
			Handler: CORSMiddleware(AudioUploadHandler(serverCtx)),
		},
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
		{
			// 查询单个分离任务详情（带权限验证）
			// GET /api/v1/inference/tasks/{task_id}
			// 必须登录，且只能查询自己的任务
			Method:  http.MethodGet,
			Path:    "/inference/tasks/:task_id",
			Handler: TaskDetailHandler(serverCtx),
		},
		{
			// 删除历史分离任务（带权限验证和级联清理）
			// DELETE /api/v1/inference/tasks/{task_id}
			// 必须登录，且只能删除自己的任务
			// 会同时删除关联的音轨记录
			Method:  http.MethodDelete,
			Path:    "/inference/tasks/:task_id",
			Handler: TaskDeleteHandler(serverCtx),
		},
		{
			// 删除历史分离任务（兼容查询参数格式）
			// DELETE /api/v1/inference/tasks?task_id={id}
			// 支持旧版客户端或特殊场景
			Method:  http.MethodDelete,
			Path:    "/inference/tasks",
			Handler: TaskDeleteHandler(serverCtx),
		},
		{
			// 取消分离任务（带权限验证和状态检查）
			// POST /api/v1/inference/tasks/{task_id}/cancel
			// 只能取消 pending 或 processing 状态的任务
			// 通过 WebSocket 实时通知用户取消结果
			Method:  http.MethodPost,
			Path:    "/inference/tasks/:task_id/cancel",
			Handler: TaskCancelHandler(serverCtx),
		},
		{
			// 取消分离任务（兼容查询参数格式）
			// POST /api/v1/inference/tasks/cancel?id=2
			// 支持通过主键 ID 或 task_id 取消
			Method:  http.MethodPost,
			Path:    "/inference/tasks/cancel",
			Handler: TaskCancelHandler(serverCtx),
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

	// ⚠️ 注意：WebSocket 路由已在 ai-worker.go 中单独注册
	// go-zero 的 REST 路由引擎无法正确处理 WebSocket 升级协议
	// 因此 WebSocket 路由使用自定义 Handler 包装器注册（见 ai-worker.go）
}
