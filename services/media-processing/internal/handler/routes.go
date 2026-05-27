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
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/thirdparty/stream/url",
				Handler: ThirdPartyStreamUrlHandler(ctx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/thirdparty/stream/play",
				Handler: ThirdPartyStreamProxyHandler(ctx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/protocol/convert",
				Handler: ProtocolConvertHandler(ctx),
			},
			// 播放/下载歌曲接口（DRM透传版本，原样返回加密流+DRM信息）
			{
				Method:  http.MethodGet,
				Path:    "/api/v1/song/play",
				Handler: PlaySongWithDRMHandler(ctx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/api/v1/song/play",
				Handler: PlaySongWithDRMHandler(ctx),
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

	// 第三方登录接口（公开，无需 JWT）
	// 网易云音乐 IoT 开放平台 - 二维码登录
	server.AddRoutes(
		[]rest.Route{
			// 获取二维码 Key（POST 请求）
			{
				Method:  http.MethodPost,
				Path:    "/openapi/music/basic/user/oauth2/qrcodekey/get/v2",
				Handler: GetQRCodeKeyHandler(ctx),
			},
			// 轮询查询二维码状态（POST 请求）
			{
				Method:  http.MethodPost,
				Path:    "/openapi/music/basic/oauth2/device/login/qrcode/get",
				Handler: CheckQRCodeStatusHandler(ctx),
			},
			// 二维码登录回调（备用）
			{
				Method:  http.MethodPost,
				Path:    "/openapi/music/basic/oauth2/qrcode/callback",
				Handler: QRCodeCallbackHandler(ctx),
			},
		},
	)
}
