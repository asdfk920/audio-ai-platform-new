package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jacklau/audio-ai-platform/pkg/jwtx"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/model"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	HandshakeTimeout: 10 * time.Second,
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
}

// AudioSeparateWSHandler WebSocket 音轨分离处理
func AudioSeparateWSHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[WebSocket] 收到连接请求：%s %s", r.Method, r.URL.Path)

		var tokenString string
		var userID int64 = 0

		tokenString = r.URL.Query().Get("token")
		if tokenString == "" {
			tokenString = r.Header.Get("Authorization")
			if strings.HasPrefix(tokenString, "Bearer ") {
				tokenString = strings.TrimPrefix(tokenString, "Bearer ")
			}
		}

		if tokenString != "" {
			claims, err := jwtx.ParseAccessToken(svcCtx.Config.Auth.AccessSecret, tokenString)
			if err == nil && claims != nil {
				userID = claims.UserID
				log.Printf("[WebSocket] 认证成功：user_id=%d", userID)
			} else {
				log.Printf("[WebSocket] 认证失败：token无效或过期, error=%v", err)
			}
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[WebSocket] 升级失败：%v", err)
			http.Error(w, "WebSocket upgrade failed", http.StatusBadRequest)
			return
		}
		defer conn.Close()

		taskID := r.URL.Query().Get("task_id")
		if taskID == "" {
			taskID = fmt.Sprintf("task_%d", time.Now().UnixNano())
		}

		log.Printf("[WebSocket] 连接成功：task_id=%s, user_id=%d", taskID, userID)

		session := &Session{
			conn:   conn,
			taskID: taskID,
			userID: userID,
			svcCtx: svcCtx,
		}

		if tokenString == "" || userID == 0 {
			log.Printf("[WebSocket] ⚠️ 未提供有效token，发送登录提示")
			session.sendMessage(WSMessage{
				Type:    "auth_required",
				Message: "用户未登录或者是登录过期，请重新登录后再试",
				Data: map[string]interface{}{
					"task_id":    taskID,
					"reason":     "token_missing_or_invalid",
					"suggestion": "请重新获取token后连接WebSocket",
				},
			})
		}

		session.handleConnection()
	}
}

type Session struct {
	conn    *websocket.Conn
	taskID  string
	userID  int64
	svcCtx  *svc.ServiceContext
	mu      sync.Mutex
	tempDir string // 临时目录路径（用于下载的音频文件）
}

// WSMessage 通用消息格式
type WSMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// AudioContentInfo 音频内容信息
type AudioContentInfo struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	AudioURL string `json:"audio_url"`
	Duration int64  `json:"duration_sec"`
	Artist   string `json:"artist"`
}

// handleConnection 处理连接和消息
func (s *Session) handleConnection() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[WebSocket] 发生 panic：task_id=%s, error=%v", s.taskID, r)
			s.sendError(fmt.Sprintf("服务器内部错误：%v", r))
		}
	}()

	db := s.getDB()
	if db == nil {
		s.sendError("数据库连接失败")
		return
	}
	defer db.Close()

	s.conn.SetReadLimit(65536)
	s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	s.conn.SetPongHandler(func(appData string) error {
		log.Printf("[WebSocket] 收到 pong: task_id=%s", s.taskID)
		s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	go s.heartbeat()

	audioList := s.getAudioList(db)
	s.sendAudioList(audioList)

	for {
		messageType, message, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] 连接异常关闭：task_id=%s, err=%v", s.taskID, err)
			} else {
				log.Printf("[WebSocket] 连接正常关闭：task_id=%s", s.taskID)
			}
			break
		}

		s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		if messageType != websocket.TextMessage {
			continue
		}

		var msg WSMessage
		if json.Unmarshal(message, &msg) != nil {
			s.sendError("JSON 解析失败")
			continue
		}

		switch msg.Type {
		case "separate":
			s.handleSeparate(msg.Data, db)
		case "get_audio_list":
			audioList := s.getAudioList(db)
			s.sendAudioList(audioList)
		case "cancel":
			s.handleCancel(db)
		case "ping":
			s.sendMessage(WSMessage{Type: "pong"})
		default:
			s.sendError("未知消息类型：" + msg.Type)
		}
	}
}

