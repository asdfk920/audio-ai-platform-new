package inference

import (
	"context"
	"io"
	"time"
)

// InferenceServiceServer is the server API for InferenceService service.
type InferenceServiceServer interface {
	// 双向流式推理 - 实时音轨分离
	StreamSeparate(stream InferenceService_StreamSeparateServer) error

	// 单次推理任务
	Separate(context.Context, *SeparateRequest) (*SeparateResponse, error)

	// 提交批量任务
	SubmitBatch(context.Context, *BatchRequest) (*BatchResponse, error)

	// 查询任务状态
	GetTaskStatus(context.Context, *TaskStatusRequest) (*TaskStatusResponse, error)

	// 获取任务结果
	GetTaskResult(context.Context, *TaskResultRequest) (*TaskResultResponse, error)

	// 取消任务
	CancelTask(context.Context, *CancelTaskRequest) (*CancelTaskResponse, error)

	// 获取模型信息
	GetModelInfo(context.Context, *ModelInfoRequest) (*ModelInfoResponse, error)

	// 健康检查
	HealthCheck(context.Context, *HealthCheckRequest) (*HealthCheckResponse, error)
}

// UnimplementedInferenceServiceServer must be embedded to have forward compatible implementations.
type UnimplementedInferenceServiceServer struct{}

func (UnimplementedInferenceServiceServer) StreamSeparate(stream InferenceService_StreamSeparateServer) error {
	return nil
}
func (UnimplementedInferenceServiceServer) Separate(ctx context.Context, req *SeparateRequest) (*SeparateResponse, error) {
	return nil, nil
}
func (UnimplementedInferenceServiceServer) SubmitBatch(ctx context.Context, req *BatchRequest) (*BatchResponse, error) {
	return nil, nil
}
func (UnimplementedInferenceServiceServer) GetTaskStatus(ctx context.Context, req *TaskStatusRequest) (*TaskStatusResponse, error) {
	return nil, nil
}
func (UnimplementedInferenceServiceServer) GetTaskResult(ctx context.Context, req *TaskResultRequest) (*TaskResultResponse, error) {
	return nil, nil
}
func (UnimplementedInferenceServiceServer) CancelTask(ctx context.Context, req *CancelTaskRequest) (*CancelTaskResponse, error) {
	return nil, nil
}
func (UnimplementedInferenceServiceServer) GetModelInfo(ctx context.Context, req *ModelInfoRequest) (*ModelInfoResponse, error) {
	return nil, nil
}
func (UnimplementedInferenceServiceServer) HealthCheck(ctx context.Context, req *HealthCheckRequest) (*HealthCheckResponse, error) {
	return nil, nil
}

// InferenceService_StreamSeparateServer is the server API for InferenceService.StreamSeparate RPC.
type InferenceService_StreamSeparateServer interface {
	Send(*SeparatedTrack) error
	Recv() (*AudioFrame, error)
	grpc.ServerStream
}

// ==================== 消息类型定义 ====================

type AudioFrame struct {
	TaskId     string    `protobuf:"bytes,1,opt,name=task_id,json=taskId,proto3" json:"task_id,omitempty"`
	FrameIndex int32     `protobuf:"varint,2,opt,name=frame_index,json=frameIndex,proto3" json:"frame_index,omitempty"`
	AudioData  []byte    `protobuf:"bytes,3,opt,name=audio_data,json=audioData,proto3" json:"audio_data,omitempty"`
	SampleRate int32     `protobuf:"varint,4,opt,name=sample_rate,json=sampleRate,proto3" json:"sample_rate,omitempty"`
	Channels    int32     `protobuf:"varint,5,opt,name=channels,proto3" json:"channels,omitempty"`
	Format      AudioFormat `protobuf:"varint,6,opt,name=format,proto3,enum=inference.AudioFormat" json:"format,omitempty"`
	Timestamp   time.Time `protobuf:"bytes,7,opt,name=timestamp,proto3,stdtime" json:"timestamp"`
}

func (m *AudioFrame) Reset()         { *m = AudioFrame{} }
func (m *AudioFrame) String() string { return "" }

