package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/media-processing/internal/svc"
	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, ctx *svc.ServiceContext) {
	// JWT 保护（网关透传 Authorization: Bearer）
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/api/v1/stream/push/address",
				Handler: PushAddressHandler(ctx),
			},
		},
		rest.WithJwt(ctx.Config.Auth.AccessSecret),
	)

	// 流媒体服务器回调鉴权：通常不带用户 JWT（由 token/expire 参数鉴权）
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/api/v1/stream/push/verify",
				Handler: PushVerifyHandler(ctx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/v1/stream/push/on_publish",
				Handler: PushNotifyHandler(ctx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/v1/stream/push/on_unpublish",
				Handler: PushUnnotifyHandler(ctx),
			},
		},
	)
}