func (s *Session) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		err := s.conn.WriteControl(
			websocket.PingMessage,
			[]byte("heartbeat"),
			time.Now().Add(5*time.Second),
		)
		s.mu.Unlock()

		if err != nil {
			log.Printf("[WebSocket] 心跳发送失败：task_id=%s, err=%v", s.taskID, err)
			return
		}

		log.Printf("[WebSocket] 发送心跳：task_id=%s", s.taskID)
	}
}

func (s *Session) getDB() *sql.DB {
	cfg := s.svcCtx.Config.Database
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=10",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName,
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("[WebSocket] 数据库连接失败：%v", err)
		return nil
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(3 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Printf("[WebSocket] 数据库 Ping 失败：%v", err)
		db.Close()
		return nil
	}

	return db
}

func (s *Session) getAudioList(db *sql.DB) []AudioContentInfo {
	query := `
		SELECT id, title, audio_url, duration_sec, artist 
		FROM content
		WHERE audio_url IS NOT NULL AND audio_url != ''
		ORDER BY id
		LIMIT 50
	`
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("[WebSocket] 查询音频列表失败：%v", err)
		return nil
	}
	defer rows.Close()

	var list []AudioContentInfo
	for rows.Next() {
		var item AudioContentInfo
		if rows.Scan(&item.ID, &item.Title, &item.AudioURL, &item.Duration, &item.Artist) == nil {
			list = append(list, item)
		}
	}
	log.Printf("[WebSocket] 获取音频列表：%d 条记录", len(list))
	return list
}

func (s *Session) sendAudioList(list []AudioContentInfo) {
	msg := WSMessage{
		Type: "audio_list",
		Data: map[string]interface{}{
			"audio_list": list,
			"total":      len(list),
		},
		Message: fmt.Sprintf("获取到 %d 条音频记录", len(list)),
	}
	s.sendMessage(msg)
}

func (s *Session) handleSeparate(data interface{}, db *sql.DB) {
	if s.userID == 0 {
		log.Printf("[WebSocket] ⚠️ 用户未登录，拒绝分离请求：task_id=%s", s.taskID)
		s.sendMessage(WSMessage{
			Type:    "auth_error",
			Message: "用户未登录或者是登录过期，请重新登录后再试",
			Data: map[string]interface{}{
				"task_id":    s.taskID,
				"action":     "separate",
				"reason":     "authentication_required",
				"suggestion": "请先登录获取有效token，然后重新连接WebSocket",
			},
		})
		return
	}

	var req struct {
		ContentID int64 `json:"content_id"`
	}

	dataBytes, _ := json.Marshal(data)
	if err := json.Unmarshal(dataBytes, &req); err != nil {
		s.sendError("请求参数解析失败")
		return
	}

	if req.ContentID <= 0 {
		s.sendError("content_id 必须大于 0")
		return
	}

	log.Printf("[WebSocket] 开始分离：task_id=%s, content_id=%d", s.taskID, req.ContentID)

	if err := s.sendMessage(WSMessage{
		Type:    "status",
		Message: "任务已接收，正在查询音频信息",
	}); err != nil {
		log.Printf("[WebSocket] 发送状态消息失败：%v", err)
		return
	}

	if err := s.createTask(db, req.ContentID, ""); err != nil {
		log.Printf("[WebSocket] ⚠️ 创建初始任务记录失败: %v", err)
	}

	var audioURL string
	err := db.QueryRow(
		`SELECT audio_url FROM content WHERE id = $1`,
		req.ContentID,
	).Scan(&audioURL)

	if err != nil {
		log.Printf("[WebSocket] 查询音频失败：content_id=%d, err=%v", req.ContentID, err)
		s.updateStatus(db, "failed", 0, fmt.Sprintf("查询音频失败：%v", err))

		if err == sql.ErrNoRows {
			s.sendError(fmt.Sprintf("音频内容不存在：content_id=%d", req.ContentID))
		} else {
			s.sendError(fmt.Sprintf("查询音频失败：%v", err))
		}
		return
	}

	if audioURL == "" {
		log.Printf("[WebSocket] 音频 URL 为空：content_id=%d", req.ContentID)
		s.updateStatus(db, "failed", 0, "音频 URL 为空")
		s.sendError("音频 URL 为空")
		return
	}

	log.Printf("[WebSocket] 查询到音频：content_id=%d, url=%s", req.ContentID, audioURL)

	s.updateTaskWithAudioURL(db, audioURL)

	s.executeSeparation(req.ContentID, audioURL, db)
}

