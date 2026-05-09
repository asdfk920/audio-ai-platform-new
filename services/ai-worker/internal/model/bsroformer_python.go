package model

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// PythonBSRoformer 基于 Python BSRoFormer 库的模型封装
// 使用 pip install BS-RoFormer 安装
type PythonBSRoformer struct {
	name             string
	modelPath        string
	device           int
	deviceType       string // "cpu" 或 "cuda"
	segmentSize      int
	overlap          float64
	useFP16          bool
	useINT8          bool
	batchSize        int
	pythonEnv        string // Python 虚拟环境路径
	modelName        string // 模型名称（用于自动下载）
	currentInputFile string // 当前处理的输入文件
	currentOutputDir string // 当前输出目录
	currentTaskID    string // 当前任务ID

	mu             sync.RWMutex
	isLoaded       bool
	isInstalled    bool
	inferenceCount int64
	totalLatency   time.Duration
}

// PythonBSRoformerConfig Python BSRoformer 配置
type PythonBSRoformerConfig struct {
	Name        string
	ModelPath   string
	Device      int
	SegmentSize int
	Overlap     float64
	UseFP16     bool
	UseINT8     bool
	BatchSize   int
	PythonEnv   string
	ModelName   string // 如："lucidrains/bs-roformer-mel"
}

// NewPythonBSRoformer 创建 Python BSRoformer 实例
func NewPythonBSRoformer(cfg *PythonBSRoformerConfig) *PythonBSRoformer {
	return &PythonBSRoformer{
		name:        cfg.Name,
		modelPath:   cfg.ModelPath,
		device:      cfg.Device,
		segmentSize: cfg.SegmentSize,
		overlap:     cfg.Overlap,
		useFP16:     cfg.UseFP16,
		useINT8:     cfg.UseINT8,
		batchSize:   cfg.BatchSize,
		pythonEnv:   cfg.PythonEnv,
		modelName:   cfg.ModelName,
	}
}

// CheckPythonInstallation 检查 Python 和 BS-RoFormer 是否已安装
func (m *PythonBSRoformer) CheckPythonInstallation() error {
	logx.Info("检查 Python 环境...")

	// 检查 Python 是否安装
	pythonCmd := "python"
	if runtime.GOOS == "windows" {
		pythonCmd = "python.exe"
	}

	cmd := exec.Command(pythonCmd, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Python 未安装或不在 PATH 中：%w", err)
	}
	logx.Infof("Python 版本：%s", strings.TrimSpace(string(output)))

	// 检查 BS-RoFormer 是否安装
	cmd = exec.Command(pythonCmd, "-c", "import bs_roformer; print(bs_roformer.__version__ if hasattr(bs_roformer, '__version__') else 'installed')")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("BS-RoFormer 未安装：%w", err)
	}
	logx.Infof("BS-RoFormer 已安装：%s", strings.TrimSpace(string(output)))

	m.isInstalled = true
	return nil
}

// InstallBSRoformer 安装 BS-RoFormer 库
func (m *PythonBSRoformer) InstallBSRoformer() error {
	logx.Info("正在安装 BS-RoFormer...")

	pythonCmd := "python"
	if runtime.GOOS == "windows" {
		pythonCmd = "python.exe"
	}

	// 使用 pip 安装
	cmd := exec.Command(pythonCmd, "-m", "pip", "install", "BS-RoFormer", "--upgrade")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("安装 BS-RoFormer 失败：%w", err)
	}

	logx.Info("BS-RoFormer 安装完成")
	m.isInstalled = true
	return nil
}

// Load 加载模型（自动下载预训练权重）
func (m *PythonBSRoformer) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isLoaded {
		return nil
	}

	logx.Infof("开始加载 BSRoformer 模型：%s", m.modelName)
	startTime := time.Now()

	// 检查 Python 环境
	if err := m.CheckPythonInstallation(); err != nil {
		logx.Errorf("Python 环境检查失败：%v", err)
		logx.Info("尝试自动安装 BS-RoFormer...")
		if installErr := m.InstallBSRoformer(); installErr != nil {
			return fmt.Errorf("自动安装失败：%w", installErr)
		}
	}

	// 创建模型目录
	if err := os.MkdirAll(m.modelPath, 0755); err != nil {
		return fmt.Errorf("创建模型目录失败：%w", err)
	}

	// 测试模型加载
	if err := m.testModelLoad(); err != nil {
		return fmt.Errorf("模型加载测试失败：%w", err)
	}

	m.isLoaded = true
	loadTime := time.Since(startTime)
	logx.Infof("BSRoformer 模型加载完成：耗时 %v, 模型：%s", loadTime, m.modelName)
	logx.Infof("模型配置：FP16=%v, INT8=%v, 设备=%d", m.useFP16, m.useINT8, m.device)

	return nil
}

