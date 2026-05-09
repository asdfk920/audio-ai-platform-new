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

	if _, err := os.Stat(job.InputPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("输入文件不存在: %s", job.InputPath)
	}

	outputDir := filepath.Join(s.config.OutputDir, job.ID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	progressCallback(10)

	args := []string{
		"-n", "htdemucs",
		"--two-stems", "vocals",
		"--segment", "10",
		"--overlap", "0.25",
		"-o", outputDir,
	}

	if model.UseFP16 {
		args = append(args, "--float16")
	}

	deviceStr := "cpu"
	if model.Device >= 0 {
		deviceStr = fmt.Sprintf("cuda:%d", model.Device)
	}
	args = append(args, "-d", deviceStr)

	args = append(args, job.InputPath)

	s.logger.Infof("[Demucs] 执行命令: demucs %v", args)

	progressCallback(20)

	cmd := exec.Command("cmd", "/c", "demucs")

	cmd.Args = append(cmd.Args, args...)

	// 关键：1. 强制设置环境变量，解决 OpenMP 冲突
	cmd.Env = append(os.Environ(), "KMP_DUPLICATE_LIB_OK=TRUE")

	// 2. 把 Demucs 的标准输出和错误，直接打印到 Go 服务的控制台
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	startTime := time.Now()

	err := cmd.Run()
	processTime := time.Since(startTime)

	if err != nil {
		s.logger.Errorf("[Demucs] 推理失败: %v, 耗时=%v", err, processTime)
		return nil, fmt.Errorf("Demucs 推理失败: %w", err)
	}

	progressCallback(90)

	tracks := make(map[string]string)

	inputFileName := strings.TrimSuffix(filepath.Base(job.InputPath), filepath.Ext(job.InputPath))
	separatedDirs := []string{
		filepath.Join(outputDir, "htdemucs", inputFileName),
		filepath.Join(outputDir, inputFileName),
		outputDir,
	}

	for _, separatedDir := range separatedDirs {
		if _, statErr := os.Stat(separatedDir); statErr != nil {
			continue
		}

		expectedTracks := []TrackType{TrackVocals, TrackDrums, TrackBass, TrackOther}
		for _, trackType := range expectedTracks {
			trackPath := filepath.Join(separatedDir, string(trackType)+".wav")
			if _, err := os.Stat(trackPath); err == nil {
				tracks[string(trackType)] = trackPath
				s.logger.Infof("[Demucs] 生成音轨: %s -> %s", trackType, trackPath)
			}
		}

		if len(tracks) > 0 {
			break
		}

		trackedFiles, _ := os.ReadDir(separatedDir)
		for _, file := range trackedFiles {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".wav") {
				name := strings.TrimSuffix(file.Name(), ".wav")
				tracks[name] = filepath.Join(separatedDir, file.Name())
			}
		}

		if len(tracks) > 0 {
			break
		}
	}

	if len(tracks) == 0 {
		s.logger.Errorf("[Demucs] 未找到输出文件，检查目录: %s", outputDir)
		filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				s.logger.Infof("[Demucs] 发现文件: %s (%d bytes)", path, info.Size())
			}
			return nil
		})
	}

	duration := estimateAudioDuration(job.InputPath)
	progressCallback(100)

	result := &SeparationResult{
		TaskID:      job.ID,
		Tracks:      tracks,
		Duration:    duration,
		ProcessTime: processTime,
		ModelType:   ModelHTDemucs,
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

	cmd := exec.Command("cmd", "/c", "spleeter")

	cmd.Args = append(cmd.Args, args...)

	// 关键：1. 强制设置环境变量，解决 OpenMP 冲突
	cmd.Env = append(os.Environ(), "KMP_DUPLICATE_LIB_OK=TRUE")

	// 2. 把 Spleeter 的标准输出和错误，直接打印到 Go 服务的控制台
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
