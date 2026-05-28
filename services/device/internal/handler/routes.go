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
		// 设备注册接口（设备首次联网时向云端注册身份）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/register",
			Handler: deviceRegisterHandler(svcCtx),
		})
		// 设备认证接口（设备使用SN+密钥获取JWT Token，用于后续API调用）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/auth",
			Handler: deviceAuthHandler(svcCtx),
		})
		// 设备WebSocket长连接接口（设备建立实时双向通信通道）
		// 流程：携带JWT Token → 握手前校验 → 升级WebSocket → 首包签名认证 → 建立长连接
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/ws/device",
			Handler: DeviceWsHandler(svcCtx),
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
		// 设备影子详情查询接口（用于设备详情页展示完整影子数据）
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/api/device/shadow/detail",
			Handler: DeviceShadowDetailHandler(svcCtx),
		})
		// 设备影子定时上报接口（设备定时采集状态数据上报云端）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/shadow/report",
			Handler: deviceShadowReportHandler(svcCtx),
		})
		// 设备影子批量更新（管理端/平台：可含 desired，需 expect_version）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/shadow/batch",
			Handler: DeviceShadowBatchHandler(svcCtx),
		})
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/v1/device/shadow/batch",
			Handler: DeviceShadowBatchHandler(svcCtx),
		})
		// 网关批量上报子设备真实状态（仅 reported，禁止 desired）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/shadow/batch-report",
			Handler: DeviceShadowBatchReportHandler(svcCtx),
		})
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/v1/device/shadow/batch-report",
			Handler: DeviceShadowBatchReportHandler(svcCtx),
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
		// 点播 / URL 播放（与 WS 链路一致：command_code=play_audio）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/cmd/play_audio",
			Handler: devicePlayAudioHandler(svcCtx),
		})
		// OpenAPI/BasePath /api/v1 下的别名路径（与同 handler 完全一致）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/v1/device/play/audio",
			Handler: devicePlayAudioHandler(svcCtx),
		})
		// 进度条跳转（command_code=seek）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/cmd/seek",
			Handler: deviceSeekHandler(svcCtx),
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
		// 设备音量调节统一接口（用户通过 App 下发音量调节指令，支持直接设置目标音量0-100，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/cmd/volume",
			Handler: deviceVolumeHandler(svcCtx),
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
		// 歌曲下载接口（用户通过前端发起下载请求，后端下发指令到设备，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/v1/song/download",
			Handler: SongDownloadHandler(svcCtx),
		})

		// ========== 设备下载完整流程接口（端口8002） ==========
		// 设备下载接口（用户点击下载按钮，创建记录并下发指令到设备，需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/v1/device/download",
			Handler: DeviceDownloadHandler(svcCtx),
		})
		// 设备下载回调接口（设备下载完成后上报结果）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/v1/device/download/callback",
			Handler: DeviceDownloadCallbackHandler(svcCtx),
		})
		// 查询设备下载状态（需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/api/v1/device/download/status",
			Handler: DeviceDownloadStatusHandler(svcCtx),
		})
		// 设备下载历史列表（需 JWT 鉴权）
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/api/v1/device/downloads",
			Handler: DeviceDownloadListHandler(svcCtx),
		})

		// ========== 设备影子 V2 接口（Redis Hash 实现） ==========
		shadowV2Handler := NewShadowV2Handler(svcCtx)

		// 初始化设备影子
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/shadow/v2/init",
			Handler: shadowV2Handler.InitShadow,
		})
		// 更新设备上报状态（reported）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/shadow/v2/reported",
			Handler: shadowV2Handler.UpdateReported,
		})
		// 更新平台期望状态（desired）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/shadow/v2/desired",
			Handler: shadowV2Handler.UpdateDesired,
		})
		// 查询完整设备影子
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/api/device/shadow/v2/query",
			Handler: shadowV2Handler.QueryShadow,
		})
		// 查询设备影子（Apifox / 开放平台路径：v2 优先，v1 回退）
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/api/v2/device/shadow",
			Handler: DeviceShadowV2QueryHandler(svcCtx),
		})
		// 仅查询 reported 状态
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/api/device/shadow/v2/reported",
			Handler: shadowV2Handler.QueryReportedOnly,
		})
		// 仅查询版本号
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/api/device/shadow/v2/version",
			Handler: shadowV2Handler.QueryVersionOnly,
		})
		// 更新在线状态
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/shadow/v2/status",
			Handler: shadowV2Handler.UpdateStatus,
		})
		// CAS 原子更新
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/shadow/v2/cas",
			Handler: shadowV2Handler.CASUpdate,
		})

		// ========== Admin 后台接口（设备影子管理） ==========
		adminShadowHandler := NewAdminShadowHandler(svcCtx)

		// 获取设备影子完整详情
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/api/admin/device/shadow/detail",
			Handler: adminShadowHandler.GetDeviceShadowDetail,
		})
		// 批量查询设备影子列表
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/admin/device/shadow/list",
			Handler: adminShadowHandler.ListDeviceShadows,
		})
		// 获取设备影子统计信息
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/api/admin/device/shadow/stats",
			Handler: adminShadowHandler.GetDeviceShadowStats,
		})
		// 删除设备影子（谨慎使用）
		routes = append(routes, rest.Route{
			Method:  http.MethodDelete,
			Path:    "/api/admin/device/shadow",
			Handler: adminShadowHandler.DeleteDeviceShadow,
		})
		// 批量更新设备影子（v1 Redis Hash + PostgreSQL，与 /api/device/shadow/batch 相同语义）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/admin/device/shadow/batch",
			Handler: DeviceShadowBatchHandler(svcCtx),
		})

		// ========== 诊断指令下发接口 ==========
		// 用户通过App/后台发起设备日志收集等诊断指令（需JWT鉴权，WebSocket下发）
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/device/diagnosis/command",
			Handler: DiagnosisCommandHandler(svcCtx),
		})
		// OpenAPI/BasePath /api/v1 下的别名路径
		routes = append(routes, rest.Route{
			Method:  http.MethodPost,
			Path:    "/api/v1/device/diagnosis/command",
			Handler: DiagnosisCommandHandler(svcCtx),
		})

		// ========== 设备状态 WebSocket 订阅接口 ==========
		// App 用户订阅设备状态变更推送（支持批量订阅/取消，实时推送）
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/ws/device/subscribe",
			Handler: UserWsHandler(svcCtx),
		})
		// 兼容旧路径：用户 WebSocket 长连接
		routes = append(routes, rest.Route{
			Method:  http.MethodGet,
			Path:    "/ws/user",
			Handler: UserWsHandler(svcCtx),
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
