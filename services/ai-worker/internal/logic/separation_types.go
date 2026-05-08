package logic

// SeparationJob 分离任务
type SeparationJob struct {
	ID        string
	InputPath string
	Tracks    []string // 需要分离的音轨列表
}

// SeparationResult 分离结果
type SeparationResult struct {
	TaskID   string            `json:"task_id"`
	Tracks   map[string][]byte `json:"tracks"` // track_name -> audio_data
	Duration float64           `json:"duration"`
}