type SeparatedTrack struct {
	TaskId       string          `protobuf:"bytes,1,opt,name=task_id,json=taskId,proto3" json:"task_id,omitempty"`
	TrackType    TrackType       `protobuf:"varint,2,opt,name=track_type,proto3,enum=inference.TrackType" json:"track_type,omitempty"`
	TrackData    []byte          `protobuf:"bytes,3,opt,name=track_data,json=trackData,proto3" json:"track_data,omitempty"`
	FrameIndex   int32           `protobuf:"varint,4,opt,name=frame_index,json=frameIndex,proto3" json:"frame_index,omitempty"`
	Progress     float32         `protobuf:"fixed32,5,opt,name=progress,proto3" json:"progress,omitempty"`
	Status       InferenceStatus `protobuf:"varint,6,opt,name=status,proto3,enum=inference.InferenceStatus" json:"status,omitempty"`
	ErrorMessage string          `protobuf:"bytes,7,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"`
}

func (m *SeparatedTrack) Reset()         { *m = SeparatedTrack{} }
func (m *SeparatedTrack) String() string { return "" }

type SeparateRequest struct {
	TaskId       string                 `protobuf:"bytes,1,opt,name=task_id,json=taskId,proto3" json:"task_id,omitempty"`
	AudioUrl     string                 `protobuf:"bytes,2,opt,name=audio_url,json=audioUrl,proto3" json:"audio_url,omitempty"`
	ModelType    ModelType              `protobuf:"varint,3,opt,name=model_type,proto3,enum=inference.ModelType" json:"model_type,omitempty"`
	OutputFormat OutputFormat           `protobuf:"varint,4,opt,name=output_format,proto3,enum=inference.OutputFormat" json:"output_format,omitempty"`
	Metadata      map[string]string     `protobuf:"bytes,5,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (m *SeparateRequest) Reset()         { *m = SeparateRequest{} }
func (m *SeparateRequest) String() string { return "" }

type SeparateResponse struct {
	TaskId       string                `protobuf:"bytes,1,opt,name=task_id,json=taskId,proto3" json:"task_id,omitempty"`
	Status       InferenceStatus       `protobuf:"varint,2,opt,name=status,proto3,enum=inference.InferenceStatus" json:"status,omitempty"`
	Tracks       map[string]*TrackResult `protobuf:"bytes,3,rep,name=tracks,proto3" json:"tracks,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Duration     float64               `protobuf:"fixed64,4,opt,name=duration,proto3" json:"duration,omitempty"`
	ProcessTime  float64               `protobuf:"fixed64,5,opt,name=process_time,json=processTime,proto3" json:"process_time,omitempty"`
	ModelInfo    *ModelInfo            `protobuf:"bytes,6,opt,name=model_info,json=modelInfo,proto3" json:"model_info,omitempty"`
	ErrorMessage string                `protobuf:"bytes,7,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"`
}

func (m *SeparateResponse) Reset()         { *m = SeparateResponse{} }
func (m *SeparateResponse) String() string { return "" }

type BatchRequest struct {
	BatchId        string             `protobuf:"bytes,1,opt,name=batch_id,json=batchId,proto3" json:"batch_id,omitempty"`
	Songs          []*SongItem        `protobuf:"bytes,2,rep,name=songs,proto3" json:"songs,omitempty"`
	ModelType      ModelType          `protobuf:"varint,3,opt,name=model_type,proto3,enum=inference.ModelType" json:"model_type,omitempty"`
	MaxConcurrency int32              `protobuf:"varint,4,opt,name=max_concurrency,json=maxConcurrency,proto3" json:"max_concurrency,omitempty"`
	Callback       *CallbackConfig    `protobuf:"bytes,5,opt,name=callback,proto3" json:"callback,omitempty"`
}

func (m *BatchRequest) Reset()         { *m = BatchRequest{} }
func (m *BatchRequest) String() string { return "" }

