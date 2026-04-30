package logic

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
)

// AccountCancellationTask 账号注销定时任务处理
type AccountCancellationTask struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewAccountCancellationTask 创建账号注销定时任务实例
func NewAccountCancellationTask(ctx context.Context, svcCtx *svc.ServiceContext) *AccountCancellationTask {
	return &AccountCancellationTask{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CancellationTaskResult 单个用户注销任务执行结果
type CancellationTaskResult struct {
	UserID    int64  `json:"user_id"`    // 用户 ID
	Success   bool   `json:"success"`    // 是否成功
	Message   string `json:"message"`    // 结果说明
	LogID     int64  `json:"log_id"`     // 注销记录 ID
	Executed  bool   `json:"executed"`   // 是否执行了注销
	Skipped   bool   `json:"skipped"`    // 是否跳过（冷静期未到）
	Failed    bool   `json:"failed"`     // 是否失败
	StartTime int64  `json:"start_time"` // 开始时间戳
	EndTime   int64  `json:"end_time"`   // 结束时间戳
}

// ExecuteCancellationTask 执行账号注销定时任务
// 扫描所有冷静期结束的注销申请，自动执行注销流程
//
// 参数说明：
//   - batchSize: 每批次处理的用户数量（避免一次性加载过多数据）
//
// 返回说明：
//   - total: 扫描的总用户数
//   - executed: 成功执行注销的用户数
//   - skipped: 跳过的用户数（冷静期未到或其他原因）
//   - failed: 失败的用户数
//   - err: 错误信息
func (l *AccountCancellationTask) ExecuteCancellationTask(batchSize int) (total, executed, skipped, failed int64, err error) {
	startTime := time.Now()
	l.Logger.Infof("开始执行账号注销定时任务，batch_size=%d", batchSize)

	// 1. 查询待处理的注销申请（冷静期已结束）
	// 条件：status=1 (冷静期), cooling_end_at <= 当前时间
	pendingUsers, err := l.queryPendingCancellations(batchSize)
	if err != nil {
		l.Logger.Errorf("查询待处理注销申请失败：%v", err)
		return 0, 0, 0, 0, err
	}

	total = int64(len(pendingUsers))
	if total == 0 {
		l.Logger.Infof("没有待处理的注销申请")
		return 0, 0, 0, 0, nil
	}

	l.Logger.Infof("扫描到 %d 个待处理的注销申请", total)

	// 2. 逐个处理每个用户的注销申请
	for _, user := range pendingUsers {
		result := l.processSingleUser(user.UserID, user.LogID, user.CoolingEndAt)

		if result.Success && result.Executed {
			executed++
		} else if result.Skipped {
			skipped++
		} else if result.Failed {
			failed++
		}

		// 记录单个用户的处理结果
		l.logSingleResult(result)
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	l.Logger.Infof(
		"账号注销定时任务执行完成，总用户数=%d, 成功=%d, 跳过=%d, 失败=%d, 耗时=%v",
		total, executed, skipped, failed, duration,
	)

	return total, executed, skipped, failed, nil
}

// queryPendingCancellations 查询待处理的注销申请
// 返回冷静期已结束但尚未执行注销的用户列表
func (l *AccountCancellationTask) queryPendingCancellations(batchSize int) ([]*dao.CancellationPendingUser, error) {
	ctx := l.ctx

	// 查询条件：
	// 1. status = 1 (冷静期中)
	// 2. cooling_end_at <= 当前时间
	// 3. 按 cooling_end_at 排序，优先处理先到期的
	query := `
		SELECT ucl.id, ucl.user_id, ucl.cooling_end_at
		FROM user_cancellation_log ucl
		WHERE ucl.status = $1
		  AND ucl.cooling_end_at <= $2
		ORDER BY ucl.cooling_end_at ASC
		LIMIT $3
	`

	rows, err := l.svcCtx.DB.QueryContext(ctx, query, dao.CancellationStatusCooling, time.Now(), batchSize)
	if err != nil {
		return nil, fmt.Errorf("query pending cancellations: %w", err)
	}
	defer rows.Close()

	var users []*dao.CancellationPendingUser
	for rows.Next() {
		var user dao.CancellationPendingUser
		if err := rows.Scan(&user.LogID, &user.UserID, &user.CoolingEndAt); err != nil {
			l.Logger.Errorf("扫描注销记录失败：%v", err)
			continue
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return users, nil
}

// processSingleUser 处理单个用户的注销申请
func (l *AccountCancellationTask) processSingleUser(userID, logID int64, coolingEndAt time.Time) *CancellationTaskResult {
	startTime := time.Now()

	result := &CancellationTaskResult{
		UserID:    userID,
		LogID:     logID,
		StartTime: startTime.Unix(),
	}

	// 1. 再次校验用户状态（确保未被撤销、未被恢复）
	user, err := l.checkUserStatus(userID)
	if err != nil {
		result.Failed = true
		result.Message = fmt.Sprintf("校验用户状态失败：%v", err)
		result.EndTime = time.Now().Unix()
		return result
	}

	if user == nil {
		result.Skipped = true
		result.Message = "用户不存在或已注销"
		result.EndTime = time.Now().Unix()
		return result
	}

	// 2. 检查是否仍在冷静期
	now := time.Now()
	if coolingEndAt.After(now) {
		result.Skipped = true
		result.Message = "冷静期尚未结束"
		result.EndTime = time.Now().Unix()
		return result
	}

	// 3. 检查是否已被用户撤销
	if user.CancellationCoolingUntil.Valid && user.CancellationCoolingUntil.Time.Before(now) {
		// 冷静期已过，但未被执行，需要检查是否有撤销记录
		withdrawn, err := l.checkIfWithdrawn(user.Id, logID)
		if err != nil {
			result.Failed = true
			result.Message = fmt.Sprintf("检查撤销状态失败：%v", err)
			result.EndTime = time.Now().Unix()
			return result
		}
		if withdrawn {
			result.Skipped = true
			result.Message = "用户已撤销注销申请"
			result.EndTime = time.Now().Unix()
			return result
		}
	}

	// 4. 执行注销流程（使用已有的事务方法）
	err = l.executeCancellation(userID, logID)
	if err != nil {
		result.Failed = true
		result.Message = fmt.Sprintf("执行注销失败：%v", err)
		result.EndTime = time.Now().Unix()
		return result
	}

	// 5. 注销成功
	result.Success = true
	result.Executed = true
	result.Message = "注销成功"
	result.EndTime = time.Now().Unix()

	l.Logger.Infof(
		"[AUDIT] 自动注销执行成功 user_id=%d log_id=%d cooling_end=%s",
		userID, logID, coolingEndAt.Format(time.RFC3339),
	)

	return result
}

// checkUserStatus 检查用户状态
func (l *AccountCancellationTask) checkUserStatus(userID int64) (*dao.User, error) {
	ctx := l.ctx

	query := `
		SELECT id, status, cancellation_cooling_until, account_cancelled_at
		FROM users
		WHERE id = $1
	`

	var user dao.User
	var cancelledAt sql.NullTime
	err := l.svcCtx.DB.QueryRowContext(ctx, query, userID).Scan(
		&user.Id,
		&user.Status,
		&user.CancellationCoolingUntil,
		&cancelledAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // 用户不存在
	}
	if err != nil {
		return nil, fmt.Errorf("check user status: %w", err)
	}

	user.AccountCancelledAt = cancelledAt
	return &user, nil
}

// checkIfWithdrawn 检查用户是否已撤销注销申请
func (l *AccountCancellationTask) checkIfWithdrawn(userID, logID int64) (bool, error) {
	ctx := l.ctx

	// 检查是否有撤销状态的记录
	query := `
		SELECT COUNT(*)
		FROM user_cancellation_log
		WHERE user_id = $1 AND id = $2 AND status = $3
	`

	var count int64
	err := l.svcCtx.DB.QueryRowContext(ctx, query, userID, logID, dao.CancellationStatusWithdrawn).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check withdrawal: %w", err)
	}

	return count > 0, nil
}

// executeCancellation 执行注销流程
// 调用已有的事务方法完成注销
func (l *AccountCancellationTask) executeCancellation(userID, logID int64) error {
	ctx := l.ctx

	// 开启事务
	tx, err := l.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 使用已有的方法执行注销
	// 这个方法已经包含了所有必要的清理步骤
	_, err = l.svcCtx.UserRepo.FinalizeCancellationIfDueWithUserLockedTx(ctx, tx, userID)
	if err != nil {
		return fmt.Errorf("finalize cancellation: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// logSingleResult 记录单个用户的处理结果
func (l *AccountCancellationTask) logSingleResult(result *CancellationTaskResult) {
	if result.Success && result.Executed {
		l.Logger.Infof(
			"注销成功 user_id=%d log_id=%d 耗时=%ds",
			result.UserID, result.LogID, result.EndTime-result.StartTime,
		)
	} else if result.Skipped {
		l.Logger.Infof(
			"注销跳过 user_id=%d log_id=%d 原因=%s",
			result.UserID, result.LogID, result.Message,
		)
	} else if result.Failed {
		l.Logger.Errorf(
			"注销失败 user_id=%d log_id=%d 原因=%s",
			result.UserID, result.LogID, result.Message,
		)
	}
}

// GetPendingCancellationsCount 获取待处理的注销申请数量（用于监控）
func (l *AccountCancellationTask) GetPendingCancellationsCount() (int64, error) {
	ctx := l.ctx

	query := `
		SELECT COUNT(*)
		FROM user_cancellation_log
		WHERE status = $1 AND cooling_end_at <= $2
	`

	var count int64
	err := l.svcCtx.DB.QueryRowContext(ctx, query, dao.CancellationStatusCooling, time.Now()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending cancellations: %w", err)
	}

	return count, nil
}

// GetCoolingPeriodUsers 获取所有处于冷静期的用户（用于监控）
func (l *AccountCancellationTask) GetCoolingPeriodUsers() (int64, error) {
	ctx := l.ctx

	query := `
		SELECT COUNT(*)
		FROM user_cancellation_log
		WHERE status = $1
	`

	var count int64
	err := l.svcCtx.DB.QueryRowContext(ctx, query, dao.CancellationStatusCooling).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count cooling period users: %w", err)
	}

	return count, nil
}
