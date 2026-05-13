package logic

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
)

// TaskDetailReq 任务详情请求
type TaskDetailReq struct {
	TaskID string `json:"task_id"` // 任务 ID（必填）
}

// TaskDetailResp 任务详情响应
type TaskDetailResp struct {
	TaskID    string        `json:"task_id"`
	UserID    int64         `json:"user_id"`
	ContentID int64         `json:"content_id"`
	AudioURL  string        `json:"audio_url"`
	Status    string        `json:"status"`
	Progress  int           `json:"progress"`
	Message   string        `json:"message,omitempty"`
	ResultURL string        `json:"result_url,omitempty"`
	Tracks    []TrackDetail `json:"tracks,omitempty"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
}

// TrackDetail 音轨详情
type TrackDetail struct {
	TrackName string  `json:"track_name"`
	FilePath  string  `json:"file_path"`
	FileSize  int64   `json:"file_size"`
	Duration  float64 `json:"duration"`
	URL       string  `json:"url,omitempty"`
}

// TaskDetailLogic 任务详情逻辑
type TaskDetailLogic struct {
	svcCtx *svc.ServiceContext
}

// NewTaskDetailLogic 创建任务详情逻辑实例
func NewTaskDetailLogic(svcCtx *svc.ServiceContext) *TaskDetailLogic {
	return &TaskDetailLogic{
		svcCtx: svcCtx,
	}
}

// GetTaskDetail 获取任务详情（带权限验证）
func (l *TaskDetailLogic) GetTaskDetail(req *TaskDetailReq, userID int64) (*TaskDetailResp, error) {
	if req.TaskID == "" {
		return nil, fmt.Errorf("任务 ID 不能为空")
	}

	task, err := l.getTaskFromDB(req.TaskID, userID)
	if err != nil {
		return nil, err
	}

	if task == nil {
		return nil, fmt.Errorf("任务不存在或无权访问")
	}

	tracks, err := l.getTracksFromDB(req.TaskID)
	if err != nil {
		logx.Errorf("获取音轨信息失败: %v", err)
	}

	resp := &TaskDetailResp{
		TaskID:    task.TaskID,
		UserID:    task.UserID,
		ContentID: task.ContentID,
		AudioURL:  task.AudioURL,
		Status:    task.Status,
		Progress:  task.Progress,
		Tracks:    tracks,
		CreatedAt: task.CreatedAt.Format(time.RFC3339),
		UpdatedAt: task.UpdatedAt.Format(time.RFC3339),
	}

	if task.Message.Valid {
		resp.Message = task.Message.String
	}
	if task.ResultURL.Valid {
		resp.ResultURL = task.ResultURL.String
	}

	return resp, nil
}

// taskDBModel 数据库中的任务模型
type taskDBModel struct {
	TaskID    string
	UserID    int64
	ContentID int64
	AudioURL  string
	Status    string
	Progress  int
	Message   sql.NullString
	ResultURL sql.NullString
	CreatedAt time.Time
	UpdatedAt time.Time
}

// getTaskFromDB 从数据库获取任务（带用户权限验证）
func (l *TaskDetailLogic) getTaskFromDB(taskID string, userID int64) (*taskDBModel, error) {
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
		return nil, fmt.Errorf("连接数据库失败：%w", err)
	}
	defer db.Close()

	query := `
		SELECT 
			task_id, user_id, content_id, audio_url, status, progress, message, result_url,
			created_at, updated_at
		FROM audio_separation_tasks
		WHERE task_id = $1 AND user_id = $2
	`

	var task taskDBModel
	err = db.QueryRow(query, taskID, userID).Scan(
		&task.TaskID,
		&task.UserID,
		&task.ContentID,
		&task.AudioURL,
		&task.Status,
		&task.Progress,
		&task.Message,
		&task.ResultURL,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询任务失败：%w", err)
	}

	return &task, nil
}

// getTracksFromDB 获取任务的音轨列表
func (l *TaskDetailLogic) getTracksFromDB(taskID string) ([]TrackDetail, error) {
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
		return nil, fmt.Errorf("连接数据库失败：%w", err)
	}
	defer db.Close()

	query := `
		SELECT 
			track_name, track_url, file_size, duration
		FROM audio_tracks
		WHERE task_id = $1
		ORDER BY order_index, track_name
	`

	rows, err := db.Query(query, taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []TrackDetail{}, nil
		}
		return nil, fmt.Errorf("查询音轨失败：%w", err)
	}
	defer rows.Close()

	var tracks []TrackDetail
	for rows.Next() {
		var track TrackDetail

		err := rows.Scan(
			&track.TrackName,
			&track.URL,
			&track.FileSize,
			&track.Duration,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描音轨失败：%w", err)
		}

		tracks = append(tracks, track)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历音轨失败：%w", err)
	}

	return tracks, nil
}