type BatchResponse struct {
	BatchId   string                 `protobuf:"bytes,1,opt,name=batch_id,json=batchId,proto3" json:"batch_id,omitempty"`
	Status    BatchStatus            `protobuf:"varint,2,opt,name=status,proto3,enum=inference.BatchStatus" json:"status,omitempty"`
	TotalCount int32                  `protobuf:"varint,3,opt,name=total_count,jsontotalCount,proto3" json:"total_count,omitempty"`
	CreatedAt time.Time              `protobuf:"bytes,4,opt,name=created_at,json=createdAt,proto3,stdtime" json:"created_at,omitempty"`
	Message   string                 `protobuf:"bytes,5,opt,name=message,proto3" json:"message,omitempty"`
}

func (m *BatchResponse) Reset()         { *m = BatchResponse{} }
func (m *BatchResponse) String() string { return "" }

type TaskStatusRequest struct {
	Identifier isTaskStatusRequest_Identifier `protobuf_oneof:"identifier"`
}

func (m *TaskStatusRequest) Reset()         { *m = TaskStatusRequest{} }
func (m *TaskStatusRequest) String() string { return "" }

type TaskStatusRequest_TaskId struct {
	TaskId string `protobuf:"bytes,1,opt,name=task_id,json=taskId,proto3,oneof" json:"task_id,omitempty"`
}

type TaskStatusRequest_BatchId struct {
	BatchId string `protobuf:"bytes,2,opt,name=batch_id,json=batchId,proto3,oneof" json:"batch_id,omitempty"`
}

type isTaskStatusRequest_Identifier interface {
	isTaskStatusRequest_Identifier()
}

func (*TaskStatusRequest_TaskId) isTaskStatusRequest_Identifier()   {}
func (*TaskStatusRequest_BatchId) isTaskStatusRequest_Identifier() {}

type TaskStatusResponse struct {
	TaskId        string         `protobuf:"bytes,1,opt,name=task_id,json=taskId,proto3" json:"task_id,omitempty"`
	Status        InferenceStatus `protobuf:"varint,2,opt,name=status,proto3,enum=inference.InferenceStatus" json:"status,omitempty"`
	Progress      float32        `protobuf:"fixed32,3,opt,name=progress,proto3" json:"progress,omitempty"`
	StartedAt     time.Time      `protobuf:"bytes,4,opt,name=started_at,json=startedAt,proto3,stdtime" json:"started_at,omitempty"`
	UpdatedAt     time.Time      `protobuf:"bytes,5,opt,name=updated_at,json=updatedAt,proto3,stdtime" json:"updated_at,omitempty"`
	ErrorMessage string         `protobuf:"bytes,6,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"`
	BatchProgress *BatchProgress  `protobuf:"bytes,7,opt,name=batch_progress,json=batchProgress,proto3" json:"batch_progress,omitempty"`
}

func (m *TaskStatusResponse) Reset()         { *m = TaskStatusResponse{} }
func (m *TaskStatusResponse) String() string { return "" }

type TaskResultRequest struct {
	TaskId string `protobuf:"bytes,1,opt,name=task_id,json=taskId,proto3" json:"task_id,omitempty"`
}

func (m *TaskResultRequest) Reset()         { *m = TaskResultRequest{} }
func (m *TaskResultRequest) String() string { return "" }

