package logic

// AudioTrack 分离后的单个音轨
type AudioTrack struct {
	ID         int64   `json:"id"`          // 音轨 ID
	TaskID     string  `json:"task_id"`     // 所属任务 ID
	TrackName  string  `json:"track_name"`  // 音轨名称（vocals/drums/bass/other）
	TrackURL   string  `json:"track_url"`   // 音轨文件 URL
	FileSize   int64   `json:"file_size"`   // 文件大小（字节）
	Duration   float64 `json:"duration"`    // 时长（秒）
	SampleRate int     `json:"sample_rate"` // 采样率
	Channels   int     `json:"channels"`    // 声道数
	Format     string  `json:"format"`      // 文件格式（wav）
	OrderIndex int     `json:"order_index"` // 排序索引
	CreatedAt  string  `json:"created_at"`  // 创建时间
}

// TrackSeparationResult 音轨分离结果（一个任务对应多个音轨）
type TrackSeparationResult struct {
	TaskID      string       `json:"task_id"`      // 任务 ID
	ContentID   int64        `json:"content_id"`   // 内容 ID
	UserID      int64        `json:"user_id"`      // 用户 ID
	AudioURL    string       `json:"audio_url"`    // 原始音频 URL
	Tracks      []AudioTrack `json:"tracks"`       // 分离后的音轨列表
	Status      string       `json:"status"`       // 分离状态（pending/processing/completed/failed）
	Message     string       `json:"message"`      // 状态消息
	CreatedAt   string       `json:"created_at"`   // 创建时间
	CompletedAt string       `json:"completed_at"` // 完成时间
}
