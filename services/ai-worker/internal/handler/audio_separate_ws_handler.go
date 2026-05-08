package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jacklau/audio-ai-platform/pkg/jwtx"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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
				log.Printf("[WebSocket] 认证失败：token无效或过期")
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

		session.handleConnection()
	}
}

type Session struct {
	conn   *websocket.Conn
	taskID string
	userID int64
	svcCtx *svc.ServiceContext
	mu     sync.Mutex
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
	db := s.getDB()
	if db == nil {
		s.sendError("数据库连接失败")
		return
	}
	defer db.Close()

	audioList := s.getAudioList(db)
	s.sendAudioList(audioList)

	for {
		messageType, message, err := s.conn.ReadMessage()
		if err != nil {
			log.Printf("[WebSocket] 连接关闭：task_id=%s", s.taskID)
			break
		}

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
		default:
			s.sendError("未知消息类型：" + msg.Type)
		}
	}
}

func (s *Session) getDB() *sql.DB {
	cfg := s.svcCtx.Config.Database
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName,
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("[WebSocket] 数据库连接失败：%v", err)
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
	var req struct {
		ContentID int64 `json:"content_id"`
	}

	dataBytes, _ := json.Marshal(data)
	json.Unmarshal(dataBytes, &req)

	if req.ContentID <= 0 {
		s.sendError("content_id 必须大于 0")
		return
	}

	log.Printf("[WebSocket] 开始分离：task_id=%s, content_id=%d", s.taskID, req.ContentID)

	s.sendMessage(WSMessage{
		Type:    "status",
		Message: "任务已接收，正在查询音频信息",
	})

	var audioURL string
	err := db.QueryRow(
		`SELECT audio_url FROM content WHERE id = $1`,
		req.ContentID,
	).Scan(&audioURL)

	if err != nil {
		log.Printf("[WebSocket] 查询音频失败：content_id=%d, err=%v", req.ContentID, err)
		s.sendError(fmt.Sprintf("查询音频失败：%v", err))
		return
	}

	log.Printf("[WebSocket] 查询到音频：content_id=%d, url=%s", req.ContentID, audioURL)

	s.createTask(db, req.ContentID, audioURL)
	s.executeSeparation(req.ContentID, audioURL, db)
}

func (s *Session) createTask(db *sql.DB, contentID int64, audioURL string) {
	query := `
		INSERT INTO audio_separation_tasks 
		(task_id, user_id, content_id, audio_url, status, progress, message, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'pending', 0, '任务已创建', NOW(), NOW())
		ON CONFLICT (task_id) DO NOTHING
	`
	db.Exec(query, s.taskID, s.userID, contentID, audioURL)
}

func (s *Session) resolveAudioPath(audioURL string) (string, error) {
	if strings.HasPrefix(audioURL, "/") {
		log.Printf("[WebSocket] 使用容器内绝对路径: %s", audioURL)
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
		log.Printf("[WebSocket] 下载远程文件: %s", audioURL)
		localPath := fmt.Sprintf("./output/audio_%s_%d.tmp", s.taskID, time.Now().Unix())

		resp, err := http.Get(audioURL)
		if err != nil {
			return "", fmt.Errorf("下载失败: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("HTTP 错误: %d", resp.StatusCode)
		}

		out, err := os.Create(localPath)
		if err != nil {
			return "", fmt.Errorf("创建文件失败: %w", err)
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return "", fmt.Errorf("写入文件失败: %w", err)
		}

		log.Printf("[WebSocket] 文件下载完成: %s", localPath)
		return localPath, nil
	}

	return "", fmt.Errorf("不支持的音频路径格式: %s", audioURL)
}

func (s *Session) executeSeparation(contentID int64, audioURL string, db *sql.DB) {
	s.updateStatus(db, "processing", 30, "准备分离...")

	aiService := s.svcCtx.AIService
	if aiService == nil {
		s.updateStatus(db, "failed", 0, "AI 服务未初始化")
		s.sendError("AI 服务未初始化")
		return
	}

	localPath, err := s.resolveAudioPath(audioURL)
	if err != nil {
		log.Printf("[WebSocket] 音频路径处理失败：task_id=%s, url=%s, err=%v", s.taskID, audioURL, err)
		s.updateStatus(db, "failed", 0, fmt.Sprintf("音频处理失败：%v", err))
		s.sendError(fmt.Sprintf("音频处理失败：%v", err))
		return
	}

	log.Printf("[WebSocket] 使用音频文件：task_id=%s, path=%s", s.taskID, localPath)

	s.updateStatus(db, "processing", 50, "调用 AI 模型分离...")

	result, err := aiService.Separate(s.taskID, localPath)
	if err != nil {
		log.Printf("[WebSocket] AI 分离失败：task_id=%s, err=%v", s.taskID, err)
		s.updateStatus(db, "failed", 0, fmt.Sprintf("AI 分离失败：%v", err))
		s.sendError(fmt.Sprintf("AI 分离失败：%v", err))
		return
	}

	log.Printf("[WebSocket] 分离完成：task_id=%s, tracks=%d", s.taskID, len(result.Tracks))

	s.updateStatus(db, "processing", 80, "保存结果...")

	tracks, err := s.saveTracks(db, result.Tracks)
	if err != nil {
		s.updateStatus(db, "failed", 80, fmt.Sprintf("保存失败：%v", err))
		s.sendError(fmt.Sprintf("保存失败：%v", err))
		return
	}

	s.updateStatus(db, "completed", 100, "分离完成")

	s.sendMessage(WSMessage{
		Type:    "separation_complete",
		Message: "分离成功！",
		Data: map[string]interface{}{
			"task_id":    s.taskID,
			"content_id": contentID,
			"tracks":     tracks,
		},
	})
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

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.WriteMessage(websocket.TextMessage, data)
}

func (s *Session) sendError(errMsg string) {
	log.Printf("[WebSocket] 错误：task_id=%s, msg=%s", s.taskID, errMsg)
	s.sendMessage(WSMessage{
		Type:  "error",
		Error: errMsg,
	})
}