type TaskResultResponse struct {
	TaskId       string                `protobuf:"bytes,1,opt,name=task_id,json=taskId,proto3" json:"task_id,omitempty"`
	Status       InferenceStatus       `protobuf:"varint,2,opt,name=status,proto3,enum=inference.InferenceStatus" json:"status,omitempty"`
	Tracks       map[string]*TrackResult `protobuf:"bytes,3,rep,name=tracks,proto3" json:"tracks,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Duration     float64               `protobuf:"fixed64,4,opt,name=duration,proto3" json:"duration,omitempty"`
	ProcessTime  float64               `protobuf:"fixed64,5,opt,name=process_time,json=processTime,proto3" json:"process_time,omitempty"`
	ErrorMessage string                `protobuf:"bytes,6,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"`
}

func (m *TaskResultResponse) Reset()         { *m = TaskResultResponse{} }
func (m *TaskResultResponse) String() string { return "" }

type CancelTaskRequest struct {
	TaskId string `protobuf:"bytes,1,opt,name=task_id,json=taskId,proto3" json:"task_id,omitempty"`
	Reason string `protobuf:"bytes,2,opt,name=reason,proto3" json:"reason,omitempty"`
}

func (m *CancelTaskRequest) Reset()         { *m = CancelTaskRequest{} }
func (m *CancelTaskRequest) String() string { return "" }

type CancelTaskResponse struct {
	TaskId         string          `protobuf:"bytes,1,opt,name=task_id,json=taskId,proto3" json:"task_id,omitempty"`
	Success        bool            `protobuf:"varint,2,opt,name=success,proto3" json:"success,omitempty"`
	PreviousStatus InferenceStatus `protobuf:"varint,3,opt,name=previous_status,json=previousStatus,proto3,enum=inference.InferenceStatus" json:"previous_status,omitempty"`
	CurrentStatus  InferenceStatus `protobuf:"varint,4,opt,name=current_status,json=currentStatus,proto3,enum=inference.InferenceStatus" json:"current_status,omitempty"`
	CancelledAt    time.Time       `protobuf:"bytes,5,opt,name=cancelled_at,json=cancelledAt,proto3,stdtime" json:"cancelled_at,omitempty"`
	Message        string          `protobuf:"bytes,6,opt,name=message,proto3" json:"message,omitempty"`
}

func (m *CancelTaskResponse) Reset()         { *m = CancelTaskResponse{} }
func (m *CancelTaskResponse) String() string { return "" }

type ModelInfoRequest struct {
	ModelType ModelType `protobuf:"varint,1,opt,name=model_type,proto3,enum=inference.ModelType" json:"model_type,omitempty"`
}

func (m *ModelInfoRequest) Reset()         { *m = ModelInfoRequest{} }
func (m *ModelInfoRequest) String() string { return "" }

type ModelInfoResponse struct {
	Models       []*ModelInfo `protobuf:"bytes,1,rep,name=models,proto3" json:"models,omitempty"`
	IsReady      bool          `protobuf:"varint,2,opt,name=is_ready,json=isReady,proto3" json:"is_ready,omitempty"`
	ActiveJobs   int32         `protobuf:"varint,3,opt,name=active_jobs,json=activeJobs,proto3" json:"active_jobs,omitempty"`
	QueueSize    int32         `protobuf:"varint,4,opt,name=queue_size,json=queueSize,proto3" json:"queue_size,omitempty"`
	MaxConcurrent int32        `protobuf:"varint,5,opt,name=max_concurrent,json=maxConcurrent,proto3" json:"max_concurrent,omitempty"`
}

func (m *ModelInfoResponse) Reset()         { *m = ModelInfoResponse{} }
func (m *ModelInfoResponse) String() string { return "" }

type HealthCheckRequest struct{}

func (m *HealthCheckRequest) Reset()         { *m = HealthCheckRequest{} }
func (m *HealthCheckRequest) String() string { return "" }

type HealthCheckResponse struct {
	Status     ServiceStatus      `protobuf:"varint,1,opt,name=status,proto3,enum=inference.ServiceStatus" json:"status,omitempty"`
	Version    string             `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
	Uptime     time.Time          `protobuf:"bytes,3,opt,name=uptime,proto3,stdtime" json:"uptime,omitempty"`
	Resources  *SystemResources   `protobuf:"bytes,4,opt,name=resources,proto3" json:"resources,omitempty"`
	Components []*ComponentHealth `protobuf:"bytes,5,rep,name=components,proto3" json:"components,omitempty"`
}

func (m *HealthCheckResponse) Reset()         { *m = HealthCheckResponse{} }
func (m *HealthCheckResponse) String() string { return "" }

// ==================== 枚举类型 ====================

type ModelType int32

const (
	ModelType_Unspecified  ModelType = 0
	ModelType_Demucs        ModelType = 1
	ModelType_HtDemucs      ModelType = 2
	ModelType_HtDemucsFt    ModelType = 3
	ModelType_HtDemucsLarge ModelType = 4
	ModelType_BsrFormer     ModelType = 5
	ModelType_Spleeter      ModelType = 6
)

