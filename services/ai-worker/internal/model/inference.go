package model

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type DemucsModel struct {
	Name        string
	ModelPath   string
	Device      int
	SegmentSize int
	Overlap     float64
	UseFP16     bool
	BatchSize   int
}

type SpleeterModel struct {
	Name      string
	ModelPath string
	Device    int
}

func (s *AIService) runDemucsInference(job *SeparationJob, progressCallback func(float64)) (*SeparationResult, error) {
	model, ok := s.models[job.Config.Type].(*DemucsModel)
	if !ok {
		return nil, fmt.Errorf("Demucs 模型未初始化")
	}

	s.logger.Infof("[Demucs] 开始推理: task_id=%s, input=%s", job.ID, job.InputPath)

	outputDir := filepath.Join(s.config.OutputDir, job.ID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	progressCallback(10)

	args := []string{
		"-n", string(job.Config.Type),
		"--two-stems=vocals",
		fmt.Sprintf("-o%s", outputDir),
	}

	if model.UseFP16 {
		args = append(args, "--float16")
	}

	if model.SegmentSize > 0 {
		args = append(args, fmt.Sprintf("--segment=%d", model.SegmentSize))
	}

	if model.Overlap > 0 {
		args = append(args, fmt.Sprintf("--overlap=%.2f", model.Overlap))
	}

	if model.Device >= 0 {
		args = append(args, fmt.Sprintf("--device=cuda:%d", model.Device))
	}

	args = append(args, job.InputPath)

	progressCallback(20)

	cmd := exec.CommandContext(s.ctx, "demucs", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	startTime := time.Now()
	err := cmd.Run()
	processTime := time.Since(startTime)

	if err != nil {
		return nil, fmt.Errorf("Demucs 推理失败: %w", err)
	}

	progressCallback(90)

	tracks := make(map[string]string)
	separatedDir := filepath.Join(outputDir, strings.TrimSuffix(filepath.Base(job.InputPath), filepath.Ext(job.InputPath)))

	expectedTracks := []TrackType{TrackVocals, TrackDrums, TrackBass, TrackOther}
	for _, trackType := range expectedTracks {
		trackPath := filepath.Join(separatedDir, string(trackType)+".wav")
		if _, err := os.Stat(trackPath); err == nil {
			tracks[string(trackType)] = trackPath
			s.logger.Infof("[Demucs] 生成音轨: %s -> %s", trackType, trackPath)
		}
	}

	if len(tracks) == 0 {
		trackedFiles, _ := os.ReadDir(separatedDir)
		for _, file := range trackedFiles {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".wav") {
				name := strings.TrimSuffix(file.Name(), ".wav")
				tracks[name] = filepath.Join(separatedDir, file.Name())
			}
		}
	}

	duration := estimateAudioDuration(job.InputPath)
	progressCallback(100)

	result := &SeparationResult{
		TaskID:     job.ID,
		Tracks:     tracks,
		Duration:   duration,
		ProcessTime: processTime,
		ModelType:  job.Config.Type,
	}

	s.logger.Infof("[Demucs] 推理完成: task_id=%s, tracks=%d, time=%v", 
		job.ID, len(tracks), processTime)

	return result, nil
}

func (s *AIService) runSpleeterInference(job *SeparationJob, progressCallback func(float64)) (*SeparationResult, error) {
	model, ok := s.models[job.Config.Type].(*SpleeterModel)
	if !ok {
		return nil, fmt.Errorf("Spleeter 模型未初始化")
	}

	s.logger.Infof("[Spleeter] 开始推理: task_id=%s, input=%s", job.ID, job.InputPath)

	outputDir := filepath.Join(s.config.OutputDir, job.ID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	progressCallback(10)

	args := []string{
		"separate",
		job.InputPath,
		"-p", "spleeter:4stems",
		"-o", outputDir,
		"-f", "wav",
	}

	if model.Device >= 0 {
		args = append(args, "--device", fmt.Sprintf("/gpu:%d", model.Device))
	}

	progressCallback(20)

	cmd := exec.CommandContext(s.ctx, "spleeter", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	startTime := time.Now()
	err := cmd.Run()
	processTime := time.Since(startTime)

	if err != nil {
		return nil, fmt.Errorf("Spleeter 推理失败: %w", err)
	}

	progressCallback(90)

	tracks := make(map[string]string)
	separatedDir := filepath.Join(outputDir, "separated_audio", "spleeter:4stems", 
		strings.TrimSuffix(filepath.Base(job.InputPath), filepath.Ext(job.InputPath)))

	trackMapping := map[string]string{
		"vocals": string(TrackVocals),
		"drums":  string(TrackDrums),
		"bass":   string(TrackBass),
		"other":  string(TrackOther),
	}

	for spleeterName, standardName := range trackMapping {
		trackPath := filepath.Join(separatedDir, spleeterName+".wav")
		if _, err := os.Stat(trackPath); err == nil {
			tracks[standardName] = trackPath
			s.logger.Infof("[Spleeter] 生成音轨: %s -> %s", spleeterName, trackPath)
		}
	}

	duration := estimateAudioDuration(job.InputPath)
	progressCallback(100)

	result := &SeparationResult{
		TaskID:      job.ID,
		Tracks:      tracks,
		Duration:    duration,
		ProcessTime: processTime,
		ModelType:   job.Config.Type,
	}

	s.logger.Infof("[Spleeter] 推理完成: task_id=%s, tracks=%d, time=%v", 
		job.ID, len(tracks), processTime)

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