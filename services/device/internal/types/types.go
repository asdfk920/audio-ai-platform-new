package types

import "time"

// EnumItem 枚举项
// 用于下拉选项、列表等场景
type EnumItem struct {
	Label string `json:"label"` // 显示文本
	Value string `json:"value"` // 实际值
}

// DeviceRegisterReq 设备注册请求（预录入+首次激活模式）
// 设备生产时SN和密钥已预录入云端（状态：未激活），设备首次联网时触发注册/激活流程
// 流程：参数校验 → 查询预录入设备 → 密钥验证 → 状态检查 → 激活记录
type DeviceRegisterReq struct {
	Sn              string `json:"sn" validate:"required"`            // 设备序列号（16位字母数字组合，从Flash/OTP读取）
	DeviceSecret    string `json:"device_secret" validate:"required"` // 设备密钥（生产阶段预烧录，用于身份验证）
	FirmwareVersion string `json:"firmware_version,omitempty"`        // 固件版本号（可选，用于记录设备当前固件版本）
	HardwareVersion string `json:"hardware_version,omitempty"`        // 硬件版本号（可选，用于记录设备硬件型号）
	Mac             string `json:"mac,omitempty"`                     // MAC地址（可选，用于设备识别和定位）
}

// DeviceRegisterResp 设备注册响应（激活模式）
// 注册成功后不返回token，设备需调用 /api/device/auth 接口获取访问凭证
type DeviceRegisterResp struct {
	Sn          string `json:"sn"`           // 设备序列号
	DeviceID    int64  `json:"device_id"`    // 设备ID（云端分配的唯一标识）
	Status      string `json:"status"`       // 激活状态：activated-已激活, already_activated-已是激活状态
	ActivatedAt string `json:"activated_at"` // 激活时间（ISO8601格式，首次激活或上次激活时间）
	Message     string `json:"message"`      // 提示信息（成功原因说明）
}

// DeviceAuthReq 设备认证请求（密钥模式）
// 设备使用 SN + device_secret 向云端获取访问凭证（JWT Token）
// 流程：参数校验 → 设备查询 → 密钥验证 → 状态检查 → 生成Token
type DeviceAuthReq struct {
	Sn           string `json:"sn" validate:"required"`            // 设备序列号（16位字母数字组合）
	DeviceSecret string `json:"device_secret" validate:"required"` // 设备密钥（生产阶段预烧录，用于身份验证）
}

// DeviceAuthResp 设备认证响应（返回JWT Token）
// 返回设备访问凭证、有效期、设备基础信息，设备端保存Token用于后续API调用
type DeviceAuthResp struct {
	Token           string `json:"token"`            // 访问凭证（JWT Token，用于后续API调用和WebSocket连接）
	ExpiresIn       int64  `json:"expires_in"`       // 凭证有效期（秒），默认2592000秒（30天）
	DeviceID        int64  `json:"device_id"`        // 设备ID（云端分配的唯一标识）
	Sn              string `json:"sn"`               // 设备序列号
	Model           string `json:"model"`            // 设备型号
	FirmwareVersion string `json:"firmware_version"` // 固件版本号
	HardwareVersion string `json:"hardware_version"` // 硬件版本号
	OnlineStatus    int16  `json:"online_status"`    // 在线状态：0-离线, 1-在线
	Status          int16  `json:"status"`           // 设备状态：1-正常, 2-禁用, 3-停用, 4-未注册, 5-未认证
	Message         string `json:"message"`          // 提示信息（成功原因说明）
}

// WsAuthMessage WebSocket认证消息
// 设备建立 WebSocket 连接后须立即发送的首包 JSON，用于身份认证。
//
// 握手：JWT 放在请求头 Authorization: Bearer <token>；若客户端无法自定义握手头，可使用 URL ?token= / ?access_token=。
//
// 签名算法（与 POST /api/device/register 一致）：signData = sn + register_timestamp（字符串拼接，register_timestamp 为注册时生成的毫秒 Unix 时间戳）；
// signature = HMAC-SHA256(key=device_secret 明文, data=signData) 的 hex 小写。
//
// 安全设计：
//   - Token通过HTTP请求头传递，避免在消息体中暴露
//   - 使用客户端发送的时间戳进行签名验证
//   - 支持设备时间与服务端时间存在一定偏差的情况
type WsAuthMessage struct {
	Type      string `json:"type"`      // 消息类型，固定为 "auth"
	Sn        string `json:"sn"`        // 设备序列号（16位），须与 JWT 内 sn 一致（忽略大小写）
	Timestamp int64  `json:"timestamp"` // 客户端时间戳（毫秒级Unix时间戳，用于签名计算）
	Signature string `json:"signature"` // HMAC-SHA256 hex（signData = sn + timestamp）
}

// WsAuthResponse WebSocket认证响应
// 云端返回给设备的认证结果。失败时若已通过 JWT 校验或已定位设备行，可能携带非零 device_id 便于排查。
type WsAuthResponse struct {
	Type      string `json:"type"`       // 响应类型："auth_response"
	Success   bool   `json:"success"`    // 是否认证成功
	Message   string `json:"message"`    // 提示信息
	DeviceID  int64  `json:"device_id"`  // 设备ID（认证成功时返回）
	ExpiresIn int64  `json:"expires_in"` // Token剩余有效时间（秒）
}

// DeviceBindReq 设备绑定请求
// 用户将设备绑定到自己的账户下
type DeviceBindReq struct {
	Sn string `json:"sn" validate:"required"` // 设备序列号，16位字母数字组合
}

// DeviceBindResp 设备绑定响应
// 返回绑定结果
type DeviceBindResp struct {
	Sn      string `json:"sn"`       // 设备序列号
	BoundAt string `json:"bound_at"` // 绑定时间
}

