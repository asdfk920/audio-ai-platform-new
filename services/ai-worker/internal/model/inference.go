package model

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (s *AIService) runBSRoformerInference(job *SeparationJob, progressCallback func(float64)) (*SeparationResult, error) {
	s.logger.Infof("[BS-RoFormer] 开始推理：task_id=%s, input=%s", job.ID, job.InputPath)

	model, ok := s.models[ModelBSRFormer].(*PythonBSRoformer)
	if !ok {
		return nil, fmt.Errorf("BS-RoFormer 模型未初始化或类型错误")
	}

	if _, err := os.Stat(job.InputPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("输入文件不存在: %s", job.InputPath)
	}

	outputDir := filepath.Join(s.config.OutputDir, job.ID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	progressCallback(10)

	model.SetCurrentTask(job.ID, job.InputPath, outputDir)

	result, err := model.Inference(nil)
	if err != nil {
		s.logger.Errorf("[BS-RoFormer] 推理失败: %v", err)
		return nil, fmt.Errorf("BS-RoFormer 推理失败: %w", err)
	}

	progressCallback(100)

	s.logger.Infof("[BS-RoFormer] 推理完成：task_id=%s, tracks=%d, time=%v",
		job.ID, len(result.Tracks), result.ProcessTime)

	return result, nil
}

func estimateAudioDuration(audioPath string) float64 {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries",
		"format=duration", "-of", "default=noprint_wrappers=1:nokey=1", audioPath)

	output, err := cmd.Output()
	if err != nil {
		return 180.0
	}

	var duration float64
	fmt.Sscanf(strings.TrimSpace(string(output)), "%f", &duration)

	if duration <= 0 {
		duration = 180.0
	}

	return duration
}
