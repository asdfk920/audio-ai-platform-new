package logic

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jacklau/audio-ai-platform/services/ai-worker/internal/svc"
)

// TaskDeleteReq 任务删除请求
type TaskDeleteReq struct {
	TaskID string `json:"task_id"` // 任务标识符 (UUID格式，如: task_5_3_xxx)
	ID     int64  `json:"id"`      // 主键ID (数字，如: 2)
}

// TaskDeleteResp 任务删除响应
type TaskDeleteResp struct {
	TaskID      string `json:"task_id"`
	Deleted     bool   `json:"deleted"`
	Message     string `json:"message"`
	DeletedAt   string `json:"deleted_at"`
	TracksCount int    `json:"tracks_count"` // 删除的音轨数量
}

// TaskDeleteLogic 任务删除逻辑
type TaskDeleteLogic struct {
	svcCtx *svc.ServiceContext
}

// NewTaskDeleteLogic 创建任务删除逻辑实例
func NewTaskDeleteLogic(svcCtx *svc.ServiceContext) *TaskDeleteLogic {
	return &TaskDeleteLogic{
		svcCtx: svcCtx,
	}
}

// DeleteTask 删除任务（带权限验证和级联清理）
// 支持两种方式：
//  1. 通过 task_id 删除 (UUID格式字符串)
//  2. 通过 id 删除 (主键数字)
func (l *TaskDeleteLogic) DeleteTask(req *TaskDeleteReq, userID int64) (*TaskDeleteResp, error) {
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

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开启事务失败：%w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	var taskExists bool
	var actualTaskID string
	var taskStatus string

	if req.TaskID != "" {
		err = tx.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM audio_separation_tasks WHERE task_id = $1 AND user_id = $2),
			        COALESCE((SELECT task_id FROM audio_separation_tasks WHERE task_id = $1 AND user_id = $2), ''),
			        COALESCE((SELECT status FROM audio_separation_tasks WHERE task_id = $1 AND user_id = $2), '')`,
			req.TaskID, userID,
		).Scan(&taskExists, &actualTaskID, &taskStatus)
	} else {
		err = tx.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM audio_separation_tasks WHERE id = $1 AND user_id = $2),
			        COALESCE((SELECT task_id FROM audio_separation_tasks WHERE id = $1 AND user_id = $2), ''),
			        COALESCE((SELECT status FROM audio_separation_tasks WHERE id = $1 AND user_id = $2), '')`,
			req.ID, userID,
		).Scan(&taskExists, &actualTaskID, &taskStatus)
	}

	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("查询任务失败：%w", err)
	}

	if !taskExists || actualTaskID == "" {
		tx.Rollback()
		return nil, fmt.Errorf("任务不存在或无权访问")
	}

	tracksResult, err := tx.Exec(`DELETE FROM audio_tracks WHERE task_id = $1`, actualTaskID)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("删除关联音轨失败：%w", err)
	}

	tracksAffected, _ := tracksResult.RowsAffected()

	taskResult, err := tx.Exec(
		`DELETE FROM audio_separation_tasks WHERE task_id = $1 AND user_id = $2`,
		actualTaskID, userID,
	)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("删除任务失败：%w", err)
	}

	taskAffected, _ := taskResult.RowsAffected()
	if taskAffected == 0 {
		tx.Rollback()
		return nil, fmt.Errorf("任务不存在或已被删除")
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败：%w", err)
	}

	logx.Infof("用户 %d 成功删除任务: %s (状态: %s, 音轨数: %d)",
		userID, actualTaskID, taskStatus, tracksAffected)

	resp := &TaskDeleteResp{
		TaskID:      actualTaskID,
		Deleted:     true,
		Message:     "任务删除成功",
		DeletedAt:   time.Now().Format(time.RFC3339),
		TracksCount: int(tracksAffected),
	}

	return resp, nil
}
