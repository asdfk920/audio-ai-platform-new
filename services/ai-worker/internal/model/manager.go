package model

import (
	"encoding/json"
	"fmt"
	"os"
)

type ModelManager struct {
	services map[string]*AIService
	configs  map[string]*ModelConfig
}

var globalModelManager *ModelManager

func InitModelManager(configPath string) error {
	config, err := LoadModelConfig(configPath)
	if err != nil {
		return fmt.Errorf("加载模型配置失败: %w", err)
	}

	globalModelManager = &ModelManager{
		services: make(map[string]*AIService),
		configs:  make(map[string]*ModelConfig),
	}

	for name, cfg := range config {
		service, err := NewAIService(cfg)
		if err != nil {
			return fmt.Errorf("初始化模型服务 %s 失败: %w", name, err)
		}
		globalModelManager.services[name] = service
		globalModelManager.configs[name] = cfg
	}

	return nil
}

func GetModelService(name string) (*AIService, bool) {
	if globalModelManager == nil {
		return nil, false
	}
	svc, ok := globalModelManager.services[name]
	return svc, ok
}

func GetDefaultModelService() (*AIService, bool) {
	if globalModelManager == nil {
		return nil, false
	}
	for _, svc := range globalModelManager.services {
		return svc, true
	}
	return nil, false
}

func StopAllModels() {
	if globalModelManager == nil {
		return
	}
	for name, svc := range globalModelManager.services {
		svc.Stop()
		fmt.Printf("已停止模型服务: %s\n", name)
	}
}

type ConfigFile struct {
	Default string                  `json:"default"`
	Models  map[string]*ModelConfig `json:"models"`
}

func LoadModelConfig(path string) (map[string]*ModelConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultModelConfigs(), nil
	}

	var config ConfigFile
	if err := json.Unmarshal(data, &config); err != nil {
		return DefaultModelConfigs(), nil
	}

	if len(config.Models) > 0 {
		return config.Models, nil
	}

	return DefaultModelConfigs(), nil
}

func DefaultModelConfigs() map[string]*ModelConfig {
	baseDir := os.Getenv("MODEL_BASE_DIR")
	if baseDir == "" {
		baseDir = "./models"
	}

	outputDir := os.Getenv("OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "./output"
	}

	cacheDir := os.Getenv("CACHE_DIR")
	if cacheDir == "" {
		cacheDir = "./cache"
	}

	gpuDevice := -1
	if env := os.Getenv("GPU_DEVICE"); env != "" {
		fmt.Sscanf(env, "%d", &gpuDevice)
	}

	maxConcurrency := 2
	if env := os.Getenv("MAX_CONCURRENT_JOBS"); env != "" {
		fmt.Sscanf(env, "%d", &maxConcurrency)
	}

	return map[string]*ModelConfig{
		"bsroformer": {
			Type:              ModelBSRFormer,
			Name:              "bss_roformer",
			ModelPath:         baseDir,
			GPUDevice:         gpuDevice,
			SegmentSize:       10,
			Overlap:           0.25,
			UseFP16:           false,
			BatchSize:         1,
			OutputDir:         outputDir,
			CacheDir:          cacheDir,
			MaxConcurrentJobs: maxConcurrency,
		},
	}
}
