package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/content/internal/svc"
	"github.com/zeromicro/go-zero/rest"
)

// RegisterHandlers 注册内容服务接口
func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: contentListHandler(serverCtx),
			},
			{
				// 我的歌单列表（须在 /:id 之前注册，避免 playlists 被当成内容 ID）
				Method:  http.MethodGet,
				Path:    "/playlists",
				Handler: contentPlaylistListHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/playlists",
				Handler: contentPlaylistCreateHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/playlists/:id",
				Handler: contentPlaylistUpdateHandler(serverCtx),
			},
			{
				Method:  http.MethodDelete,
				Path:    "/playlists/:id",
				Handler: contentPlaylistDeleteHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/playlists/:id/songs",
				Handler: contentPlaylistAddSongHandler(serverCtx),
			},
			{
				Method:  http.MethodDelete,
				Path:    "/playlists/:id/songs/:songId",
				Handler: contentPlaylistRemoveSongHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/:id/like",
				Handler: contentLikeHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/:id/favorite",
				Handler: contentFavoriteHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/:id",
				Handler: contentDetailHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/subscribe",
				Handler: contentSubscribeHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/unsubscribe",
				Handler: contentUnsubscribeHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/artists/:id/subscribe",
				Handler: artistSubscribeHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/artists/:id/unsubscribe",
				Handler: artistUnsubscribeHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/artists/:id/subscription-status",
				Handler: artistSubscriptionStatusHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/my/subscriptions/artists",
				Handler: myArtistSubscriptionsHandler(serverCtx),
			},
		},
		rest.WithPrefix("/api/v1/content"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/messages",
				Handler: messageListHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/messages/read",
				Handler: messageMarkReadHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/messages/unread-count",
				Handler: messageUnreadCountHandler(serverCtx),
			},
		},
		rest.WithPrefix("/api/v1/user"),
	)
}
