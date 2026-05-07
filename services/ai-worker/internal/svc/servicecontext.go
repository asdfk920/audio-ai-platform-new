package svc

import (
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/config"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/model"
	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config    config.Config
	AIService *model.AIService
}

func NewServiceContext(c config.Config) (*ServiceContext, error) {
	logx.Info("Initializing AI Worker ServiceContext...")

	aiSvc, err := initializeAIService(c)
	if err != nil {
		logx.Errorf("Failed to initialize AI service: %v", err)
		return nil, err
	}

	svcCtx := &ServiceContext{
		Config:    c,
		AIService: aiSvc,
	}

	logx.Info("AI Worker ServiceContext initialized successfully")
	return svcCtx, nil
}

func initializeAIService(c config.Config) (*model.AIService, error) {
	modelConfig := &model.ModelConfig{
		Type:              model.ModelTypeHTDemucs,
		ModelPath:         c.AI.ModelPath,
		GPUDevice:         c.AI.GPUDevice,
		SegmentSize:       c.AI.SegmentSize,
		Overlap:           c.AI.Overlap,
		UseFP16:           c.AI.UseFP16,
		BatchSize:         1,
		OutputDir:         c.AI.OutputDir,
		CacheDir:          c.AI.CacheDir,
		MaxConcurrentJobs: c.AI.MaxConcurrentJobs,
	}

	svc, err := model.NewAIService(modelConfig)
	if err != nil {
		return nil, err
	}

	return svc, nil
}
