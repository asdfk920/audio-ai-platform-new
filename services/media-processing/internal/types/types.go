package types

// PushAddressReq 对应：POST /api/v1/stream/push/address
type PushAddressReq struct {
	SourceType string   `json:"source_type"`
	SourceID   string   `json:"source_id"`
	Protocols  []string `json:"protocols"`
	Expires    int64    `json:"expires"` // 秒
}

type PushAddressResp struct {
	ChannelID  string   `json:"channel_id"`
	StreamKey  string   `json:"stream_key"`
	PushURL    string   `json:"push_url"`
	ExpiresIn  int64    `json:"expires_in"`
	ExpireTime string   `json:"expire_time"` // RFC3339
	Protocols  []string `json:"protocols"`
	RTMPURL    string   `json:"rtmp_url"`
	FLVURL     string   `json:"flv_url"`
}

// PushVerifyReq 对应：POST /api/v1/stream/push/verify（由 SRS/Nginx-RTMP 回调/鉴权）
type PushVerifyReq struct {
	StreamKey  string `json:"stream_key"`
	Token      string `json:"token"`
	Expire     int64  `json:"expire"` // unix 秒
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
	Timestamp  int64  `json:"timestamp"` // 生成签名时的 timestamp
}

type PushVerifyResp struct {
	Allowed bool   `json:"allowed"`
	Msg     string `json:"msg,omitempty"`
}

// PushNotifyReq 对应：POST /api/v1/stream/push/on_publish（推流开始回调）
type PushNotifyReq struct {
	StreamKey string `json:"stream_key"` // 流标识
	Token     string `json:"token"`      // 签名令牌
	Expire    int64  `json:"expire"`     // 过期时间戳
	ClientIP  string `json:"client_ip"`  // 推流客户端 IP
	ServerIP  string `json:"server_ip"`  // 流媒体服务器 IP
	Timestamp int64  `json:"timestamp"`  // 回调时间戳
}

type PushNotifyResp struct {
	Code          int    `json:"code"`                     // 状态码 0 表示允许推流
	Message       string `json:"message"`                  // 提示信息
	Allowed       bool   `json:"allowed"`                  // 是否允许推流
	ChannelID     string `json:"channel_id,omitempty"`     // 通道 ID
	MaxBitrate    int    `json:"max_bitrate,omitempty"`    // 最大允许码率
	MaxViewers    int    `json:"max_viewers,omitempty"`    // 最大允许观众数
	RecordEnabled bool   `json:"record_enabled,omitempty"` // 是否启用录制
	Reason        string `json:"reason,omitempty"`         // 拒绝原因
}

// PushUnnotifyReq 对应：POST /api/v1/stream/push/on_unpublish（推流停止回调）
type PushUnnotifyReq struct {
	StreamKey string `json:"stream_key"` // 流标识
	ChannelID string `json:"channel_id"` // 通道 ID
	ClientIP  string `json:"client_ip"`  // 推流客户端 IP
	ServerIP  string `json:"server_ip"`  // 流媒体服务器 IP
	Reason    string `json:"reason"`     // 停止原因：client_disconnect, network_error, timeout, manual, server_maintenance
	Timestamp int64  `json:"timestamp"`  // 回调时间戳
	Duration  int64  `json:"duration"`   // 推流持续时长（秒）
	BytesSent int64  `json:"bytes_sent"` // 发送的总字节数
}

type PushUnnotifyResp struct {
	Code             int     `json:"code"`                        // 状态码 0 表示成功
	Message          string  `json:"message"`                     // 提示信息
	ChannelID        string  `json:"channel_id,omitempty"`        // 通道 ID
	Processed        bool    `json:"processed"`                   // 是否已处理
	Duration         int64   `json:"duration,omitempty"`          // 推流时长（秒）
	TrafficMB        float64 `json:"traffic_mb,omitempty"`        // 流量（MB）
	NotificationSent bool    `json:"notification_sent,omitempty"` // 是否已通知下游
	Recorded         bool    `json:"recorded,omitempty"`          // 是否已录制
}