func (s *Session) createTask(db *sql.DB, contentID int64, audioURL string) error {
	query := `
		INSERT INTO audio_separation_tasks
		(task_id, user_id, content_id, audio_url, status, progress, message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'pending', 0, '任务已创建', NOW(), NOW())
		ON CONFLICT (task_id) DO UPDATE SET
			user_id = $2,
			content_id = $3,
			audio_url = $4,
			status = 'pending',
			progress = 0,
			message = '任务已创建',
			updated_at = NOW()
	`
	result, err := db.Exec(query, s.taskID, s.userID, contentID, audioURL)
	if err != nil {
		log.Printf("[WebSocket] ❌ 创建任务记录失败: task_id=%s, error=%v", s.taskID, err)
		return fmt.Errorf("创建任务记录失败: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("[WebSocket] ✅ 任务记录已创建/更新: task_id=%s, content_id=%d, rows_affected=%d", s.taskID, contentID, rowsAffected)
	return nil
}

func (s *Session) updateTaskWithAudioURL(db *sql.DB, audioURL string) {
	query := `
		UPDATE audio_separation_tasks
		SET audio_url = $1,
		    message = '音频信息已获取，准备分离',
		    updated_at = NOW()
		WHERE task_id = $2
	`
	result, err := db.Exec(query, audioURL, s.taskID)
	if err != nil {
		log.Printf("[WebSocket] ⚠️ 更新任务音频URL失败: task_id=%s, error=%v", s.taskID, err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("[WebSocket] ✅ 任务记录已更新音频URL: task_id=%s, url=%s, rows_affected=%d", s.taskID, audioURL, rowsAffected)
}

func (s *Session) cleanURL(rawURL string) string {
	cleaned := strings.TrimSpace(rawURL)
	cleaned = strings.ReplaceAll(cleaned, "\r", "")
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	cleaned = strings.ReplaceAll(cleaned, "\t", "")

	if cleaned != rawURL {
		log.Printf("[WebSocket] URL 已清理: 原始='%s' → 清理后='%s'", rawURL, cleaned)
	}

	return cleaned
}

func (s *Session) createTempDir() (string, error) {
	tempBaseDir := "./temp"
	if err := os.MkdirAll(tempBaseDir, 0755); err != nil {
		return "", fmt.Errorf("创建临时根目录失败: %w", err)
	}

	tempDir := filepath.Join(tempBaseDir, fmt.Sprintf("task_%s_%d", s.taskID, time.Now().UnixNano()))

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("创建任务临时目录失败: %w", err)
	}

	log.Printf("[WebSocket] 创建临时目录: %s", tempDir)
	return tempDir, nil
}

func (s *Session) cleanupTempDir(tempDir string) {
	if tempDir == "" || tempDir == "." || tempDir == "./" {
		log.Printf("[WebSocket] 跳过清理无效目录: %s", tempDir)
		return
	}

	log.Printf("[WebSocket] 开始清理临时目录: %s", tempDir)

	err := os.RemoveAll(tempDir)
	if err != nil {
		log.Printf("[WebSocket] ⚠️ 清理临时目录失败: %s, 错误: %v", tempDir, err)
	} else {
		log.Printf("[WebSocket] ✅ 临时目录已清理: %s", tempDir)
	}
}

func (s *Session) detectFileExtension(url string) string {
	urlLower := strings.ToLower(url)

	extMap := map[string]string{
		".mp3":  ".mp3",
		".wav":  ".wav",
		".flac": ".flac",
		".ogg":  ".ogg",
		".m4a":  ".m4a",
		".aac":  ".aac",
		".wma":  ".wma",
	}

	for ext, result := range extMap {
		if strings.Contains(urlLower, ext) {
			return result
		}
	}

	return ".mp3"
}

func (s *Session) downloadAudioFile(audioURL string, tempDir string) (string, error) {
	ext := s.detectFileExtension(audioURL)
	localPath := filepath.Join(tempDir, fmt.Sprintf("input%s", ext))

	log.Printf("[WebSocket] 📥 开始下载音频文件")
	log.Printf("[WebSocket]    源 URL: %s", audioURL)
	log.Printf("[WebSocket]    目标路径: %s", localPath)

	client := &http.Client{
		Timeout: 300 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
		},
	}

	resp, err := client.Get(audioURL)
	if err != nil {
		return "", fmt.Errorf("HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[WebSocket]    HTTP 响应状态: %d (%s)", resp.StatusCode, resp.Status)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("服务器返回错误 %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	contentLength := resp.ContentLength
	contentType := resp.Header.Get("Content-Type")

	log.Printf("[WebSocket]    文件大小: %.2f MB", float64(contentLength)/1024/1024)
	log.Printf("[WebSocket]    Content-Type: %s", contentType)

	out, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("创建本地文件失败: %w", err)
	}

	written, err := io.Copy(out, resp.Body)
	out.Close()

	if err != nil {
		os.Remove(localPath)
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	if written == 0 {
		os.Remove(localPath)
		return "", fmt.Errorf("下载的文件为空（0 bytes）")
	}

	fileInfo, _ := os.Stat(localPath)
	log.Printf("[WebSocket] ✅ 文件下载完成:")
	log.Printf("[WebSocket]    路径: %s", localPath)
	log.Printf("[WebSocket]    大小: %.2f KB", float64(written)/1024)
	log.Printf("[WebSocket]    权限: %v", fileInfo.Mode())

	return localPath, nil
}

func (s *Session) resolveAudioPath(audioURL string) (string, error) {
	audioURL = s.cleanURL(audioURL)

	if audioURL == "" {
		return "", fmt.Errorf("音频 URL 为空")
	}

	if strings.HasPrefix(audioURL, "/") {
		log.Printf("[WebSocket] 使用容器内绝对路径: %s", audioURL)
		if _, err := os.Stat(audioURL); os.IsNotExist(err) {
			return "", fmt.Errorf("容器内文件不存在: %s", audioURL)
		}
		return audioURL, nil
	}

	if strings.HasPrefix(audioURL, "./") || strings.HasPrefix(audioURL, "../") {
		if _, err := os.Stat(audioURL); err == nil {
			log.Printf("[WebSocket] 使用本地相对路径: %s", audioURL)
			return audioURL, nil
		}
		return "", fmt.Errorf("本地文件不存在: %s", audioURL)
	}

	if strings.HasPrefix(audioURL, "http://") || strings.HasPrefix(audioURL, "https://") {
		tempDir, err := s.createTempDir()
		if err != nil {
			return "", fmt.Errorf("创建临时目录失败: %w", err)
		}

		localPath, err := s.downloadAudioFile(audioURL, tempDir)
		if err != nil {
			s.cleanupTempDir(tempDir)
			return "", err
		}

		s.tempDir = tempDir
		return localPath, nil
	}

	return "", fmt.Errorf("不支持的音频路径格式: %s", audioURL)
}

func (s *Session) executeSeparation(contentID int64, audioURL string, db *sql.DB) {
	log.Printf("[WebSocket] 🎵 开始执行音频分离任务")
	log.Printf("[WebSocket]    task_id: %s", s.taskID)
	log.Printf("[WebSocket]    content_id: %d", contentID)

	defer func() {
		if s.tempDir != "" {
			log.Printf("[WebSocket] 🧹 任务结束，清理临时目录...")
			s.cleanupTempDir(s.tempDir)
			s.tempDir = ""
		}
	}()

	taskManager := model.GetGlobalTaskManager()
	_, cancel := context.WithCancel(context.Background())

	if err := taskManager.RegisterTask(s.taskID, s.userID, "processing", cancel); err != nil {
		log.Printf("[WebSocket] ⚠️ 注册任务到管理器失败：%v", err)
	}

	defer taskManager.UnregisterTask(s.taskID)

	s.updateStatus(db, "processing", 30, "准备分离...")

	cancelCh := taskManager.GetCancelChannel(s.taskID)

	s.checkCancellation(cancelCh)

	aiService := s.svcCtx.AIService
	if aiService == nil {
		log.Printf("[WebSocket] AI 服务未初始化：task_id=%s", s.taskID)
		s.updateStatus(db, "failed", 0, "AI 服务未初始化")
		s.sendError("AI 服务未初始化，请稍后重试或联系管理员")
		return
	}

	if !aiService.IsReady() {
		log.Printf("[WebSocket] AI 模型未就绪：task_id=%s", s.taskID)
		s.updateStatus(db, "failed", 0, "AI 模型未就绪")
		s.sendError("AI 模型正在加载中，请稍后重试")
		return
	}

	localPath, err := s.resolveAudioPath(audioURL)
	if err != nil {
		log.Printf("[WebSocket] ❌ 音频路径处理失败：task_id=%s", s.taskID)
		log.Printf("[WebSocket]    URL: %s", audioURL)
		log.Printf("[WebSocket]    错误: %v", err)
		s.updateStatus(db, "failed", 0, fmt.Sprintf("音频处理失败：%v", err))
		s.sendError(fmt.Sprintf("音频处理失败：%v", err))
		return
	}

	log.Printf("[WebSocket] ✅ 音频文件已准备")
	log.Printf("[WebSocket]    本地路径: %s", localPath)

	fileInfo, statErr := os.Stat(localPath)
	if statErr == nil {
		log.Printf("[WebSocket]    文件大小: %.2f KB", float64(fileInfo.Size())/1024)
	}

	s.updateStatus(db, "processing", 50, "调用 AI 模型分离...")

	s.checkCancellation(cancelCh)

	result, err := aiService.Separate(s.taskID, localPath)
	if err != nil {
		if taskManager.IsTaskCancelled(s.taskID) {
			log.Printf("[WebSocket] 🛑 任务已被用户取消：task_id=%s", s.taskID)
			s.sendMessage(WSMessage{
				Type:    "task_cancelled",
				Message: "任务已取消",
				Data: map[string]interface{}{
					"task_id":      s.taskID,
					"cancelled_at": time.Now().Format(time.RFC3339Nano),
					"reason":       "用户主动取消",
				},
			})
			return
		}

		log.Printf("[WebSocket] ❌ AI 分离失败：task_id=%s", s.taskID)
		log.Printf("[WebSocket]    错误详情: %v", err)
		s.updateStatus(db, "failed", 0, fmt.Sprintf("AI 分离失败：%v", err))

		errorMsg := "AI 分离失败"
		if strings.Contains(err.Error(), "model") {
			errorMsg = "模型加载失败，请检查模型配置"
		} else if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
			errorMsg = "处理超时，请稍后重试"
		} else if strings.Contains(err.Error(), "memory") || strings.Contains(err.Error(), "OOM") {
			errorMsg = "内存不足，请尝试更短的音频"
		}

		s.sendError(fmt.Sprintf("%s：%v", errorMsg, err))
		return
	}

	s.checkCancellation(cancelCh)

	log.Printf("[WebSocket] ✅ AI 分离完成")
	log.Printf("[WebSocket]    生成音轨数: %d", len(result.Tracks))

	s.updateStatus(db, "processing", 80, "保存结果...")

	tracks, err := s.saveTracks(db, result.Tracks)
	if err != nil {
		log.Printf("[WebSocket] ❌ 保存音轨失败：task_id=%s", s.taskID)
		log.Printf("[WebSocket]    错误: %v", err)
		s.updateStatus(db, "failed", 80, fmt.Sprintf("保存失败：%v", err))
		s.sendError(fmt.Sprintf("保存结果失败：%v", err))
		return
	}

	s.updateStatus(db, "completed", 100, "分离完成")

	if err := s.sendMessage(WSMessage{
		Type:    "separation_complete",
		Message: "分离成功！",
		Data: map[string]interface{}{
			"task_id":    s.taskID,
			"content_id": contentID,
			"tracks":     tracks,
		},
	}); err != nil {
		log.Printf("[WebSocket] ⚠️ 发送完成消息失败：%v", err)
	}

	log.Printf("[WebSocket] 🎉 任务完成！task_id=%s", s.taskID)
}

// checkCancellation 检查是否收到取消信号
func (s *Session) checkCancellation(cancelCh <-chan struct{}) {
	if cancelCh == nil {
		return
	}

	select {
	case <-cancelCh:
		log.Printf("[WebSocket] 🛑 检测到取消信号：task_id=%s", s.taskID)
		s.sendMessage(WSMessage{
			Type:    "cancellation_detected",
			Message: "检测到取消请求，正在停止...",
			Data: map[string]interface{}{
				"task_id": s.taskID,
			},
		})
	default:
	}
}

func (s *Session) saveTracks(db *sql.DB, tracks map[string]string) ([]map[string]interface{}, error) {
	var trackInfos []map[string]interface{}
	idx := 0

	for name, _ := range tracks {
		url := fmt.Sprintf("/static/tracks/%s_%s.wav", s.taskID, name)

		var id int64
		err := db.QueryRow(`
			INSERT INTO audio_tracks 
			(task_id, track_name, track_url, file_size, duration, sample_rate, channels, format, order_index, created_at)
			VALUES ($1, $2, $3, 0, 0, 44100, 2, 'wav', $4, NOW())
			RETURNING id
		`, s.taskID, name, url, idx).Scan(&id)

		if err != nil {
			return nil, err
		}

		trackInfos = append(trackInfos, map[string]interface{}{
			"id":         id,
			"track_name": name,
			"track_url":  url,
		})
		idx++
	}

	return trackInfos, nil
}

func (s *Session) updateStatus(db *sql.DB, status string, progress int, msg string) {
	db.Exec(`
		UPDATE audio_separation_tasks 
		SET status=$1, progress=$2, message=$3, updated_at=NOW()
		WHERE task_id=$4
	`, status, progress, msg, s.taskID)
}

func (s *Session) sendMessage(msg WSMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WebSocket] JSON 序列化失败：task_id=%s, err=%v", s.taskID, err)
		return err
	}

	err = s.conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		log.Printf("[WebSocket] 发送消息失败：task_id=%s, err=%v", s.taskID, err)
		return err
	}

	log.Printf("[WebSocket] 消息发送成功：task_id=%s, type=%s", s.taskID, msg.Type)
	return nil
}

