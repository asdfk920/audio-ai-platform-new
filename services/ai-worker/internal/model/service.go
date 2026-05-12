package model

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type ModelType string

const (
	ModelBSRFormer ModelType = "bsroformer"
)

type TrackType string

const (
	TrackVocals TrackType = "vocals"
	TrackDrums  TrackType = "drums"
	TrackBass   TrackType = "bass"
	TrackOther  TrackType = "other"
)

type SeparationResult struct {
	TaskID      string            `json:"task_id"`
	Tracks      map[string]string `json:"tracks"`   // track type -> file path
	Duration    float64           `json:"duration"` // audio duration in seconds
	ProcessTime time.Duration     `json:"process_time"`
	ModelType   ModelType         `json:"model_type"`
}

type ModelConfig struct {
	Type              ModelType `json:"type"`
	Name              string    `json:"name"`
	ModelPath         string    `json:"model_path"`
	GPUDevice         int       `json:"gpu_device"`
	SegmentSize       int       `json:"segment_size"` // in seconds, default 10
	Overlap           float64   `json:"overlap"`      // overlap ratio, default 0.25
	UseFP16           bool      `json:"use_fp16"`     // use FP16 precision
	UseINT8           bool      `json:"use_int8"`     // use INT8 quantization
	BatchSize         int       `json:"batch_size"`   // batch size for inference
	OutputDir         string    `json:"output_dir"`
	CacheDir          string    `json:"cache_dir"`
	MaxConcurrentJobs int       `json:"max_concurrent_jobs"`
}

type AIService struct {
	config *ModelConfig
	logger logx.Logger

	mu      sync.RWMutex
	models  map[ModelType]interface{}
	isReady bool

	jobQueue chan *SeparationJob
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

type SeparationJob struct {
	ID        string
	InputPath string
	Config    *ModelConfig
	Result    chan *SeparationResult
	Error     chan error
	Progress  chan float64
}

func NewAIService(config *ModelConfig) (*AIService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	svc := &AIService{
		config:   config,
		logger:   logx.WithContext(ctx),
		models:   make(map[ModelType]interface{}),
		ctx:      ctx,
		cancel:   cancel,
		jobQueue: make(chan *SeparationJob, config.MaxConcurrentJobs*2),
	}

	if err := svc.ensureDirectories(); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	if err := svc.initializeModels(); err != nil {
		svc.logger.Errorf("模型初始化警告: %v", err)
	} else {
		svc.isReady = true
	}

	svc.startWorkers()

	return svc, nil
}

func (s *AIService) ensureDirectories() error {
	dirs := []string{
		s.config.OutputDir,
		s.config.CacheDir,
		filepath.Join(s.config.ModelPath, s.config.Type.String()),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
	}
	return nil
}

func (s *AIService) initializeModels() error {
	s.logger.Infof("正在初始化模型：type=%s, name=%s", s.config.Type, s.config.Name)

	switch s.config.Type {
	case ModelBSRFormer:
		return s.initBSRoformerModel()
	default:
		return fmt.Errorf("不支持的模型类型：%s", s.config.Type)
	}
}

func (s *AIService) initBSRoformerModel() error {
	s.logger.Info("初始化 BS-RoFormer 模型（Python 实现）...")

	modelPath := s.config.ModelPath
	if modelPath == "" {
		modelPath = "./models"
	}

	pythonBSR := NewPythonBSRoformer(&PythonBSRoformerConfig{
		Name:        "BS-RoFormer",
		ModelPath:   modelPath,
		Device:      s.config.GPUDevice,
		SegmentSize: int(s.config.SegmentSize),
		Overlap:     s.config.Overlap,
		UseFP16:     s.config.UseFP16,
		UseINT8:     s.config.UseINT8,
		BatchSize:   s.config.BatchSize,
		ModelName:   s.config.Name,
	})

	if err := pythonBSR.Load(); err != nil {
		s.logger.Errorf("加载 BS-RoFormer 模型失败：%v", err)
		return fmt.Errorf("加载 BS-RoFormer 模型失败：%w", err)
	}

	s.models[ModelBSRFormer] = pythonBSR
	s.logger.Infof("✅ BS-RoFormer 模型初始化完成（Python 实现）")
	s.logger.Infof("模型配置：Name=%s, SegmentSize=%d, FP16=%v, GPUDevice=%d",
		s.config.Name, s.config.SegmentSize, s.config.UseFP16, s.config.GPUDevice)

	return nil
}

func (s *AIService) startWorkers() {
	for i := 0; i < s.config.MaxConcurrentJobs; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
	s.logger.Infof("启动了 %d 个推理工作进程", s.config.MaxConcurrentJobs)
}

func (s *AIService) worker(workerID int) {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return

		case job := <-s.jobQueue:
			s.logger.Infof("[Worker-%d] 开始处理任务: task_id=%s", workerID, job.ID)
			result, err := s.processJob(job)
			if err != nil {
				job.Error <- err
			} else {
				job.Result <- result
			}
			s.logger.Infof("[Worker-%d] 任务完成: task_id=%s", workerID, job.ID)
		}
	}
}

func (s *AIService) processJob(job *SeparationJob) (*SeparationResult, error) {
	startTime := time.Now()

	progressCallback := func(p float64) {
		select {
		case job.Progress <- p:
		default:
		}
	}

	result, err := s.runBSRoformerInference(job, progressCallback)
	if err != nil {
		return nil, fmt.Errorf("BS-RoFormer 推理失败: %w", err)
	}

	result.ProcessTime = time.Since(startTime)
	return result, nil
}

func (s *AIService) Separate(taskID, inputPath string) (*SeparationResult, error) {
	if !s.isReady {
		return nil, fmt.Errorf("AI 服务未就绪，请稍后重试")
	}

	job := &SeparationJob{
		ID:        taskID,
		InputPath: inputPath,
		Config:    s.config,
		Result:    make(chan *SeparationResult, 1),
		Error:     make(chan error, 1),
		Progress:  make(chan float64, 100),
	}

	select {
	case s.jobQueue <- job:
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("任务队列已满，请稍后重试")
	}

	select {
	case result := <-job.Result:
		return result, nil
	case err := <-job.Error:
		return nil, err
	case <-time.After(10 * time.Minute):
		return nil, fmt.Errorf("任务处理超时")
	}
}

func (s *AIService) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isReady
}

func (s *AIService) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"type":       s.config.Type,
		"name":       s.config.Name,
		"model_path": s.config.ModelPath,
		"gpu_device": s.config.GPUDevice,
		"use_fp16":   s.config.UseFP16,
		"is_ready":   s.isReady,
		"queue_size": len(s.jobQueue),
		"capacity":   cap(s.jobQueue),
	}
}

// GetConfig 返回模型配置
func (s *AIService) GetConfig() *ModelConfig {
	return s.config
}

// SubmitJob 提交分离任务到队列
func (s *AIService) SubmitJob(job *SeparationJob) error {
	select {
	case s.jobQueue <- job:
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("任务队列已满，请稍后重试")
	}
}

func (s *AIService) Stop() {
	s.cancel()
	close(s.jobQueue)
	s.wg.Wait()
	s.logger.Info("AI 服务已停止")
}

func (m ModelType) String() string {
	return string(m)
}
