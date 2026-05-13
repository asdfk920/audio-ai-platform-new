package logic

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"

	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/model"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
)

// AudioSeparateReq 音频分离请求
type AudioSeparateReq struct {
	ContentID int64 `json:"content_id"`
}

// AudioSeparateResp 音频分离响应
type AudioSeparateResp struct {
	TaskID    string `json:"task_id"`
	ContentID int64  `json:"content_id"`
	AudioURL  string `json:"audio_url"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

// AudioSeparateLogic 音频分离逻辑
type AudioSeparateLogic struct {
	svcCtx *svc.ServiceContext
}

// NewAudioSeparateLogic 创建音频分离逻辑实例
func NewAudioSeparateLogic(svcCtx *svc.ServiceContext) *AudioSeparateLogic {
	return &AudioSeparateLogic{
		svcCtx: svcCtx,
	}
}

// SeparateAudio 根据 content_id 发起音频分离
func (l *AudioSeparateLogic) SeparateAudio(r *AudioSeparateReq, userID int64) (*AudioSeparateResp, error) {
	// 1. 验证 content_id
	if r.ContentID <= 0 {
		return nil, fmt.Errorf("content_id 无效")
	}

	// 2. 查询 content 表获取 audio_url
	audioURL, err := l.getAudioURLByContentID(r.ContentID)
	if err != nil {
		return nil, fmt.Errorf("查询音频信息失败：%w", err)
	}

	// 3. 生成任务 ID
	taskID := fmt.Sprintf("task_%d_%d_%d", userID, r.ContentID, time.Now().UnixNano())

	// 4. 保存任务到数据库
	err = l.saveTaskToDB(taskID, userID, r.ContentID, audioURL)
	if err != nil {
		return nil, fmt.Errorf("保存任务失败：%w", err)
	}

	// 5. 异步发起 AI 分离任务
	go l.processSeparation(taskID, audioURL)

	resp := &AudioSeparateResp{
		TaskID:    taskID,
		ContentID: r.ContentID,
		AudioURL:  audioURL,
		Status:    "pending",
		Message:   "分离任务已创建，正在处理中",
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	return resp, nil
}

// processSeparation 处理音频分离（异步执行）
func (l *AudioSeparateLogic) processSeparation(taskID, audioURL string) {
	// 1. 更新任务状态为 processing
	l.updateTaskStatus(taskID, "processing", 10, "正在下载音频文件...", "")

	// 2. 下载音频文件
	localPath, err := l.downloadAudio(taskID, audioURL)
	if err != nil {
		l.updateTaskStatus(taskID, "failed", 0, fmt.Sprintf("下载音频失败：%v", err), "")
		return
	}
	defer os.Remove(localPath) // 完成后清理

	// 3. 更新进度
	l.updateTaskStatus(taskID, "processing", 30, "音频下载完成，正在分离...", "")

	// 4. 调用 AI 模型进行分离
	result, err := l.callAIModel(localPath, taskID)
	if err != nil {
		l.updateTaskStatus(taskID, "failed", 50, fmt.Sprintf("AI 分离失败：%v", err), "")
		return
	}

	// 5. 更新进度
	l.updateTaskStatus(taskID, "processing", 80, "分离完成，正在保存结果...", "")

	// 6. 保存结果并生成 ZIP
	resultURL, err := l.saveSeparationResult(taskID, result)
	if err != nil {
		l.updateTaskStatus(taskID, "failed", 80, fmt.Sprintf("保存结果失败：%v", err), "")
		return
	}

	// 7. 更新任务状态为 completed
	l.updateTaskStatus(taskID, "completed", 100, "分离完成", resultURL)
}

// downloadAudio 下载音频文件
func (l *AudioSeparateLogic) downloadAudio(taskID, audioURL string) (string, error) {
	// 创建临时目录
	tempDir := filepath.Join(os.TempDir(), "audio-separation", taskID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("创建临时目录失败：%w", err)
	}

	// 从 URL 下载文件
	resp, err := http.Get(audioURL)
	if err != nil {
		return "", fmt.Errorf("下载请求失败：%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败，状态码：%d", resp.StatusCode)
	}

	// 保存文件
	filename := filepath.Join(tempDir, "input.wav")
	file, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("创建文件失败：%w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("写入文件失败：%w", err)
	}

	return filename, nil
}

// callAIModel 调用 AI 模型进行分离
func (l *AudioSeparateLogic) callAIModel(inputPath, taskID string) (*model.SeparationResult, error) {
	// 使用 ServiceContext 中的 AI 服务
	aiService := l.svcCtx.AIService
	if aiService == nil {
		return nil, fmt.Errorf("AI 服务未初始化")
	}

	// 创建分离任务
	job := &model.SeparationJob{
		ID:        taskID,
		InputPath: inputPath,
		Config:    aiService.GetConfig(),
		Result:    make(chan *model.SeparationResult, 1),
		Error:     make(chan error, 1),
		Progress:  make(chan float64, 10),
	}

	// 提交任务
	if err := aiService.SubmitJob(job); err != nil {
		return nil, fmt.Errorf("提交任务失败：%w", err)
	}

	// 等待结果
	select {
	case result := <-job.Result:
		return result, nil
	case err := <-job.Error:
		return nil, err
	case <-time.After(10 * time.Minute): // 超时处理
		return nil, fmt.Errorf("分离超时")
	}
}

// saveSeparationResult 保存分离结果并生成 ZIP
func (l *AudioSeparateLogic) saveSeparationResult(taskID string, result *model.SeparationResult) (string, error) {
	// 创建结果目录
	resultsDir := filepath.Join("static-media", "results", taskID)
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return "", fmt.Errorf("创建结果目录失败：%w", err)
	}

	// 复制分离后的音轨文件
	trackPaths := make(map[string]string)
	for trackType, trackPath := range result.Tracks {
		// 复制到结果目录
		filename := filepath.Join(resultsDir, fmt.Sprintf("%s.wav", trackType))

		// 读取源文件
		data, err := os.ReadFile(trackPath)
		if err != nil {
			return "", fmt.Errorf("读取音轨文件失败：%w", err)
		}

		// 写入目标文件
		if err := os.WriteFile(filename, data, 0644); err != nil {
			return "", fmt.Errorf("写入音轨文件失败：%w", err)
		}

		trackPaths[trackType] = filename
	}

	// 生成 ZIP 文件
	zipPath := filepath.Join(resultsDir, fmt.Sprintf("%s.zip", taskID))
	if err := createZipFile(zipPath, trackPaths); err != nil {
		return "", fmt.Errorf("生成 ZIP 文件失败：%w", err)
	}

	// 返回结果 URL（相对于 static-media 目录）
	resultURL := filepath.Join("/static-media/results", taskID, fmt.Sprintf("%s.zip", taskID))
	return resultURL, nil
}

// updateTaskStatus 更新任务状态
func (l *AudioSeparateLogic) updateTaskStatus(taskID, status string, progress int, message, resultURL string) error {
	dbConfig := l.svcCtx.Config.Database

	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.User,
		dbConfig.Password,
		dbConfig.DBName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("连接数据库失败：%w", err)
	}
	defer db.Close()

	query := `
		UPDATE audio_separation_tasks
		SET status = $1, progress = $2, message = $3, result_url = $4, updated_at = $5
		WHERE task_id = $6
	`

	now := time.Now()
	_, err = db.Exec(query, status, progress, message, resultURL, now, taskID)
	return err
}

// createZipFile 创建 ZIP 文件
func createZipFile(zipPath string, files map[string]string) error {
	// 这里简化实现，实际需要使用 zip 库
	// 可以使用 github.com/mholt/archiver 或标准库 archive/zip
	return nil // TODO: 实现 ZIP 生成
}

// saveTaskToDB 保存任务到数据库
func (l *AudioSeparateLogic) saveTaskToDB(taskID string, userID, contentID int64, audioURL string) error {
	dbConfig := l.svcCtx.Config.Database

	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.User,
		dbConfig.Password,
		dbConfig.DBName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("连接数据库失败：%w", err)
	}
	defer db.Close()

	query := `
		INSERT INTO audio_separation_tasks (task_id, user_id, content_id, audio_url, status, progress, message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	now := time.Now()
	_, err = db.Exec(query, taskID, userID, contentID, audioURL, "pending", 0, "任务已创建", now, now)
	return err
}

// getAudioURLByContentID 根据 content_id 查询音频 URL
func (l *AudioSeparateLogic) getAudioURLByContentID(contentID int64) (string, error) {
	// 从配置中获取数据库连接
	dbConfig := l.svcCtx.Config.Database

	// 构建连接字符串
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.User,
		dbConfig.Password,
		dbConfig.DBName,
	)

	// 连接数据库
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return "", fmt.Errorf("连接数据库失败：%w", err)
	}
	defer db.Close()

	// 查询 audio_url
	var audioURL string
	query := "SELECT audio_url FROM content WHERE id = $1 AND is_deleted = 0 AND status = 1"
	err = db.QueryRow(query, contentID).Scan(&audioURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("音频不存在或已删除")
		}
		return "", fmt.Errorf("查询失败：%w", err)
	}

	// 如果是相对路径，拼接完整 URL
	if len(audioURL) > 0 && audioURL[0] == '/' {
		// 假设 content 服务在 8000 端口
		audioURL = "http://127.0.0.1:8000" + audioURL
	}

	return audioURL, nil
}