func (s *Session) sendError(errMsg string) {
	log.Printf("[WebSocket] 错误：task_id=%s, msg=%s", s.taskID, errMsg)
	s.sendMessage(WSMessage{
		Type:  "error",
		Error: errMsg,
	})
}

// handleCancel 处理取消任务请求
func (s *Session) handleCancel(db *sql.DB) {
	log.Printf("[WebSocket] 🛑 收到取消请求：task_id=%s, user_id=%d", s.taskID, s.userID)

	if s.taskID == "" || s.userID == 0 {
		log.Printf("[WebSocket] ⚠️ 用户未登录或任务信息不完整，拒绝取消请求")
		s.sendMessage(WSMessage{
			Type:    "auth_error",
			Message: "用户未登录或者是登录过期，无法执行此操作",
			Data: map[string]interface{}{
				"task_id":    s.taskID,
				"action":     "cancel",
				"reason":     "authentication_required",
				"suggestion": "请重新登录后再试",
			},
		})
		return
	}

	taskManager := model.GetGlobalTaskManager()

	taskInfo, exists := taskManager.GetTaskInfo(s.taskID)
	if !exists {
		s.sendMessage(WSMessage{
			Type:    "cancel_failed",
			Message: "任务不存在或已完成",
			Data: map[string]interface{}{
				"task_id": s.taskID,
				"reason":  "任务不在活跃列表中",
			},
		})
		return
	}

	if taskInfo.UserID != s.userID {
		s.sendError("无权取消此任务")
		return
	}

	currentStatus := taskInfo.GetStatus()
	wasRunning := (currentStatus == model.TaskStatusProcessing)

	if currentStatus != model.TaskStatusPending && currentStatus != model.TaskStatusProcessing {
		statusDesc := map[string]string{
			model.TaskStatusCompleted: "已完成",
			model.TaskStatusFailed:    "已失败",
			model.TaskStatusCancelled: "已取消",
		}
		desc, _ := statusDesc[currentStatus]
		if desc == "" {
			desc = currentStatus
		}

		s.sendMessage(WSMessage{
			Type:    "cancel_failed",
			Message: fmt.Sprintf("无法取消%s的任务", desc),
			Data: map[string]interface{}{
				"task_id":     s.taskID,
				"status":      currentStatus,
				"status_desc": desc,
			},
		})
		return
	}

	log.Printf("[WebSocket] 正在取消任务：task_id=%s, 当前状态=%s", s.taskID, currentStatus)

	s.sendMessage(WSMessage{
		Type:    "cancel_progress",
		Message: "正在处理取消请求...",
		Data: map[string]interface{}{
			"task_id":     s.taskID,
			"was_running": wasRunning,
		},
	})

	cancelled, message, err := taskManager.CancelTask(s.taskID, s.userID)
	if err != nil {
		log.Printf("[WebSocket] ❌ 取消失败：task_id=%s, error=%v", s.taskID, err)
		s.sendError(fmt.Sprintf("取消失败：%v", err))
		return
	}

	if !cancelled {
		s.sendMessage(WSMessage{
			Type:    "cancel_failed",
			Message: message,
			Data: map[string]interface{}{
				"task_id": s.taskID,
			},
		})
		return
	}

	s.updateStatus(db, model.TaskStatusCancelled, -1, "用户取消任务")

	log.Printf("[WebSocket] ✅ 任务取消成功：task_id=%s", s.taskID)

	cancelResp := WSMessage{
		Type:    "task_cancelled",
		Message: "任务已成功取消",
		Data: map[string]interface{}{
			"task_id":      s.taskID,
			"cancelled_at": time.Now().Format(time.RFC3339Nano),
			"was_running":  wasRunning,
			"message":      message,
			"cleanup_hint": "临时文件将在后台自动清理",
		},
	}

	if err := s.sendMessage(cancelResp); err != nil {
		log.Printf("[WebSocket] ⚠️ 发送取消成功消息失败：%v", err)
	}

	if s.tempDir != "" {
		go func() {
			time.Sleep(2 * time.Second)
			s.cleanupTempDir(s.tempDir)
			s.tempDir = ""
			log.Printf("[WebSocket] 🧹 取消后清理完成：task_id=%s", s.taskID)
		}()
	}

	log.Printf("[WebSocket] 🎉 取消流程完成：task_id=%s", s.taskID)
}
