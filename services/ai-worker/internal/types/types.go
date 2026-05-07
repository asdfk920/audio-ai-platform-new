package types

// InferenceStartReq 音轨分离推理任务创建请求
type InferenceStartReq struct {
	AudioURL     string   `json:"audio_url" form:"audio_url"`                   // 音频文件地址
	ModelType    string   `json:"model_type,omitempty" form:"model_type"`       // 模型类型，默认 bsroformer
	OutputTracks []string `json:"output_tracks,omitempty" form:"output_tracks"` // 输出音轨，默认全部
	CallbackURL  string   `json:"callback_url,omitempty" form:"callback_url"`   // 完成后回调地址
}

// InferenceStartResp 音轨分离推理任务创建响应
type InferenceStartResp struct {
	TaskID    string `json:"task_id"`    // 任务 ID
	Status    string `json:"status"`     // 任务状态
	Message   string `json:"message"`    // 返回消息
	CreatedAt string `json:"created_at"` // 创建时间
}

// InferenceTaskInfo 推理任务信息
type InferenceTaskInfo struct {
	TaskID       string            `json:"task_id"`                 // 任务 ID
	UserID       int64             `json:"user_id"`                 // 用户 ID
	AudioURL     string            `json:"audio_url"`               // 音频文件地址
	AudioPath    string            `json:"audio_path"`              // 本地存储路径
	ModelType    string            `json:"model_type"`              // 使用的模型
	OutputTracks []string          `json:"output_tracks"`           // 输出配置
	Status       string            `json:"status"`                  // 任务状态：pending, processing, completed, failed
	Progress     int               `json:"progress"`                // 进度 0-100
	ErrorMessage string            `json:"error_message,omitempty"` // 错误信息
	ResultURLs   map[string]string `json:"result_urls,omitempty"`   // 分离结果 URL
	CreatedAt    string            `json:"created_at"`              // 创建时间
	StartedAt    string            `json:"started_at,omitempty"`    // 开始处理时间
	CompletedAt  string            `json:"completed_at,omitempty"`  // 完成时间
}

// InferenceTaskQueryReq 推理任务查询请求
type InferenceTaskQueryReq struct {
	TaskID string `json:"task_id" form:"task_id"` // 任务 ID
}

// InferenceTaskQueryResp 推理任务查询响应
type InferenceTaskQueryResp struct {
	TaskID      string            `json:"task_id"`
	Status      string            `json:"status"`
	Progress    int               `json:"progress"`
	Message     string            `json:"message"`
	ResultURLs  map[string]string `json:"result_urls,omitempty"`
	CreatedAt   string            `json:"created_at"`
	CompletedAt string            `json:"completed_at,omitempty"`
}

// InferenceCallbackReq 推理完成回调请求
type InferenceCallbackReq struct {
	TaskID     string            `json:"task_id"`
	Status     string            `json:"status"`
	ResultURLs map[string]string `json:"result_urls"`
}

// InferenceTaskListReq 推理任务列表查询请求
type InferenceTaskListReq struct {
	UserID   int64  `json:"user_id" form:"user_id"`     // 用户 ID（可选）
	Status   string `json:"status" form:"status"`       // 状态筛选：pending, processing, completed, failed（可选）
	Page     int    `json:"page" form:"page"`           // 页码，默认 1
	PageSize int    `json:"page_size" form:"page_size"` // 每页数量，默认 10，最大 100
}

// InferenceTaskListResp 推理任务列表查询响应
type InferenceTaskListResp struct {
	Total    int64                   `json:"total"`     // 总数
	Page     int                     `json:"page"`      // 当前页码
	PageSize int                     `json:"page_size"` // 每页数量
	Tasks    []InferenceTaskListItem `json:"tasks"`     // 任务列表
}