// DeviceDetailReq 设备详情查询请求
// 用户查询设备的详细信息
type DeviceDetailReq struct {
	Sn string `json:"sn" validate:"required"` // 设备序列号，16位字母数字组合
}

// DeviceShadowInfo 设备影子信息
// 包含设备实时状态数据
type DeviceShadowInfo struct {
	Battery         int    `json:"battery"`          // 电量百分比
	RunState        string `json:"run_state"`        // 运行状态
	LastActiveMs    int64  `json:"last_active_ms"`   // 最后活跃时间（毫秒时间戳）
	FirmwareVersion string `json:"firmware_version"` // 固件版本号
	IP              string `json:"ip"`               // 设备 IP 地址
}

// DeviceDetailResp 设备详情查询响应
// 返回设备详细信息
type DeviceDetailResp struct {
	ID              int64            `json:"id"`               // 设备 ID
	Sn              string           `json:"sn"`               // 设备序列号
	Model           string           `json:"model"`            // 设备型号
	FirmwareVersion string           `json:"firmware_version"` // 固件版本号
	OnlineStatus    int              `json:"online_status"`    // 在线状态：1-在线、0-离线
	BoundAt         string           `json:"bound_at"`         // 绑定时间
	IsBound         bool             `json:"is_bound"`         // 是否已绑定当前用户
	Shadow          DeviceShadowInfo `json:"shadow"`           // 设备影子信息
}

// DeviceListItem 设备列表项
// 用于设备列表展示
type DeviceListItem struct {
	ID              int64  `json:"id"`               // 设备 ID
	Sn              string `json:"sn"`               // 设备序列号
	Model           string `json:"model"`            // 设备型号
	FirmwareVersion string `json:"firmware_version"` // 固件版本号
	OnlineStatus    int    `json:"online_status"`    // 在线状态：1-在线、0-离线
	BoundAt         string `json:"bound_at"`         // 绑定时间
	Battery         int    `json:"battery"`          // 电量百分比
	RunState        string `json:"run_state"`        // 运行状态
}

// DeviceListResp 设备列表查询响应
// 返回用户绑定的设备列表
type DeviceListResp struct {
	Total int              `json:"total"` // 设备总数
	List  []DeviceListItem `json:"list"`  // 设备列表
}

// DeviceRebootReq 设备重启指令请求
// 用户通过 App 向设备下发重启指令
type DeviceRebootReq struct {
	Sn     string `json:"sn" validate:"required"`     // 设备序列号，16位字母数字组合
	Action string `json:"action" validate:"required"` // 操作类型：reboot
}

// DeviceRebootResp 设备重启指令响应
// 返回指令下发结果
type DeviceRebootResp struct {
	InstructionID int64  `json:"instruction_id"` // 指令 ID
	Status        string `json:"status"`         // 指令状态：delivered-已下发, cached-已缓存
	Message       string `json:"message"`        // 提示信息
}

// DeviceStatusUpdateReq 设备状态更新请求
// 设备上报自身状态变化
type DeviceStatusUpdateReq struct {
	Sn           string `json:"sn" validate:"required"`            // 设备序列号，16位字母数字组合
	OnlineStatus int    `json:"online_status" validate:"required"` // 在线状态：1-在线、0-离线
}

// DeviceStatusUpdateResp 设备状态更新响应
// 返回更新结果
type DeviceStatusUpdateResp struct {
	Sn           string `json:"sn"`            // 设备序列号
	OnlineStatus int    `json:"online_status"` // 在线状态
	UpdatedAt    string `json:"updated_at"`    // 更新时间
}

// DeviceShadowReportResp 设备影子上报响应
// 返回上报结果
type DeviceShadowReportResp struct {
	Sn        string `json:"sn"`         // 设备序列号
	UpdatedAt string `json:"updated_at"` // 更新时间
	Message   string `json:"message"`    // 提示信息
}

// UWBPosition UWB定位数据
// 包含三维坐标位置信息
type UWBPosition struct {
	X *float64 `json:"x,omitempty"` // X坐标（米）
	Y *float64 `json:"y,omitempty"` // Y坐标（米）
	Z *float64 `json:"z,omitempty"` // Z坐标（米）
}

// AcousticCalib 声学校准参数
// 包含声学校准状态和偏移量
type AcousticCalib struct {
	Calibrated *int     `json:"calibrated,omitempty"` // 是否已校准：0-未校准、1-已校准
	Offset     *float64 `json:"offset,omitempty"`     // 校准偏移量
}

// DevicePlayReq 设备播放指令请求
// 用户通过 App 向设备下发播放指令
type DevicePlayReq struct {
	Sn       string `json:"sn" validate:"required"`     // 设备序列号，16位字母数字组合
	Action   string `json:"action" validate:"required"` // 操作类型：play/pause/stop/next/prev
	MediaURL string `json:"media_url"`                  // 媒体资源 URL
	Volume   int    `json:"volume"`                     // 音量 0-100
}

// DevicePlayResp 设备播放指令响应
// 返回指令下发结果
type DevicePlayResp struct {
	InstructionID int64  `json:"instruction_id"` // 指令 ID
	Status        string `json:"status"`         // 指令状态：delivered-已下发, cached-已缓存
	Message       string `json:"message"`        // 提示信息
}

// DevicePauseReq 设备暂停指令请求
// 用户通过 App 向设备下发暂停指令
type DevicePauseReq struct {
	Sn     string `json:"sn" validate:"required"`     // 设备序列号，16位字母数字组合
	Action string `json:"action" validate:"required"` // 操作类型：pause
}