// ThirdPartyStreamUrlReq 获取第三方音频流地址请求（网易云音乐）
type ThirdPartyStreamUrlReq struct {
	ID    int64  `json:"id" form:"id"`       // 第三方平台歌曲ID（必填，如网易云音乐ID：1816904098）
	Level string `json:"level" form:"level"` // 音质等级（可选，默认standard）
	// 音质选项：
	// - standard: 标准音质（默认）
	// - higher: 较高音质
	// - exhigh: 极高音质
	// - lossless: 无损音质
	// - hires: Hi-Res音质
	// - jyeffect: 高解析度音质
	// - sky: 沉浸环绕声
	// - dolby: 杜比全景声
	// - master: 母带
}

// ThirdPartyStreamUrlResp 获取第三方音频流地址响应
type ThirdPartyStreamUrlResp struct {
	Success bool   `json:"success"` // 是否成功
	Message string `json:"message"` // 提示信息
	Data    struct {
		URL        string `json:"url"`         // 音频播放地址（MP3/FLAC等）
		Br         int    `json:"br"`          // 码率（bps）
		Size       int64  `json:"size"`        // 文件大小（字节）
		CODEc      string `json:"codec"`       // 编码格式（mp3/flac等）
		Expires    int    `json:"expires"`     // 过期时间（秒）
		Level      string `json:"level"`       // 实际返回的音质等级
		Quality    int    `json:"quality"`     // 音质评分（0-1000）
		Type       string `json:"type"`        // 文件类型
		EncodeType string `json:"encode_type"` // 编码类型
	} `json:"data"`
	RawResponse interface{} `json:"raw_response,omitempty"` // 原始响应数据（调试用）
}

// ========== 协议转换相关类型 ==========

const (
	SourceFormatMP3 = "mp3"
	SourceFormatHLS = "hls"
	SourceFormatWAV = "wav"

	OutputProtocolHTTPChunked = "http_chunked"

	DefaultChunkSize         = 32 * 1024 // 32KB
	EmbeddedDefaultChunkSize = 4 * 1024  // 4KB，嵌入式默认 HTTP 分片
)

// ProtocolConvertReq 协议转换请求
type ProtocolConvertReq struct {
	SourceURL      string `json:"source_url" form:"source_url"`           // 第三方音频源地址（必填）
	SourceFormat   string `json:"source_format" form:"source_format"`     // 源格式：mp3/hls/wav（可选，默认自动检测）
	OutputProtocol string `json:"output_protocol" form:"output_protocol"` // 仅 http_chunked；空则默认；简写 chunked / http
	ChunkSize      int    `json:"chunk_size" form:"chunk_size"`           // 分块字节（可选；embedded=1 未指定时用 EmbeddedDefaultChunkSize）
	Embedded       bool   `json:"embedded" form:"embedded"`               // 小包分片默认 4KB（弱网/MCU）
	PCMRawStrip    bool   `json:"pcm_raw" form:"pcm_raw"`                 // wav：去 RIFF/WAV 头输出裸 PCM
}

// ProtocolConvertResp 协议转换响应（仅用于错误或状态返回）
type ProtocolConvertResp struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	SessionID  string `json:"session_id,omitempty"`  // 会话ID（用于追踪）
	SourceURL  string `json:"source_url,omitempty"`  // 源地址
	Protocol   string `json:"protocol,omitempty"`    // 使用的协议
	Status     string `json:"status,omitempty"`      // 状态：streaming/completed/error
	TotalBytes int64  `json:"total_bytes,omitempty"` // 已传输字节数
	DurationMs int64  `json:"duration_ms,omitempty"` // 持续时间（毫秒）
}

// ========== DRM版权信息透传相关类型（合规核心） ==========

