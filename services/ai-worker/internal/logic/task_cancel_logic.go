package logic

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/model"
	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
)

// TaskCancelReq 任务取消请求
type TaskCancelReq struct {
	TaskID string `json:"task_id"` // 任务标识符 (UUID格式)
	ID     int64  `json:"id"`      // 主键ID (数字)
}

// TaskCancelResp 任务取消响应
type TaskCancelResp struct {
	TaskID        string `json:"task_id"`
	Cancelled     bool   `json:"cancelled"`
	Message       string `json:"message"`
	CancelledAt   string `json:"cancelled_at"`
	WasRunning    bool   `json:"was_running"`    // 取消时任务是否正在运行
	Status        string `json:"status"`         // 取消后的状态
	TracksCleaned int    `json:"tracks_cleaned"` // 清理的音轨数量
}

// TaskCancelLogic 任务取消逻辑
type TaskCancelLogic struct {
	svcCtx *svc.ServiceContext
}

// NewTaskCancelLogic 创建任务取消逻辑实例
func NewTaskCancelLogic(svcCtx *svc.ServiceContext) *TaskCancelLogic {
	return &TaskCancelLogic{
		svcCtx: svcCtx,
	}
}

// CancelTask 取消分离任务（带权限验证和状态检查）
func (l *TaskCancelLogic) CancelTask(req *TaskCancelReq, userID int64) (*TaskCancelResp, error) {
	if req.TaskID == "" && req.ID == 0 {
		return nil, fmt.Errorf("任务 ID 或主键 ID 不能为空")
	}

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

	var actualTaskID string
	var currentStatus string

	if req.TaskID != "" {
		actualTaskID = req.TaskID
	} else {
		err := db.QueryRow(
			`SELECT task_id FROM audio_separation_tasks WHERE id = $1 AND user_id = $2`,
			req.ID, userID,
		).Scan(&actualTaskID)

		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("任务不存在或无权访问")
			}
			return nil, fmt.Errorf("查询任务失败：%w", err)
		}
	}

	var taskExists bool
	err = db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM audio_separation_tasks WHERE task_id = $1 AND user_id = $2),
		        COALESCE((SELECT status FROM audio_separation_tasks WHERE task_id = $1 AND user_id = $2), '')`,
		actualTaskID, userID,
	).Scan(&taskExists, &currentStatus)

	if err != nil {
		return nil, fmt.Errorf("查询任务失败：%w", err)
	}

	if !taskExists {
		return nil, fmt.Errorf("任务不存在或无权访问")
	}

	if currentStatus == model.TaskStatusCompleted {
		return nil, fmt.Errorf("无法取消已完成的任务")
	}

	if currentStatus == model.TaskStatusFailed {
		return nil, fmt.Errorf("无法取消已失败的任务（建议使用删除功能）")
	}

	if currentStatus == model.TaskStatusCancelled {
		resp := &TaskCancelResp{
			TaskID:      actualTaskID,
			Cancelled:   true,
			Message:     "任务已经被取消",
			CancelledAt: time.Now().Format(time.RFC3339),
			WasRunning:  false,
			Status:      model.TaskStatusCancelled,
		}
		return resp, nil
	}

	wasRunning := (currentStatus == model.TaskStatusProcessing)

	taskManager := model.GetGlobalTaskManager()
	cancelled, msg, err := taskManager.CancelTask(actualTaskID, userID)
	if err != nil {
		return nil, fmt.Errorf("取消任务失败：%w", err)
	}

	if !cancelled {
		return &TaskCancelResp{
			TaskID:     actualTaskID,
			Cancelled:  false,
			Message:    msg,
			Status:     currentStatus,
			WasRunning: wasRunning,
		}, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开启事务失败：%w", err)
	}

	now := time.Now()
	result, err := tx.Exec(`
		UPDATE audio_separation_tasks 
		SET status = $1, progress = -1, message = '用户取消任务', updated_at = $2 
		WHERE task_id = $3 AND user_id = $4 AND status IN ('pending', 'processing')
	`, model.TaskStatusCancelled, now, actualTaskID, userID)

	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新任务状态失败：%w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	tracksResult, _ := tx.Exec(`DELETE FROM audio_tracks WHERE task_id = $1`, actualTaskID)
	tracksAffected, _ := tracksResult.RowsAffected()

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败：%w", err)
	}

	if rowsAffected == 0 && currentStatus != model.TaskStatusCancelled {
		return &TaskCancelResp{
			TaskID:     actualTaskID,
			Cancelled:  false,
			Message:    "任务可能已经完成或已被其他操作修改",
			Status:     currentStatus,
			WasRunning: wasRunning,
		}, nil
	}

	resp := &TaskCancelResp{
		TaskID:        actualTaskID,
		Cancelled:     true,
		Message:       msg,
		CancelledAt:   now.Format(time.RFC3339),
		WasRunning:    wasRunning,
		Status:        model.TaskStatusCancelled,
		TracksCleaned: int(tracksAffected),
	}

	return resp, nil
}