// DevicePauseResp 设备暂停指令响应
// 返回指令下发结果
type DevicePauseResp struct {
	InstructionID int64  `json:"instruction_id"` // 指令 ID
	Status        string `json:"status"`         // 指令状态：delivered-已下发, cached-已缓存
	Message       string `json:"message"`        // 提示信息
}

// DevicePlayAudioReq 点播/URL 播放（与 media-processing 点播参数对齐）
// WebSocket 下行与暂停等指令一致：type=cmd，command_code=play_audio，payload 中包含以下业务字段。
type DevicePlayAudioReq struct {
	Sn        string  `json:"sn"`                  // 设备序列号（16 位）；与 device_sn 二选一
	DeviceSn  string  `json:"device_sn,omitempty"` // 同上，兼容 Apifox 等使用 device_sn 的客户端
	ContentID int64   `json:"content_id"`          // 内容 ID（正整数）
	AudioURL  string  `json:"audio_url"`           // 可播放音频 URL
	StartPos  float64 `json:"start_pos,omitempty"` // 起始进度（秒）
	Volume    int     `json:"volume,omitempty"`    // 音量 0–100（0 或未传服务端按默认处理）
	PlayMode  string  `json:"play_mode,omitempty"` // sequential/list_loop/single_loop/random
}

// DevicePlayAudioResp 点播指令响应
type DevicePlayAudioResp struct {
	TaskID        string `json:"task_id"`        // 与设备侧载荷中的 task_id 一致，便于对账
	InstructionID int64  `json:"instruction_id"` // 指令 ID
	Status        string `json:"status"`         // delivered / queued / cached 等
	Message       string `json:"message"`        // 人类可读提示
}

// DeviceSeekReq 进度条跳转（与 media-processing seek 对齐）
type DeviceSeekReq struct {
	Sn       string  `json:"sn"`                // 设备序列号（16 位）
	Position float64 `json:"position"`          // 跳转目标秒数
	TaskID   string  `json:"task_id,omitempty"` // 可选：与当前点播任务对齐
}

