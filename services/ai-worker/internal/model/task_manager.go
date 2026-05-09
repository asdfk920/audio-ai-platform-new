package model

import (
	"context"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// TaskStatus 任务状态常量
const (
	TaskStatusPending    = "pending"
	TaskStatusProcessing = "processing"
	TaskStatusCompleted  = "completed"
	TaskStatusFailed     = "failed"
	TaskStatusCancelled  = "cancelled"
)

// ActiveTaskInfo 活跃任务信息
type ActiveTaskInfo struct {
	TaskID      string                `json:"task_id"`
	UserID      int64                 `json:"user_id"`
	Status      string                `json:"status"`
	StartTime   time.Time             `json:"start_time"`
	CancelFunc  context.CancelFunc    `json:"-"`
	Progress    float64               `json:"progress"`
	Connections map[*interface{}]bool `json:"-"` // 关联的 WebSocket 连接
	mu          sync.RWMutex          `json:"-"`
}

// GetStatus 获取任务状态（线程安全）
func (t *ActiveTaskInfo) GetStatus() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status
}

// TaskManager 全局任务管理器（单例）
type TaskManager struct {
	tasks       map[string]*ActiveTaskInfo
	mu          sync.RWMutex
	logger      logx.Logger
	cancelChans map[string]chan struct{} // 用于通知取消的 channel
}

var (
	globalTaskManager *TaskManager
	taskManagerOnce   sync.Once
)

// GetGlobalTaskManager 获取全局任务管理器实例（单例模式）
func GetGlobalTaskManager() *TaskManager {
	taskManagerOnce.Do(func() {
		globalTaskManager = &TaskManager{
			tasks:       make(map[string]*ActiveTaskInfo),
			cancelChans: make(map[string]chan struct{}),
			logger:      logx.WithContext(context.Background()),
		}
	})
	return globalTaskManager
}

// RegisterTask 注册新任务到管理器
func (tm *TaskManager) RegisterTask(taskID string, userID int64, status string, cancelFunc context.CancelFunc) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tasks[taskID]; exists {
		return nil // 已存在则不重复注册
	}

	_, cancel := context.WithCancel(context.Background())
	if cancelFunc != nil {
		cancel = cancelFunc
	}

	tm.tasks[taskID] = &ActiveTaskInfo{
		TaskID:      taskID,
		UserID:      userID,
		Status:      status,
		StartTime:   time.Now(),
		CancelFunc:  cancel,
		Progress:    0,
		Connections: make(map[*interface{}]bool),
	}
	tm.cancelChans[taskID] = make(chan struct{}, 1)

	tm.logger.Infof("[TaskManager] 注册任务: task_id=%s, user_id=%d, status=%s", taskID, userID, status)
	return nil
}

// UpdateTaskStatus 更新任务状态
func (tm *TaskManager) UpdateTaskStatus(taskID string, status string, progress float64) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return nil
	}

	task.mu.Lock()
	task.Status = status
	task.Progress = progress
	task.mu.Unlock()

	tm.logger.Infof("[TaskManager] 更新任务状态: task_id=%s, status=%s, progress=%.1f%%", taskID, status, progress)
	return nil
}

// CancelTask 取消任务
func (tm *TaskManager) CancelTask(taskID string, userID int64) (bool, string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return false, "任务不存在或已完成", nil
	}

	// 验证用户权限
	if task.UserID != userID {
		return false, "无权取消此任务", nil
	}

	task.mu.RLock()
	currentStatus := task.Status
	task.mu.RUnlock()

	// 只能取消 pending 或 processing 状态的任务
	if currentStatus != TaskStatusPending && currentStatus != TaskStatusProcessing {
		return false, "只能取消待处理或处理中的任务（当前状态：" + currentStatus + "）", nil
	}

	// 执行取消操作
	task.mu.Lock()
	task.Status = TaskStatusCancelled
	task.Progress = -1 // 特殊值表示已取消
	task.mu.Unlock()

	// 调用 cancel 函数
	if task.CancelFunc != nil {
		task.CancelFunc()
	}

	// 通过 channel 发送取消信号
	if ch, ok := tm.cancelChans[taskID]; ok {
		select {
		case ch <- struct{}{}:
		default:
		}
	}

	tm.logger.Infof("[TaskManager] 任务已取消: task_id=%s, user_id=%d", taskID, userID)

	// 延迟清理任务记录（给前端一些时间接收取消通知）
	go func() {
		time.Sleep(30 * time.Second)
		tm.UnregisterTask(taskID)
	}()

	return true, "任务取消成功", nil
}

// IsTaskCancelled 检查任务是否已被取消
func (tm *TaskManager) IsTaskCancelled(taskID string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	ch, exists := tm.cancelChans[taskID]
	if !exists {
		return false
	}

	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// GetCancelChannel 获取任务的取消 channel（用于在长时间运行的操作中监听取消信号）
func (tm *TaskManager) GetCancelChannel(taskID string) <-chan struct{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if ch, exists := tm.cancelChans[taskID]; exists {
		return ch
	}
	return nil
}

// UnregisterTask 从管理器中移除任务
func (tm *TaskManager) UnregisterTask(taskID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	delete(tm.tasks, taskID)
	delete(tm.cancelChans, taskID)
	tm.logger.Infof("[TaskManager] 移除任务: task_id=%s", taskID)
}

// GetTaskInfo 获取任务信息
func (tm *TaskManager) GetTaskInfo(taskID string) (*ActiveTaskInfo, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return nil, false
	}

	return task, true
}

// GetUserActiveTasks 获取用户的所有活跃任务
func (tm *TaskManager) GetUserActiveTasks(userID int64) []*ActiveTaskInfo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var tasks []*ActiveTaskInfo
	for _, task := range tm.tasks {
		if task.UserID == userID {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// GetAllActiveTasks 获取所有活跃任务
func (tm *TaskManager) GetAllActiveTasks() map[string]*ActiveTaskInfo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make(map[string]*ActiveTaskInfo, len(tm.tasks))
	for k, v := range tm.tasks {
		result[k] = v
	}
	return result
}

// GetActiveTaskCount 获取活跃任务数量
func (tm *TaskManager) GetActiveTaskCount() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.tasks)
}