// testModelLoad 测试模型加载
func (m *PythonBSRoformer) testModelLoad() error {
	logx.Info("测试模型加载...")

	pythonScript := fmt.Sprintf(`
import torch
from bs_roformer import BSRoFormer

print("加载模型：%s")
model = BSRoFormer.from_pretrained("%s")
model.eval()

# 移动到指定设备
device = "cuda:%d" if torch.cuda.is_available() else "cpu"
model = model.to(device)

# 测试推理
print("执行测试推理...")
audio = torch.randn(1, 2, 44100 * 5).to(device)  # 5 秒音频
with torch.no_grad():
    start = time.time()
    separated = model(audio)
    elapsed = time.time() - start

print(f"测试推理完成，耗时：{elapsed:.3f}秒")
print(f"输出音轨数：{len(separated)}")
print("模型加载成功！")
`, m.modelName, m.modelName, m.device)

	pythonCmd := "python"
	if runtime.GOOS == "windows" {
		pythonCmd = "python.exe"
	}

	cmd := exec.Command(pythonCmd, "-c", pythonScript)
	cmd.Dir = m.modelPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("测试失败：%w\n输出：%s", err, string(output))
	}

	logx.Infof("测试输出：%s", string(output))
	return nil
}

// Inference 执行推理
func (m *PythonBSRoformer) Inference(audioData []float32) (*SeparationResult, error) {
	startTime := time.Now()

	result, err := m.inference(audioData)

	// 统计延迟
	if err == nil {
		m.mu.Lock()
		m.inferenceCount++
		m.totalLatency += time.Since(startTime)
		m.mu.Unlock()
	}

	return result, err
}

