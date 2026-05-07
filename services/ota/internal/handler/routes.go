package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/ota/internal/middleware/jwt"
	"github.com/jacklau/audio-ai-platform/services/ota/internal/svc"
	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, svcCtx *svc.ServiceContext) {
	server.AddRoutes([]rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/health",
			Handler: healthHandler(),
		},
		{
			Method:  http.MethodPost,
			Path:    "/api/v1/ota/batch-check",
			Handler: jwt.JwtMiddleware(svcCtx.Config.Auth.AccessSecret)(BatchCheckVersionHandler(svcCtx)),
		},
	})
}

func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
