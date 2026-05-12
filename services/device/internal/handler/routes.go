// Package handler 注册 HTTP 路由（探活、设备定时状态上报等）。
package handler

import (
	"net/http"

	"github.com/jacklau/audio-ai-platform/services/device/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

// RegisterHandlers 注册探活、POST /api/device/status/report（设备 JWT 或 X-Device-Secret）、音频私有格式接口、设备注册接口。
func RegisterHandlers(server *rest.Server, svcCtx *svc.ServiceContext) {
	routes := []rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/health",
			Handler: healthHandler(),
		},
	}
	if svcCtx != nil {
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/status/report",
			Handler: statusReportHandler(svcCtx),
		})
		// 设备注册接口（设备首次联网注册，获取认证 token）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/register",
			Handler: deviceRegisterHandler(svcCtx),
		})
		// 设备认证接口（设备使用 token 向云端认证身份）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/auth",
			Handler: deviceAuthHandler(svcCtx),
		})
		// ⭐ MQTT Broker 设备认证接口（EMQX HTTP认证回调）
		// POST /mqtt/auth
		// 用途：EMQX Broker 调用此接口验证设备 MQTT 连接身份
		// 触发时机：设备发起 MQTT CONNECT 连接时
		// 注意：此接口无需 JWT 鉴权（由 EMQX 内部调用）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/mqtt/auth",
			Handler: MqttAuthHandler(svcCtx),
		})
		// ⭐ MQTT Broker ACL 鉴权接口（EMQX HTTP授权回调）
		// POST /mqtt/acl
		// 用途：EMQX Broker 调用此接口验证设备 Topic 发布/订阅权限
		// 触发时机：设备发起 MQTT PUBLISH 或 SUBSCRIBE 操作时
		// 注意：此接口无需 JWT 鉴权（由 EMQX 内部调用）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/mqtt/acl",
			Handler: MqttACLHandler(svcCtx),
		})
		// ⭐ MQTT Broker 设备断开连接接口（EMQX Webhook）
		// POST /mqtt/disconnect
		// 用途：EMQX Broker 在设备断开时调用，更新设备离线状态
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/mqtt/disconnect",
			Handler: MqttDisconnectHandler(svcCtx),
		})
		// ⭐ MQTT Broker 连接事件接口（EMQX Webhook - client.connected/client.disconnected）
		// POST /mqtt/event/connection
		// 用途：EMQX Broker 推送设备连接/断开事件，更新在线状态和影子数据
		// 触发时机：设备建立/断开MQTT连接时
		// 注意：此接口无需 JWT 鉴权（由 EMQX 内部调用）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/mqtt/event/connection",
			Handler: MqttConnectionEventHandler(svcCtx),
		})
		// ⭐ MQTT 设备WILL消息接口（设备异常断开通知）
		// POST /mqtt/event/will
		// 用途：接收EMQX推送的设备遗嘱消息，更新离线状态和同步影子数据
		// 触发时机：设备异常断开时（断电/断网/崩溃），EMQX自动发布WILL消息
		// 注意：此接口无需 JWT 鉴权（由 EMQX 内部调用）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/mqtt/event/will",
			Handler: WillMessageHandler(svcCtx),
		})
		// 设备重启指令接口（用户通过 App 下发重启指令）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/cmd/reboot",
			Handler: deviceRebootHandler(svcCtx),
		})
		// 设备绑定接口（用户通过 App 将设备绑定到当前登录账户）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/bind",
			Handler: deviceBindHandler(svcCtx),
		})
		// 设备状态更新接口（设备通过 HTTP 接口主动上报在线状态）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/status/update",
			Handler: deviceStatusUpdateHandler(svcCtx),
		})
		// 设备列表查询接口（用户查询已绑定设备列表）
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/api/device/list",
			Handler: deviceListHandler(svcCtx),
		})
		// 设备详情查询接口（用户查看指定设备详细信息）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/detail",
			Handler: deviceDetailHandler(svcCtx),
		})
		// 设备影子定时上报接口（设备定时采集状态数据上报云端）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/shadow/report",
			Handler: deviceShadowReportHandler(svcCtx),
		})
		// 设备日志上报接口（设备通过 HTTP POST 请求上报运行日志）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/log",
			Handler: deviceLogReportHandler(svcCtx),
		})
		// 设备影子查询接口（用户查询指定设备最新状态数据，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/api/device/shadow",
			Handler: deviceShadowQueryHandler(svcCtx),
		})
		// 设备位置查询接口（用户查询指定设备最新 UWB 定位数据，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/api/device/location",
			Handler: deviceLocationQueryHandler(svcCtx),
		})
		// 设备播放指令接口（用户通过 App 下发播放指令，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/cmd/play",
			Handler: devicePlayHandler(svcCtx),
		})
		// 设备暂停指令接口（用户通过 App 下发暂停指令，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/cmd/pause",
			Handler: devicePauseHandler(svcCtx),
		})
		// 设备继续播放指令接口（用户通过 App 下发继续播放指令，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/cmd/resume",
			Handler: deviceResumeHandler(svcCtx),
		})
		// 设备下一首指令接口（用户通过 App 下发下一首指令，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/cmd/next",
			Handler: deviceNextHandler(svcCtx),
		})
		// 设备上一首指令接口（用户通过 App 下发上一首指令，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/cmd/prev",
			Handler: devicePrevHandler(svcCtx),
		})
		// 设备音量加指令接口（用户通过 App 下发音量加指令，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/cmd/volume_up",
			Handler: deviceVolumeUpHandler(svcCtx),
		})
		// 设备音量减指令接口（用户通过 App 下发音量减指令，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/cmd/volume_down",
			Handler: deviceVolumeDownHandler(svcCtx),
		})
		// 设备设置循环播放指令接口（用户通过 App 下发设置循环播放指令，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/cmd/set_loop",
			Handler: deviceSetLoopHandler(svcCtx),
		})
		// 设备设置随机播放指令接口（用户通过 App 下发设置随机播放指令，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/cmd/set_shuffle",
			Handler: deviceSetShuffleHandler(svcCtx),
		})
		// 设备播放歌单指令接口（用户通过 App 下发播放歌单指令，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/cmd/play_playlist",
			Handler: devicePlayPlaylistHandler(svcCtx),
		})
		// 设备播放状态查询接口（用户通过 App 查询设备播放状态，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/api/device/status/playback",
			Handler: devicePlaybackStatusHandler(svcCtx),
		})
		// 设备播放进度查询接口（用户通过 App 查询设备当前播放进度，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/api/device/status/progress",
			Handler: devicePlaybackProgressHandler(svcCtx),
		})
		// 设备远程诊断接口（用户通过 App 发起设备远程诊断，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/diagnose",
			Handler: deviceDiagnoseHandler(svcCtx),
		})
	}
	server.AddRoutes(routes)
}

func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