type TrackType int32

const (
	TrackType_Unspecified TrackType = 0
	TrackType_Vocals      TrackType = 1
	TrackType_Drums       TrackType = 2
	TrackType_Bass        TrackType = 3
	TrackType_Other       TrackType = 4
	TrackType_Piano       TrackType = 5
	TrackType_Guitar      TrackType = 6
)

type InferenceStatus int32

const (
	InferenceStatus_Unspecified InferenceStatus = 0
	InferenceStatus_Pending     InferenceStatus = 1
	InferenceStatus_Processing  InferenceStatus = 2
	InferenceStatus_Completed   InferenceStatus = 3
	InferenceStatus_Failed      InferenceStatus = 4
	InferenceStatus_Cancelled   InferenceStatus = 5
	InferenceStatus_Timeout     InferenceStatus = 6
)

type BatchStatus int32

const (
	BatchStatus_Unspecified        BatchStatus = 0
	BatchStatus_Processing         BatchStatus = 1
	BatchStatus_Completed          BatchStatus = 2
	BatchStatus_PartiallyCompleted BatchStatus = 3
	BatchStatus_Failed             BatchStatus = 4
	BatchStatus_Cancelled          BatchStatus = 5
)

type AudioFormat int32

const (
	AudioFormat_Unspecified AudioFormat = 0
	AudioFormat_Pcm16bit    AudioFormat = 1
	AudioFormat_Pcm24bit    AudioFormat = 2
	AudioFormat_PcmFloat    AudioFormat = 3
	AudioFormat_Opus        AudioFormat = 4
)

type OutputFormat int32

const (
	OutputFormat_Unspecified OutputFormat = 0
	OutputFormat_Wav         OutputFormat = 1
	OutputFormat_Mp3         OutputFormat = 2
	OutputFormat_Flac        OutputFormat = 3
	OutputFormat_Ogg         OutputFormat = 4
)

type ServiceStatus int32

const (
	ServiceStatus_Healthy   ServiceStatus = 0
	ServiceStatus_Degraded  ServiceStatus = 1
	ServiceStatus_Unhealthy ServiceStatus = 2
)

// ==================== 复合类型 ====================