// inference 内部推理方法（使用独立 Python 脚本）
func (m *PythonBSRoformer) inference(audioData []float32) (*SeparationResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isLoaded {
		return nil, fmt.Errorf("模型未加载")
	}

	// 获取脚本路径
	scriptPath := filepath.Join(filepath.Dir(os.Args[0]), "..", "scripts", "bsroformer_inference.py")

	// 如果找不到相对路径，尝试绝对路径
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		scriptPath = filepath.Join(filepath.Dir(os.Args[0]), "scripts", "bsroformer_inference.py")
	}

	logx.Infof("[BS-RoFormer] 使用推理脚本: %s", scriptPath)

	pythonCmd := "python"
	if runtime.GOOS == "windows" {
		pythonCmd = "python.exe"
	}

	args := []string{
		scriptPath,
		"--input", m.currentInputFile, // 输入文件路径
		"--output", m.currentOutputDir, // 输出目录
		"--model", "bsroformer",
		"--device", fmt.Sprintf("%s:%d", m.deviceType, m.device),
		"--segment-size", fmt.Sprintf("%d", m.segmentSize),
		"--overlap", fmt.Sprintf("%.2f", m.overlap),
		"--json",
	}

	if m.useFP16 {
		args = append(args, "--fp16")
	}

	cmd := exec.Command(pythonCmd, args...)
	cmd.Env = append(os.Environ(), "KMP_DUPLICATE_LIB_OK=TRUE")
	cmd.Dir = m.modelPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("[BS-RoFormer] 推理失败: %w\n输出: %s", err, string(output))
	}

	// 解析 JSON 输出
	var result struct {
		Status     string            `json:"status"`
		UsedModel  string            `json:"used_model"`
		Tracks     map[string]string `json:"tracks"`
		TrackCount int               `json:"track_count"`
		Error      string            `json:"error"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		logx.Slowf("[BS-RoFormer] 解析JSON失败，尝试解析文本输出: %v", err)
		// 回退到简单解析
		result.Tracks = map[string]string{
			"vocals": filepath.Join(m.currentOutputDir, "vocals.wav"),
			"drums":  filepath.Join(m.currentOutputDir, "drums.wav"),
			"bass":   filepath.Join(m.currentOutputDir, "bass.wav"),
			"other":  filepath.Join(m.currentOutputDir, "other.wav"),
		}
		result.TrackCount = len(result.Tracks)
	}

	if result.Status == "error" {
		return nil, fmt.Errorf("[BS-RoFormer] 推理错误: %s", result.Error)
	}

	inferenceResult := &SeparationResult{
		TaskID:      m.currentTaskID,
		Tracks:      result.Tracks,
		Duration:    estimateAudioDuration(m.currentInputFile),
		ProcessTime: time.Since(time.Now()),
		ModelType:   ModelBSRFormer,
	}

	logx.Infof("[BS-RoFormer] 推理完成: model=%s, tracks=%d", result.UsedModel, result.TrackCount)

	return inferenceResult, nil
}

// GetAverageLatency 获取平均推理延迟
func (m *PythonBSRoformer) GetAverageLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.inferenceCount == 0 {
		return 0
	}

	return m.totalLatency / time.Duration(m.inferenceCount)
}

// GetStats 获取模型统计信息
func (m *PythonBSRoformer) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgLatency := time.Duration(0)
	if m.inferenceCount > 0 {
		avgLatency = m.totalLatency / time.Duration(m.inferenceCount)
	}

	return map[string]interface{}{
		"name":            m.name,
		"is_loaded":       m.isLoaded,
		"is_installed":    m.isInstalled,
		"use_fp16":        m.useFP16,
		"use_int8":        m.useINT8,
		"device":          m.device,
		"inference_count": m.inferenceCount,
		"avg_latency_ms":  avgLatency.Milliseconds(),
		"batch_size":      m.batchSize,
		"segment_size":    m.segmentSize,
		"model_name":      m.modelName,
	}
}

// Unload 卸载模型
func (m *PythonBSRoformer) Unload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isLoaded {
		return nil
	}

	// Python 模型不需要显式卸载
	m.isLoaded = false
	logx.Info("BSRoformer 模型已卸载")

	return nil
}

// IsReady 检查模型是否就绪
func (m *PythonBSRoformer) IsReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isLoaded
}

// GetModelInfo 获取模型信息
func (m *PythonBSRoformer) GetModelInfo() map[string]interface{} {
	return map[string]interface{}{
		"type":           "bsroformer_python",
		"name":           m.name,
		"model_name":     m.modelName,
		"install_method": "pip install BS-RoFormer",
		"is_ready":       m.IsReady(),
		"avg_latency":    fmt.Sprintf("%dms", m.GetAverageLatency().Milliseconds()),
	}
}

// GetPythonEnvInfo 获取 Python 环境信息
func (m *PythonBSRoformer) GetPythonEnvInfo() (map[string]string, error) {
	pythonCmd := "python"
	if runtime.GOOS == "windows" {
		pythonCmd = "python.exe"
	}

	// 获取 Python 版本
	cmd := exec.Command(pythonCmd, "--version")
	versionOutput, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	// 获取 torch 版本
	cmd = exec.Command(pythonCmd, "-c", "import torch; print(torch.__version__)")
	torchOutput, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	// 获取 bs_roformer 版本
	cmd = exec.Command(pythonCmd, "-c", "import bs_roformer; print('installed')")
	bsOutput, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	// 检查 CUDA 可用性
	cudaOutput, err := exec.Command(pythonCmd, "-c", "import torch; print(torch.cuda.is_available())").CombinedOutput()
	cudaAvailable := "false"
	if err == nil {
		cudaAvailable = strings.TrimSpace(string(cudaOutput))
	}

	return map[string]string{
		"python_version": strings.TrimSpace(string(versionOutput)),
		"torch_version":  strings.TrimSpace(string(torchOutput)),
		"bs_roformer":    strings.TrimSpace(string(bsOutput)),
		"cuda_available": cudaAvailable,
	}, nil
}