// InferenceTaskListItem 任务列表项
type InferenceTaskListItem struct {
	TaskID       string            `json:"task_id"`
	AudioURL     string            `json:"audio_url"`
	ModelType    string            `json:"model_type"`
	Status       string            `json:"status"`
	Progress     int               `json:"progress"`
	ResultURLs   map[string]string `json:"result_urls,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	CreatedAt    string            `json:"created_at"`
	CompletedAt  string            `json:"completed_at,omitempty"`
}

// InferenceResultReq 推理结果获取请求
type InferenceResultReq struct {
	TaskID string `json:"task_id" form:"task_id"` // 任务 ID
}

// TrackInfo 音轨信息
type TrackInfo struct {
	Name       string `json:"name"`        // 音轨名称：vocals, drums, bass, other
	Label      string `json:"label"`       // 显示标签：人声, 鼓, 贝斯, 其他
	URL        string `json:"url"`         // 音频文件下载地址
	Duration   int64  `json:"duration"`    // 时长（秒）
	FileSize   int64  `json:"size"`        // 文件大小（字节）
	Format     string `json:"format"`      // 格式：wav, mp3
	SampleRate int    `json:"sample_rate"` // 采样率
}

// InferenceResultResp 推理结果响应
type InferenceResultResp struct {
	TaskID      string      `json:"task_id"`
	Status      string      `json:"status"`
	AudioURL    string      `json:"original_audio_url"`
	ModelType   string      `json:"model_type"`
	Tracks      []TrackInfo `json:"tracks"`
	CreatedAt   string      `json:"created_at"`
	CompletedAt string      `json:"completed_at"`
}

// InferenceCancelReq 取消推理任务请求
type InferenceCancelReq struct {
	TaskID string `json:"task_id" form:"task_id"`         // 任务 ID
	Reason string `json:"reason,omitempty" form:"reason"` // 取消原因（可选）
}

// InferenceCancelResp 取消推理任务响应
type InferenceCancelResp struct {
	TaskID         string `json:"task_id"`         // 任务 ID
	PreviousStatus string `json:"previous_status"` // 取消前的状态
	CurrentStatus  string `json:"current_status"`  // 当前状态：cancelled
	CancelledAt    string `json:"cancelled_at"`    // 取消时间
	Message        string `json:"message"`         // 返回消息
}

// WSMessageType WebSocket 消息类型
const (
	WSMsgTypeAudioFrame = "audio_frame" // 客户端发送音频帧
	WSMsgTypeResult     = "result"      // 服务器返回分离结果
	WSMsgTypeComplete   = "complete"    // 处理完成，返回完整结果
	WSMsgTypeAudioEnd   = "audio_end"   // 客户端发送完成信号
	WSMsgTypeError      = "error"       // 错误消息
	WSMsgTypeConnected  = "connected"   // 连接成功
)

// WSAudioFrame 客户端发送的音频帧
type WSAudioFrame struct {
	Type       string `json:"type"`        // audio_frame
	Data       string `json:"data"`        // base64 编码的音频数据
	SampleRate int    `json:"sample_rate"` // 采样率（44100）
	Channels   int    `json:"channels"`    // 声道数（2=立体声）
	FrameIndex int    `json:"frame_index"` // 帧序号
}

// WSFrameResult 服务器返回的单帧处理结果
type WSFrameResult struct {
	Type       string `json:"type"`        // result
	Track      string `json:"track"`       // vocals/drums/bass/other
	Data       string `json:"data"`        // base64 编码的分离后音频
	FrameIndex int    `json:"frame_index"` // 帧序号
	Progress   int    `json:"progress"`    // 整体进度百分比
}

// WSCompleteResult 处理完成的完整结果
type WSCompleteResult struct {
	Type     string            `json:"type"` // complete
	TaskID   string            `json:"task_id"`
	Tracks   map[string]string `json:"tracks"`   // 各音轨文件 URL
	Duration float64           `json:"duration"` // 总时长（秒）
}

// WSError 错误消息
type WSError struct {
	Type    string `json:"type"` // error
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// WSConnected 连接成功消息
type WSConnected struct {
	Type    string `json:"type"` // connected
	TaskID  string `json:"task_id"`
	Message string `json:"message"`
}

// BatchInferenceStartReq 批量推理请求
type BatchInferenceStartReq struct {
	Songs          []BatchSongItem `json:"songs"`                     // 歌曲列表
	ModelType      string          `json:"model_type,omitempty"`      // 模型类型，默认 bsroformer
	MaxConcurrency int             `json:"max_concurrency,omitempty"` // 最大并发数，默认3
}

// BatchSongItem 批量歌曲项
type BatchSongItem struct {
	SongID   string `json:"song_id"`             // 歌曲 ID
	SongName string `json:"song_name,omitempty"` // 歌曲名称（可选）
	AudioURL string `json:"audio_url,omitempty"` // 音频 URL（可选）
}

// BatchInferenceStartResp 批量推理响应
type BatchInferenceStartResp struct {
	BatchID    string `json:"batch_id"`    // 批量任务 ID
	Status     string `json:"status"`      // 状态：processing
	TotalCount int    `json:"total_count"` // 总任务数
	CreatedAt  string `json:"created_at"`  // 创建时间
	Message    string `json:"message"`     // 返回消息
}

// BatchTaskStatus 单个任务状态
type BatchTaskStatus struct {
	SongID    string `json:"song_id"`              // 歌曲 ID
	TaskID    string `json:"task_id"`              // 推理任务 ID
	Status    string `json:"status"`               // 状态：pending/processing/completed/failed
	Progress  int    `json:"progress"`             // 进度 0-100
	ResultURL string `json:"result_url,omitempty"` // 结果 URL（完成后有值）
	ErrorMsg  string `json:"error_msg,omitempty"`  // 错误信息（失败时有值）
}

// BatchQueryResp 批量任务查询响应
type BatchQueryResp struct {
	BatchID         string            `json:"batch_id"`               // 批量任务 ID
	Status          string            `json:"status"`                 // 状态：processing/completed/failed/partially_completed
	TotalCount      int               `json:"total_count"`            // 总任务数
	CompletedCount  int               `json:"completed_count"`        // 已完成数量
	FailedCount     int               `json:"failed_count"`           // 失败数量
	PendingCount    int               `json:"pending_count"`          // 待处理数量
	ProcessingCount int               `json:"processing_count"`       // 处理中数量
	Progress        float64           `json:"progress"`               // 整体进度 0-100
	Tasks           []BatchTaskStatus `json:"tasks"`                  // 任务列表
	CreatedAt       string            `json:"created_at"`             // 创建时间
	CompletedAt     string            `json:"completed_at,omitempty"` // 完成时间
}

// BatchResultResp 批量任务结果响应
type BatchResultResp struct {
	BatchID        string            `json:"batch_id"`               // 批量任务 ID
	Status         string            `json:"status"`                 // 最终状态
	TotalCount     int               `json:"total_count"`            // 总任务数
	CompletedCount int               `json:"completed_count"`        // 成功数量
	FailedCount    int               `json:"failed_count"`           // 失败数量
	FailedSongs    []string          `json:"failed_songs,omitempty"` // 失败的歌曲 ID 列表
	Results        []BatchTaskResult `json:"results"`                // 成功的结果列表
	Duration       float64           `json:"duration"`               // 总耗时（秒）
	CompletedAt    string            `json:"completed_at"`           // 完成时间
}

// BatchTaskResult 单个任务的完整结果
type BatchTaskResult struct {
	SongID   string            `json:"song_id"`  // 歌曲 ID
	TaskID   string            `json:"task_id"`  // 任务 ID
	Tracks   map[string]string `json:"tracks"`   // 分离后的音轨 URL
	Duration float64           `json:"duration"` // 音频时长
}