type SongItem struct {
	SongId   string            `protobuf:"bytes,1,opt,name=song_id,json=songId,proto3" json:"song_id,omitempty"`
	SongName string            `protobuf:"bytes,2,opt,name=song_name,json=songName,proto3" json:"song_name,omitempty"`
	AudioUrl string            `protobuf:"bytes,3,opt,name=audio_url,json=audioUrl,proto3" json:"audio_url,omitempty"`
	Metadata  map[string]string `protobuf:"bytes,4,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (m *SongItem) Reset()         { *m = SongItem{} }
func (m *SongItem) String() string { return "" }

type TrackResult struct {
	TrackType   TrackType        `protobuf:"varint,1,opt,name=track_type,proto3,enum=inference.TrackType" json:"track_type,omitempty"`
	FilePath    string           `protobuf:"bytes,2,opt,name=file_path,json=filePath,proto3" json:"file_path,omitempty"`
	DownloadUrl string           `protobuf:"bytes,3,opt,name=download_url,json=downloadUrl,proto3" json:"download_url,omitempty"`
	FileSize    int64            `protobuf:"varint,4,opt,name=file_size,json=fileSize,proto3" json:"file_size,omitempty"`
	Duration    float64          `protobuf:"fixed64,5,opt,name=duration,proto3" json:"duration,omitempty"`
	Metadata    map[string]string `protobuf:"bytes,6,rep,name=metadata,proto3" json:"metadata,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (m *TrackResult) Reset()         { *m = TrackResult{} }
func (m *TrackResult) String() string { return "" }

type ModelInfo struct {
	Type         ModelType        `protobuf:"varint,1,opt,name=type,proto3,enum=inference.ModelType" json:"type,omitempty"`
	Name         string           `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Version      string           `protobuf:"bytes,3,opt,name=version,proto3" json:"version,omitempty"`
	Description  string           `protobuf:"bytes,4,opt,name=description,proto3" json:"description,omitempty"`
	IsLoaded     bool             `protobuf:"varint,5,opt,name=is_loaded,json=isLoaded,proto3" json:"is_loaded,omitempty"`
	MemoryUsageMb int64            `protobuf:"varint,6,opt,name=memory_usage_mb,json=memoryUsageMb,proto3" json:"memory_usage_mb,omitempty"`
	Capabilities map[string]string `protobuf:"bytes,7,rep,name=capabilities,proto3" json:"capabilities,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (m *ModelInfo) Reset()         { *m = ModelInfo{} }
func (m *ModelInfo) String() string { return "" }

type CallbackConfig struct {
	Url            string            `protobuf:"bytes,1,opt,name=url,proto3" json:"url,omitempty"`
	Method         string            `protobuf:"bytes,2,opt,name=method,proto3" json:"method,omitempty"`
	Headers        map[string]string `protobuf:"bytes,3,rep,name=headers,proto3" json:"headers,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	RetryCount     int32             `protobuf:"varint,4,opt,name=retry_count,json=retryCount,proto3" json:"retry_count,omitempty"`
	TimeoutSeconds int32             `protobuf:"varint,5,opt,name=timeout_seconds,json=timeoutSeconds,proto3" json:"timeout_seconds,omitempty"`
}

func (m *CallbackConfig) Reset()         { *m = CallbackConfig{} }
func (m *CallbackConfig) String() string { return "" }

type BatchProgress struct {
	TotalCount       int32   `protobuf:"varint,1,opt,name=total_count,json=totalCount,proto3" json:"total_count,omitempty"`
	CompletedCount   int32   `protobuf:"varint,2,opt,name=completed_count,json=completedCount,proto3" json:"completed_count,omitempty"`
	FailedCount      int32   `protobuf:"varint,3,opt,name=failed_count,json=failedCount,proto3" json:"failed_count,omitempty"`
	ProcessingCount  int32   `protobuf:"varint,4,opt,name=processing_count,json=processingCount,proto3" json:"processing_count,omitempty"`
	PendingCount     int32   `protobuf:"varint,5,opt,name=pending_count,json=pendingCount,proto3" json:"pending_count,omitempty"`
	ProgressPercentage float32 `protobuf:"fixed32,6,opt,name=progress_percentage,json=progressPercentage,proto3" json:"progress_percentage,omitempty"`
}

func (m *BatchProgress) Reset()         { *m = BatchProgress{} }
func (m *BatchProgress) String() string { return "" }

type SystemResources struct {
	CpuUsagePercent float64 `protobuf:"fixed64,1,opt,name=cpu_usage_percent,json=cpuUsagePercent,proto3" json:"cpu_usage_percent,omitempty"`
	MemoryUsageMb   float64 `protobuf:"fixed64,2,opt,name=memory_usage_mb,json=memoryUsageMb,proto3" json:"memory_usage_mb,omitempty"`
	GpuUsagePercent float64 `protobuf:"fixed64,3,opt,name=gpu_usage_percent,json=gpuUsagePercent,proto3" json:"gpu_usage_percent,omitempty"`
	GpuMemoryMb     float64 `protobuf:"fixed64,4,opt,name=gpu_memory_mb,json=gpuMemoryMb,proto3" json:"gpu_memory_mb,omitempty"`
	DiskUsageGb     float64 `protobuf:"fixed64,5,opt,name=disk_usage_gb,json=diskUsageGb,proto3" json:"disk_usage_gb,omitempty"`
}

func (m *SystemResources) Reset()         { *m = SystemResources{} }
func (m *SystemResources) String() string { return "" }

type ComponentHealth struct {
	Name      string        `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Status    ServiceStatus `protobuf:"varint,2,opt,name=status,proto3,enum=inference.ServiceStatus" json:"status,omitempty"`
	Message   string        `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
	LatencyMs float64       `protobuf:"fixed64,4,opt,name=latency_ms,json=latencyMs,proto3" json:"latency_ms,omitempty"`
}

func (m *ComponentHealth) Reset()         { *m = ComponentHealth{} }
func (m *ComponentHealth) String() string { return "" }