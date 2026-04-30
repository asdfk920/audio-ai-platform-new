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
