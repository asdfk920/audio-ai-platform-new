package types

// EnumItem 枚举项
// 用于下拉选项、列表等场景
type EnumItem struct {
	Label string `json:"label"` // 显示文本
	Value string `json:"value"` // 实际值
}

// DeviceRegisterReq 设备注册请求
// 设备首次上线时向云端注册身份
type DeviceRegisterReq struct {
	Sn              string `json:"sn" validate:"required"`               // 设备序列号，16位字母数字组合
	Model           string `json:"model" validate:"required"`            // 设备型号
	FirmwareVersion string `json:"firmware_version" validate:"required"` // 固件版本号
}

// DeviceRegisterResp 设备注册响应
// 返回设备认证 token
type DeviceRegisterResp struct {
	Token string `json:"token"` // 认证 token，设备后续请求需携带此 token
}

// DeviceAuthReq 设备认证请求
// 设备使用 token 向云端认证身份
type DeviceAuthReq struct {
	Sn    string `json:"sn" validate:"required"`    // 设备序列号，16位字母数字组合
	Token string `json:"token" validate:"required"` // 注册时获取的认证 token
}

// DeviceAuthResp 设备认证响应
// 返回认证结果和设备信息
type DeviceAuthResp struct {
	Success         bool   `json:"success"`          // 是否认证成功
	DeviceID        int64  `json:"device_id"`        // 设备 ID
	Sn              string `json:"sn"`               // 设备序列号
	Model           string `json:"model"`            // 设备型号
	FirmwareVersion string `json:"firmware_version"` // 固件版本号
	Message         string `json:"message"`          // 提示信息
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