// DeviceSeekResp 进度跳转指令响应
type DeviceSeekResp struct {
	TaskID        string `json:"task_id"`
	InstructionID int64  `json:"instruction_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

// DeviceResumeReq 设备继续播放指令请求
// 用户通过 App 向设备下发继续播放指令
type DeviceResumeReq struct {
	Sn     string `json:"sn" validate:"required"`     // 设备序列号，16位字母数字组合
	Action string `json:"action" validate:"required"` // 操作类型：resume
	Volume int    `json:"volume"`                     // 音量 0-100
}

// DeviceResumeResp 设备继续播放指令响应
// 返回指令下发结果
type DeviceResumeResp struct {
	InstructionID int64  `json:"instruction_id"` // 指令 ID
	Status        string `json:"status"`         // 指令状态：delivered-已下发, cached-已缓存
	Message       string `json:"message"`        // 提示信息
}

// DeviceNextReq 设备下一首指令请求
// 用户通过 App 向设备下发下一首指令
type DeviceNextReq struct {
	Sn     string `json:"sn" validate:"required"`     // 设备序列号，16位字母数字组合
	Action string `json:"action" validate:"required"` // 操作类型：next
}

// DeviceNextResp 设备下一首指令响应
// 返回指令下发结果
type DeviceNextResp struct {
	InstructionID int64  `json:"instruction_id"` // 指令 ID
	Status        string `json:"status"`         // 指令状态：delivered-已下发, cached-已缓存
	Message       string `json:"message"`        // 提示信息
}

// DevicePrevReq 设备上一首指令请求
// 用户通过 App 向设备下发上一首指令
type DevicePrevReq struct {
	Sn     string `json:"sn" validate:"required"`     // 设备序列号，16位字母数字组合
	Action string `json:"action" validate:"required"` // 操作类型：prev
}

// DevicePrevResp 设备上一首指令响应
// 返回指令下发结果
type DevicePrevResp struct {
	InstructionID int64  `json:"instruction_id"` // 指令 ID
	Status        string `json:"status"`         // 指令状态：delivered-已下发, cached-已缓存
	Message       string `json:"message"`        // 提示信息
}

// DeviceVolumeUpReq 设备音量加指令请求
// 用户通过 App 向设备下发音量加指令
type DeviceVolumeUpReq struct {
	Sn     string               `json:"sn" validate:"required"`     // 设备序列号，16位字母数字组合
	Action string               `json:"action" validate:"required"` // 操作类型：volume_up
	Params DeviceVolumeUpParams `json:"params"`                     // 音量加参数
	Step   int                  `json:"step"`                       // 音量增量，默认5，范围0-20
}

// DeviceVolumeUpParams 设备音量加参数
// 包含音量增量等配置参数
type DeviceVolumeUpParams struct {
	Step int `json:"step"` // 音量增量，默认5，范围0-20
}

// DeviceVolumeUpResp 设备音量加指令响应
// 返回指令下发结果
type DeviceVolumeUpResp struct {
	InstructionID int64  `json:"instruction_id"` // 指令 ID
	Status        string `json:"status"`         // 指令状态：delivered-已下发, cached-已缓存
	Message       string `json:"message"`        // 提示信息
	TargetVolume  int    `json:"target_volume"`  // 目标音量值
	CurrentVolume int    `json:"current_volume"` // 当前音量值
}

// DeviceVolumeDownReq 设备音量减指令请求
// 用户通过 App 向设备下发音量减指令
type DeviceVolumeDownReq struct {
	Sn     string                 `json:"sn" validate:"required"`     // 设备序列号，16位字母数字组合
	Action string                 `json:"action" validate:"required"` // 操作类型：volume_down
	Params DeviceVolumeDownParams `json:"params"`                     // 音量减参数
	Step   int                    `json:"step"`                       // 音量减量，默认5，范围0-20
}

// DeviceVolumeDownParams 设备音量减参数
// 包含音量减量等配置参数
type DeviceVolumeDownParams struct {
	Step int `json:"step"` // 音量减量，默认5，范围0-20
}

// DeviceVolumeDownResp 设备音量减指令响应
// 返回指令下发结果
type DeviceVolumeDownResp struct {
	InstructionID int64  `json:"instruction_id"` // 指令 ID
	Status        string `json:"status"`         // 指令状态：delivered-已下发, cached-已缓存
	Message       string `json:"message"`        // 提示信息
	TargetVolume  int    `json:"target_volume"`  // 目标音量值
	CurrentVolume int    `json:"current_volume"` // 当前音量值
}

// DeviceVolumeReq 设备音量调节指令请求（统一接口）
// 用户通过 App 向设备下发音量调节指令，支持直接设置目标音量值（0-100）
type DeviceVolumeReq struct {
	Sn           string `json:"sn" validate:"required"`            // 设备序列号，16位字母数字组合
	Action       string `json:"action" validate:"required"`        // 操作类型：set_volume/volume_up/volume_down
	TargetVolume int    `json:"target_volume" validate:"required"` // 目标音量值，范围0-100
}

// DeviceVolumeResp 设备音量调节指令响应
// 返回指令下发结果和当前/目标音量值
type DeviceVolumeResp struct {
	InstructionID int64  `json:"instruction_id"` // 指令 ID
	Status        string `json:"status"`         // 指令状态：delivered-已下发, cached-已缓存, failed-失败
	Message       string `json:"message"`        // 提示信息
	TargetVolume  int    `json:"target_volume"`  // 目标音量值（0-100）
	CurrentVolume int    `json:"current_volume"` // 调节前的当前音量值（0-100）
}

// DevicePlayPlaylistReq 设备播放歌单指令请求
// 用户通过 App 向设备下发播放歌单指令
type DevicePlayPlaylistReq struct {
	Sn     string                   `json:"sn" validate:"required"`     // 设备序列号，16位字母数字组合
	Action string                   `json:"action" validate:"required"` // 操作类型：play_playlist
	Params DevicePlayPlaylistParams `json:"params" validate:"required"` // 播放歌单参数
}

// DevicePlayPlaylistParams 设备播放歌单参数
// 包含歌单ID、起始索引、音量等配置参数
type DevicePlayPlaylistParams struct {
	PlaylistID string `json:"playlist_id" validate:"required"` // 歌单ID，必填参数
	StartIndex int    `json:"start_index"`                     // 起始播放索引，可选，默认0
	Volume     int    `json:"volume"`                          // 播放音量，可选，范围0-100
}

// DevicePlayPlaylistResp 设备播放歌单指令响应
// 返回指令下发结果
type DevicePlayPlaylistResp struct {
	InstructionID int64               `json:"instruction_id"` // 指令 ID
	Status        string              `json:"status"`         // 指令状态：delivered-已下发, cached-已缓存
	Message       string              `json:"message"`        // 提示信息
	Playlist      *DevicePlaylistInfo `json:"playlist"`       // 歌单信息
}

// DevicePlaylistInfo 设备歌单信息
// 包含歌单基本信息和第一首歌曲信息
type DevicePlaylistInfo struct {
	ID         string    `json:"id"`          // 歌单ID
	Name       string    `json:"name"`        // 歌单名称
	TotalCount int       `json:"total_count"` // 歌曲总数
	FirstSong  *SongInfo `json:"first_song"`  // 第一首歌曲信息
}

// SongInfo 歌曲信息
// 包含歌曲基本信息
type SongInfo struct {
	SongName string `json:"song_name"` // 歌曲名称
	Artist   string `json:"artist"`    // 歌手名称
	Duration int    `json:"duration"`  // 歌曲时长（秒）
}

// DeviceSetShuffleReq 设备设置随机播放指令请求
// 用户通过 App 向设备下发设置随机播放指令
type DeviceSetShuffleReq struct {
	Sn     string                 `json:"sn" validate:"required"`     // 设备序列号，16位字母数字组合
	Action string                 `json:"action" validate:"required"` // 操作类型：set_shuffle
	Params DeviceSetShuffleParams `json:"params" validate:"required"` // 随机播放参数
}

// DeviceSetShuffleParams 设备设置随机播放参数
// 包含随机播放开关等配置参数
type DeviceSetShuffleParams struct {
	Enable bool `json:"enable" validate:"required"` // 随机播放开关：true-开启随机播放、false-关闭随机播放
}

// DeviceSetShuffleResp 设备设置随机播放指令响应
// 返回指令下发结果
type DeviceSetShuffleResp struct {
	InstructionID int64  `json:"instruction_id"` // 指令 ID
	Status        string `json:"status"`         // 指令状态：delivered-已下发, cached-已缓存
	Message       string `json:"message"`        // 提示信息
	Shuffle       bool   `json:"shuffle"`        // 设置后的随机播放状态：true-已开启、false-已关闭
}

// DevicePlaybackProgressReq 设备播放进度查询请求
// 用户通过 App 查询设备当前播放进度
type DevicePlaybackProgressReq struct {
	Sn string `json:"sn" validate:"required"` // 设备序列号，16位字母数字组合
}

// DevicePlaybackProgressResp 设备播放进度查询响应
// 返回设备当前播放进度信息
type DevicePlaybackProgressResp struct {
	Sn            string  `json:"sn"`             // 设备序列号
	Online        bool    `json:"online"`         // 设备在线状态：true-在线、false-离线
	CurrentTime   int     `json:"current_time"`   // 当前播放位置（秒）
	Duration      int     `json:"duration"`       // 总时长（秒）
	Percentage    float64 `json:"percentage"`     // 播放进度百分比（保留一位小数）
	RemainingTime int     `json:"remaining_time"` // 剩余时间（秒）
	Timestamp     string  `json:"timestamp"`      // 进度记录时间
	Note          string  `json:"note,omitempty"` // 备注信息（设备离线时显示）
}

// DevicePlaybackStatusReq 设备播放状态查询请求
// 用户通过 App 查询设备当前播放状态
type DevicePlaybackStatusReq struct {
	Sn string `json:"sn" validate:"required"` // 设备序列号，16位字母数字组合
}

// DevicePlaybackStatusResp 设备播放状态查询响应
// 返回设备当前播放状态信息
type DevicePlaybackStatusResp struct {
	Sn         string                `json:"sn"`          // 设备序列号
	Online     bool                  `json:"online"`      // 设备在线状态：true-在线、false-离线
	Playback   *DevicePlaybackStatus `json:"playback"`    // 播放状态信息
	LastUpdate string                `json:"last_update"` // 最后更新时间
}

// DevicePlaybackStatus 设备播放状态详细信息
// 包含播放状态、当前歌曲、播放进度、音量、播放模式等
type DevicePlaybackStatus struct {
	State       string               `json:"state"`        // 播放状态：playing-播放中、paused-已暂停、stopped-已停止
	CurrentSong *PlaybackCurrentSong `json:"current_song"` // 当前播放歌曲信息
	Progress    *PlaybackProgress    `json:"progress"`     // 播放进度信息
	Volume      int                  `json:"volume"`       // 当前音量值，范围0-100
	Mode        *PlaybackMode        `json:"mode"`         // 播放模式信息
	QueueLength int                  `json:"queue_length"` // 播放队列歌曲数量
}

// PlaybackCurrentSong 播放当前歌曲信息
// 包含当前播放歌曲的基本信息
type PlaybackCurrentSong struct {
	MediaURL  string `json:"media_url"`  // 音频地址
	MediaName string `json:"media_name"` // 歌曲名称
	Artist    string `json:"artist"`     // 艺术家
	Album     string `json:"album"`      // 专辑名称
	Duration  int    `json:"duration"`   // 总时长（秒）
}

// PlaybackProgress 播放进度信息
// 包含当前播放进度和百分比
type PlaybackProgress struct {
	CurrentTime int     `json:"current_time"` // 当前播放位置（秒）
	Duration    int     `json:"duration"`     // 总时长（秒）
	Percentage  float64 `json:"percentage"`   // 播放进度百分比
}

// PlaybackMode 播放模式信息
// 包含循环模式和随机播放设置
type PlaybackMode struct {
	Loop    string `json:"loop"`    // 循环模式：off-关闭循环、one-单曲循环、all-列表循环
	Shuffle bool   `json:"shuffle"` // 随机播放：true-开启随机播放、false-关闭随机播放
}

// DeviceSetLoopReq 设备设置循环播放指令请求
// 用户通过 App 向设备下发设置循环播放指令
type DeviceSetLoopReq struct {
	Sn     string              `json:"sn" validate:"required"`     // 设备序列号，16位字母数字组合
	Action string              `json:"action" validate:"required"` // 操作类型：set_loop
	Params DeviceSetLoopParams `json:"params" validate:"required"` // 循环播放参数
}

// DeviceSetLoopParams 设备设置循环播放参数
// 包含循环模式等配置参数
type DeviceSetLoopParams struct {
	Mode string `json:"mode" validate:"required,oneof=off one all"` // 循环模式：off-关闭循环、one-单曲循环、all-列表循环
}

// DeviceSetLoopResp 设备设置循环播放指令响应
// 返回指令下发结果
type DeviceSetLoopResp struct {
	InstructionID int64  `json:"instruction_id"` // 指令 ID
	Status        string `json:"status"`         // 指令状态：delivered-已下发, cached-已缓存
	Message       string `json:"message"`        // 提示信息
	Loop          string `json:"loop"`           // 设置后的循环模式
}

// DeviceLocationResp 设备位置查询响应
// 返回设备位置信息
type DeviceLocationResp struct {
	Sn             string      `json:"sn"`               // 设备序列号
	Online         string      `json:"online"`           // 在线状态：online/offline
	UWB            UWBPosition `json:"uwb"`              // UWB定位数据
	Accuracy       float64     `json:"accuracy"`         // 定位精度（米）
	LastReportTime string      `json:"last_report_time"` // 最后上报时间
	IsLatest       bool        `json:"is_latest"`        // 是否为最新位置
}

// DeviceShadowReportReq 设备影子上报请求
// 设备定时上报状态数据
type DeviceShadowReportReq struct {
	Sn              string        `json:"sn"`               // 设备序列号
	Online          bool          `json:"online"`           // 在线状态
	FirmwareVersion string        `json:"firmware_version"` // 固件版本
	Battery         int           `json:"battery"`          // 电量百分比
	Volume          int           `json:"volume"`           // 音量值
	WorkMode        string        `json:"work_mode"`        // 工作模式
	Location        string        `json:"location"`         // 位置信息
	StorageUsed     int64         `json:"storage_used"`     // 已用存储空间（字节）
	StorageTotal    int64         `json:"storage_total"`    // 总存储空间（字节）
	SpeakerCount    int           `json:"speaker_count"`    // 扬声器连接数
	UWB             UWBPosition   `json:"uwb"`              // UWB定位数据
	Acoustic        AcousticCalib `json:"acoustic"`         // 声学校准参数
	RunState        string        `json:"run_state"`        // 运行状态
	Timestamp       int64         `json:"timestamp"`        // 上报时间戳（秒）
}

// DeviceShadowQueryResp 设备影子查询响应
// 返回设备影子信息
type DeviceShadowQueryResp struct {
	Sn              string        `json:"sn"`               // 设备序列号
	Online          string        `json:"online"`           // 在线状态：online/offline
	FirmwareVersion string        `json:"firmware_version"` // 固件版本
	Battery         int           `json:"battery"`          // 电量百分比
	Volume          int           `json:"volume"`           // 音量值
	WorkMode        string        `json:"work_mode"`        // 工作模式
	Position        string        `json:"position"`         // 位置信息
	StorageUsed     int64         `json:"storage_used"`     // 已用存储空间（字节）
	StorageTotal    int64         `json:"storage_total"`    // 总存储空间（字节）
	SpeakerCount    int           `json:"speaker_count"`    // 扬声器连接数
	UWB             UWBPosition   `json:"uwb"`              // UWB定位结构化数据
	Acoustic        AcousticCalib `json:"acoustic"`         // 声学校准参数
	LastReportTime  string        `json:"last_report_time"` // 最后上报时间
	RunState        string        `json:"run_state"`        // 运行状态
}

// DeviceLogReportReq 设备日志上报请求
// 设备通过 HTTP POST 请求上报运行日志
type DeviceLogReportReq struct {
	Sn        string                 `json:"sn" validate:"required"`       // 设备序列号，16位字母数字组合
	LogType   string                 `json:"log_type" validate:"required"` // 日志类型：error/warning/info/debug
	Level     string                 `json:"level" validate:"required"`    // 日志级别：debug/info/warn/error/fatal
	Content   string                 `json:"content" validate:"required"`  // 日志内容
	Metadata  map[string]interface{} `json:"metadata"`                     // 系统状态元数据（内存、CPU、WiFi信号等）
	Timestamp int64                  `json:"timestamp"`                    // 日志时间戳（秒）
}

// DeviceLogReportResp 设备日志上报响应
// 返回日志接收确认
type DeviceLogReportResp struct {
	LogID     string `json:"log_id"`    // 日志唯一 ID
	Success   bool   `json:"success"`   // 是否接收成功
	Message   string `json:"message"`   // 提示信息
	Timestamp string `json:"timestamp"` // 服务器接收时间
}

// DeviceDiagnoseReq 设备远程诊断请求
// 用户通过 App 发起设备远程诊断
type DeviceDiagnoseReq struct {
	Sn         string `json:"sn" validate:"required"`        // 设备序列号，16位字母数字组合
	DiagType   string `json:"diag_type" validate:"required"` // 诊断类型：full/quick/network/audio
	TimeoutSec int    `json:"timeout_sec"`                   // 诊断超时时间（秒），默认300
}

// DeviceDiagnoseResp 设备远程诊断响应
// 返回诊断任务信息和诊断结果
type DeviceDiagnoseResp struct {
	DiagID      string           `json:"diag_id"`      // 诊断任务 ID
	Status      string           `json:"status"`       // 诊断状态：pending/running/completed/failed
	DiagType    string           `json:"diag_type"`    // 诊断类型
	Result      *DiagnosisResult `json:"result"`       // 诊断结果（完成时返回）
	HealthScore int              `json:"health_score"` // 健康评分（0-100）
	Summary     string           `json:"summary"`      // 诊断摘要
	CreatedAt   string           `json:"created_at"`   // 创建时间
}

// DiagnosisResult 诊断结果详细信息
// 包含各项系统状态的诊断结果
type DiagnosisResult struct {
	CPU           *DiagnosisItem `json:"cpu"`            // CPU 状态
	Memory        *DiagnosisItem `json:"memory"`         // 内存状态
	Network       *DiagnosisItem `json:"network"`        // 网络状态
	WiFi          *DiagnosisItem `json:"wifi"`           // WiFi 状态
	Audio         *DiagnosisItem `json:"audio"`          // 音频状态
	Storage       *DiagnosisItem `json:"storage"`        // 存储状态
	Battery       *DiagnosisItem `json:"battery"`        // 电池状态
	Firmware      *DiagnosisItem `json:"firmware"`       // 固件状态
	TotalItems    int            `json:"total_items"`    // 总检查项数
	NormalItems   int            `json:"normal_items"`   // 正常项数
	AbnormalItems int            `json:"abnormal_items"` // 异常项数
}

// DiagnosisItem 诊断检查项
// 单个系统组件的诊断结果
type DiagnosisItem struct {
	Status   string `json:"status"`   // 状态：normal/warning/error
	Name     string `json:"name"`     // 检查项名称
	Message  string `json:"message"`  // 诊断信息
	Value    string `json:"value"`    // 当前值
	Expected string `json:"expected"` // 期望值
}

// WillMessageReq WILL消息请求（设备异常断开通知）
// EMQX自动发布WILL消息到后端，用于检测设备异常断开
type WillMessageReq struct {
	SN     string `json:"sn" validate:"required"`     // 设备序列号
	Status string `json:"status" validate:"required"` // 离线状态："offline"
	Time   string `json:"time"`                       // 离线时间（ISO8601格式）
	Reason string `json:"reason,omitempty"`           // 可选：断开原因
}

// WillMessageResp WILL消息响应
type WillMessageResp struct {
	Code    int    `json:"code"`    // 状态码：200-成功
	Message string `json:"message"` // 消息内容
}

// ========== 歌曲下载相关类型 ==========

// SongDownloadReq 歌曲下载请求（前端调用）
// 用户通过前端页面点击下载按钮，向后端发起下载请求
type SongDownloadReq struct {
	Sn       string `json:"sn" validate:"required"`      // 目标设备序列号（16位）
	SongID   int64  `json:"song_id" validate:"required"` // 歌曲唯一编号
	SongName string `json:"song_name,omitempty"`         // 歌曲名称（可选，用于日志）
}

// SongDownloadResp 歌曲下载响应
// 后端接收请求后，校验权限并下发指令给设备
type SongDownloadResp struct {
	TaskID        string `json:"task_id"`                  // 下载任务ID（用于追踪）
	SongID        int64  `json:"song_id"`                  // 歌曲ID
	DeviceSN      string `json:"device_sn"`                // 设备SN
	Status        string `json:"status"`                   // 状态：pending/delivered/failed
	Message       string `json:"message"`                  // 提示信息
	InstructionID *int64 `json:"instruction_id,omitempty"` // 指令ID
}

// WSSongDownloadInstruction WebSocket下行-歌曲下载指令
// 后端通过WebSocket下发给设备的下载指令
type WSSongDownloadInstruction struct {
	Cmd         string `json:"cmd"`          // 固定值："download_song"
	TaskID      string `json:"task_id"`      // 任务ID（UUID）
	SongID      int64  `json:"song_id"`      // 歌曲ID
	SongName    string `json:"song_name"`    // 歌曲名称
	DownloadURL string `json:"download_url"` // CDN下载地址
	Token       string `json:"token"`        // 设备鉴权Token（可选）
	Timestamp   int64  `json:"timestamp"`    // 服务端时间戳（Unix秒）
}

// WSDownloadProgressReport WebSocket上行-下载进度上报（可选）
// 设备在下载过程中可实时上报进度
type WSDownloadProgressReport struct {
	Event          string  `json:"event"`                     // 固定值："download_progress"
	TaskID         string  `json:"task_id"`                   // 任务ID
	SongID         int64   `json:"song_id"`                   // 歌曲ID
	Percent        float64 `json:"percent"`                   // 进度百分比（0-100）
	DownloadedSize int64   `json:"downloaded_size,omitempty"` // 已下载大小（字节）
	TotalSize      int64   `json:"total_size,omitempty"`      // 总大小（字节）
	Timestamp      int64   `json:"timestamp"`                 // 上报时间戳
}

// WSDownloadResultReport WebSocket上行-下载结果上报
// 设备下载完成或失败后上报结果
type WSDownloadResultReport struct {
	Event      string `json:"event"`                 // 固定值："download_finish"
	TaskID     string `json:"task_id"`               // 任务ID
	Sn         string `json:"sn,omitempty"`          // 设备 SN（HTTP 回调鉴权用）
	DeviceID   int64  `json:"device_id,omitempty"`   // 设备 ID（可选，与凭证校验）
	SongID     int64  `json:"song_id"`               // 歌曲ID
	Status     string `json:"status"`                // 状态："success" 或 "fail"
	ErrorMsg   string `json:"error_msg,omitempty"`   // 失败原因（status=fail时必填）
	LocalPath  string `json:"local_path,omitempty"`  // 本地存储路径（成功时返回）
	FileSize   int64  `json:"file_size,omitempty"`   // 文件大小（字节）
	DurationMs int64  `json:"duration_ms,omitempty"` // 下载耗时（毫秒）
	Timestamp  int64  `json:"timestamp"`             // 上报时间戳
}

// DownloadStatusQueryReq 查询下载状态请求
// 前端轮询查询某个下载任务的状态
type DownloadStatusQueryReq struct {
	TaskID string `json:"task_id" validate:"required"` // 任务ID
}

// DownloadStatusQueryResp 查询下载状态响应
type DownloadStatusQueryResp struct {
	TaskID    string  `json:"task_id"`    // 任务ID
	SongID    int64   `json:"song_id"`    // 歌曲ID
	SongName  string  `json:"song_name"`  // 歌曲名称
	DeviceSN  string  `json:"device_sn"`  // 设备SN
	Status    string  `json:"status"`     // pending/downloading/success/failed
	Progress  float64 `json:"progress"`   // 进度百分比（0-100）
	Message   string  `json:"message"`    // 状态描述
	CreatedAt string  `json:"created_at"` // 创建时间
	UpdatedAt string  `json:"updated_at"` // 最后更新时间
}

// DeviceUpdateReq 设备信息更新请求（PUT /api/v1/user/device/upd）
type DeviceUpdateReq struct {
	Sn         string `json:"sn" validate:"required"`
	DeviceName string `json:"device_name,omitempty"`
	Location   string `json:"location,omitempty"`
	GroupName  string `json:"group_name,omitempty"`
	Scene      string `json:"scene,omitempty"`
}

// DeviceUpdateResp 设备信息更新响应
type DeviceUpdateResp struct {
	Sn         string `json:"sn"`
	DeviceName string `json:"device_name"`
	Location   string `json:"location"`
	GroupName  string `json:"group_name"`
	Scene      string `json:"scene"`
	UpdatedAt  string `json:"updated_at"`
}

// ========== 设备下载完整流程相关类型 ==========

// DeviceDownloadSongReq 下载歌曲到设备（HTTP/WS）
type DeviceDownloadSongReq struct {
	DeviceID  int64  `json:"device_id,omitempty"`
	Sn        string `json:"sn" validate:"required"`
	ContentID int64  `json:"content_id" validate:"required"`
	SongName  string `json:"song_name,omitempty"`
	Quality   string `json:"quality,omitempty"`
	SourceURL string `json:"source_url,omitempty"`
	EnableDRM *bool  `json:"enable_drm,omitempty"`
}

// DeviceDownloadSongResp 下载歌曲指令响应
type DeviceDownloadSongResp struct {
	TaskID        string `json:"task_id"`
	InstructionID *int64 `json:"instruction_id,omitempty"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

// DRMMetadata DRM 元数据（来自 media-processing 响应头）
type DRMMetadata struct {
	DRMType     string `json:"drm_type,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
	KeyID       string `json:"key_id,omitempty"`
	PSSH        string `json:"pssh,omitempty"`
	LicenseURL  string `json:"license_url,omitempty"`
	AuthToken   string `json:"auth_token,omitempty"`
	Copyright   string `json:"copyright,omitempty"`
	UsagePolicy string `json:"usage_policy,omitempty"`
	Signature   string `json:"signature,omitempty"`
}

// DRMDownloadResponse 设备侧 DRM 下载处理结果
type DRMDownloadResponse struct {
	TaskID       string `json:"task_id"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	RequestTime  string `json:"request_time,omitempty"`
	LocalPath    string `json:"local_path,omitempty"`
	DRMFilePath  string `json:"drm_file_path,omitempty"`
	FileSize     int64  `json:"file_size,omitempty"`
	DRMSignature string `json:"drm_signature,omitempty"`
	CompletedAt  string `json:"completed_at,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// DRMLogRecord DRM 操作审计日志（内存/落库追溯用）
type DRMLogRecord struct {
	TaskID       string    `json:"task_id"`
	SongID       int64     `json:"song_id"`
	Action       string    `json:"action"`
	DRMType      string    `json:"drm_type"`
	ContentID    string    `json:"content_id"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// DeviceDownloadReq 设备下载请求（前端调用设备服务8002）
// 用户通过前端点击下载按钮，将歌曲下载到指定设备
type DeviceDownloadReq struct {
	Sn        string `json:"sn" validate:"required"`         // 目标设备序列号
	ContentID int64  `json:"content_id" validate:"required"` // 内容ID（对应content表）
}

// DeviceDownloadResp 设备下载响应
type DeviceDownloadResp struct {
	TaskID        string `json:"task_id"`                  // 任务ID（用于追踪）
	ContentID     int64  `json:"content_id"`               // 内容ID
	DeviceSN      string `json:"device_sn"`                // 设备SN
	Status        string `json:"status"`                   // 状态：pending/sent/failed
	Message       string `json:"message"`                  // 提示信息
	InstructionID *int64 `json:"instruction_id,omitempty"` // 指令ID
}

// DeviceDownloadCallbackReq 设备下载回调请求（设备→设备服务）
// 设备下载完成后上报结果给后端
type DeviceDownloadCallbackReq struct {
	TaskID    string `json:"task_id" validate:"required"`    // 任务ID（下发给设备的）
	DeviceSN  string `json:"device_sn" validate:"required"`  // 设备编号
	ContentID int64  `json:"content_id" validate:"required"` // 歌曲ID/内容ID
	Status    string `json:"status" validate:"required"`     // 状态：success / fail
	ErrorMsg  string `json:"error_msg,omitempty"`            // 失败原因（status=fail 时必填）
}

// DeviceDownloadCallbackResp 设备下载回调响应
type DeviceDownloadCallbackResp struct {
	Success bool   `json:"success"` // 是否成功
	Message string `json:"message"` // 消息
}

// DeviceDownloadStatusQueryReq 查询设备下载状态请求
type DeviceDownloadStatusQueryReq struct {
	TaskID string `form:"task_id" validate:"required"` // 任务ID
}

// DeviceDownloadStatusQueryResp 查询设备下载状态响应
type DeviceDownloadStatusQueryResp struct {
	TaskID     string  `json:"task_id"`               // 任务ID
	ContentID  int64   `json:"content_id"`            // 内容ID
	SongName   string  `json:"song_name"`             // 歌曲名称
	DeviceSN   string  `json:"device_sn"`             // 设备SN
	Status     string  `json:"status"`                // pending/downloading/success/failed/cancelled
	Progress   float64 `json:"progress"`              // 进度百分比（0-100）
	Message    string  `json:"message"`               // 状态描述
	CreatedAt  string  `json:"created_at"`            // 创建时间
	UpdatedAt  string  `json:"updated_at"`            // 最后更新时间
	FinishedAt string  `json:"finished_at,omitempty"` // 完成时间
}

// DeviceDownloadListReq 设备下载历史列表请求
type DeviceDownloadListReq struct {
	Page     int32  `form:"page"`             // 页码（默认1）
	PageSize int32  `form:"page_size"`        // 每页数量（默认20，最大50）
	DeviceSN string `form:"sn,omitempty"`     // 设备SN过滤（可选）
	Status   string `form:"status,omitempty"` // 状态过滤（可选）
}

// DeviceDownloadListItem 设备下载列表项
type DeviceDownloadListItem struct {
	TaskID     string  `json:"task_id"`               // 任务ID
	ContentID  int64   `json:"content_id"`            // 内容ID
	SongName   string  `json:"song_name"`             // 歌曲名称
	DeviceSN   string  `json:"device_sn"`             // 设备SN
	Status     string  `json:"status"`                // 状态
	Progress   float64 `json:"progress"`              // 进度百分比
	FileSize   int64   `json:"file_size"`             // 文件大小
	ErrorMsg   string  `json:"error_msg"`             // 错误信息
	CreatedAt  string  `json:"created_at"`            // 创建时间
	UpdatedAt  string  `json:"updated_at"`            // 更新时间
	FinishedAt string  `json:"finished_at,omitempty"` // 完成时间
}

// DeviceDownloadListResp 设备下载列表响应
type DeviceDownloadListResp struct {
	Total      int64                    `json:"total"`       // 总记录数
	List       []DeviceDownloadListItem `json:"list"`        // 当前页数据
	Page       int32                    `json:"page"`        // 当前页码
	PageSize   int32                    `json:"page_size"`   // 每页数量
	TotalPages int32                    `json:"total_pages"` // 总页数
}