// DRMMetadata DRM元数据结构（从第三方获取，原样透传，不修改、不篡改）
// 核心原则：不解密、不破解、不转码，只做透传与存储密文
type DRMMetadata struct {
	DRMType     string `json:"drm_type"`     // DRM方案：widevine/fairplay/playready
	ContentID   string `json:"content_id"`   // 内容唯一ID（UUID格式）
	KeyID       string `json:"key_id"`       // 密钥ID（PSSH里的KID，Base64编码）
	PSSH        string `json:"pssh"`         // 原始PSSH盒（EME/CDM使用，Base64编码）
	LicenseURL  string `json:"license_url"`  // 许可证地址（第三方授权服务器）
	AuthToken   string `json:"auth_token"`   // 授权令牌（短期有效JWT）
	Copyright   string `json:"copyright"`    // 版权声明（如"©2026 唱片公司"）
	UsagePolicy string `json:"usage_policy"` // 使用策略（如"personal-only,exp=2026-12-31"）
	Signature   string `json:"signature"`    // 服务端RSA签名（防篡改）
}

// DRMProxyRequest DRM代理请求（设备/客户端请求加密流+DRM信息）
type DRMProxyRequest struct {
	TaskID      string `json:"task_id"`      // 下载任务ID
	SongID      int64  `json:"song_id"`      // 歌曲ID
	Quality     string `json:"quality"`      // 音质等级
	SourceURL   string `json:"source_url"`   // 第三方加密音频源地址
	UserID      int64  `json:"user_id"`      // 用户ID
	DeviceID    int64  `json:"device_id"`    // 设备ID
	EnableDRM   bool   `json:"enable_drm"`   // 是否启用DRM保护
	RequestTime string `json:"request_time"` // 请求时间戳
}

// DRMProxyResponse DRM代理响应（返回加密流+DRM头）
type DRMProxyResponse struct {
	TaskID      string      `json:"task_id"`                // 任务ID
	Status      string      `json:"status"`                 // 状态：streaming/completed/error
	Message     string      `json:"message"`                // 状态描述
	DRM         DRMMetadata `json:"drm,omitempty"`          // DRM元数据（仅在响应头中，不在body）
	FileSize    int64       `json:"file_size,omitempty"`    // 文件大小（字节）
	DurationMs  int64       `json:"duration_ms,omitempty"`  // 持续时间（毫秒）
	CompletedAt string      `json:"completed_at,omitempty"` // 完成时间
}

// ========== 精简版DRM透传播放/下载接口（最简合规实现） ==========

// PlaySongWithDRMReq 播放/下载歌曲请求
// GET /api/v1/song/play?song_id=xxx&quality=hq
type PlaySongWithDRMReq struct {
	SongID  int64  `json:"song_id" form:"song_id"`       // 歌曲ID（必填）
	Quality string `json:"quality" form:"quality"`       // 音质：standard/hq/hires（可选，默认hq）
	Token   string `json:"token,omitempty" form:"token"` // DRM/版权令牌（测试阶段原样透传；也可来自 Authorization）
}

// PlaySongWithDRMResp 播放/下载歌曲响应（原样透传加密流+DRM信息）
// 核心原则：不修改、不解密、不处理，直接返回第三方原始数据
type PlaySongWithDRMResp struct {
	Success     bool        `json:"success"`                // 是否成功
	Message     string      `json:"message"`                // 状态描述
	SongID      int64       `json:"song_id"`                // 歌曲ID
	SongName    string      `json:"song_name,omitempty"`    // 歌曲名称
	StreamURL   string      `json:"stream_url"`             // 加密的音频流地址（原样返回，不解密）
	DRMInfo     DRMMetadata `json:"drm_info"`               // DRM/版权信息（原样透传）
	DRMToken    string      `json:"drm_token,omitempty"`    // DRM/版权令牌（原样透传）
	Copyright   string      `json:"copyright"`              // 版权声明（原样透传）
	Encrypted   bool        `json:"encrypted"`              // 标记是否为加密流（始终为true）
	EncryptType string      `json:"encrypt_type,omitempty"` // 加密类型标记（如 AES-128-DRM）
	Quality     string      `json:"quality"`                // 音质等级
}
