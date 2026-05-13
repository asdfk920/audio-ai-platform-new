package logic

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
)

// TaskListReq 任务列表请求
type TaskListReq struct {
	Page     int    `json:"page"`      // 页码，默认 1
	PageSize int    `json:"page_size"` // 每页数量，默认 20
	Status   string `json:"status"`    // 状态筛选（可选）
}

// TaskListResp 任务列表响应
type TaskListResp struct {
	Tasks      []TaskInfo `json:"tasks"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

// TaskInfo 任务信息
type TaskInfo struct {
	TaskID    string `json:"task_id"`
	ContentID int64  `json:"content_id"`
	AudioURL  string `json:"audio_url"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	Message   string `json:"message"`
	ResultURL string `json:"result_url,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// TaskListLogic 任务列表逻辑
type TaskListLogic struct {
	svcCtx *svc.ServiceContext
}

// NewTaskListLogic 创建任务列表逻辑实例
func NewTaskListLogic(svcCtx *svc.ServiceContext) *TaskListLogic {
	return &TaskListLogic{
		svcCtx: svcCtx,
	}
}

// GetTaskList 获取任务列表
func (l *TaskListLogic) GetTaskList(req *TaskListReq, userID int64) (*TaskListResp, error) {
	// 1. 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100 // 限制最大每页数量
	}

	// 2. 从数据库查询任务列表（只查询该用户的任务）
	tasks, total, err := l.getTaskListFromDB(req, userID)
	if err != nil {
		return nil, fmt.Errorf("查询任务列表失败：%w", err)
	}

	// 3. 计算总页数
	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	resp := &TaskListResp{
		Tasks:      tasks,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}

	return resp, nil
}

// getTaskListFromDB 从数据库获取任务列表
func (l *TaskListLogic) getTaskListFromDB(req *TaskListReq, userID int64) ([]TaskInfo, int64, error) {
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
		return nil, 0, fmt.Errorf("连接数据库失败：%w", err)
	}
	defer db.Close()

	// 计算偏移量
	offset := (req.Page - 1) * req.PageSize

	// 查询总数
	var total int64
	var countQuery string
	var countArgs []interface{}

	if req.Status != "" {
		countQuery = `SELECT COUNT(*) FROM audio_separation_tasks WHERE user_id = $1 AND status = $2`
		countArgs = []interface{}{userID, req.Status}
	} else {
		countQuery = `SELECT COUNT(*) FROM audio_separation_tasks WHERE user_id = $1`
		countArgs = []interface{}{userID}
	}

	err = db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("查询总数失败：%w", err)
	}

	// 查询任务列表
	var query string
	var args []interface{}

	if req.Status != "" {
		query = `
			SELECT 
				task_id, content_id, audio_url, status, progress, message, result_url,
				created_at, updated_at
			FROM audio_separation_tasks
			WHERE user_id = $1 AND status = $2
			ORDER BY created_at DESC
			LIMIT $3 OFFSET $4
		`
		args = []interface{}{userID, req.Status, req.PageSize, offset}
	} else {
		query = `
			SELECT 
				task_id, content_id, audio_url, status, progress, message, result_url,
				created_at, updated_at
			FROM audio_separation_tasks
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{userID, req.PageSize, offset}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询任务列表失败：%w", err)
	}
	defer rows.Close()

	var tasks []TaskInfo
	for rows.Next() {
		var task TaskInfo
		var createdAt, updatedAt time.Time
		var resultURL sql.NullString

		err := rows.Scan(
			&task.TaskID,
			&task.ContentID,
			&task.AudioURL,
			&task.Status,
			&task.Progress,
			&task.Message,
			&resultURL,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描任务失败：%w", err)
		}

		if resultURL.Valid {
			task.ResultURL = resultURL.String
		}
		task.CreatedAt = createdAt.Format(time.RFC3339)
		task.UpdatedAt = updatedAt.Format(time.RFC3339)

		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历任务失败：%w", err)
	}

	return tasks, total, nil
}
