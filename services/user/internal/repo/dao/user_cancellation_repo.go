package dao

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/pkg/passwd"
	"github.com/zeromicro/go-zero/core/logx"
)

// CancellationPendingUser 待处理的注销申请用户
type CancellationPendingUser struct {
	LogID      int64     `json:"log_id"`       // 注销记录 ID
	UserID     int64     `json:"user_id"`      // 用户 ID
	CoolingEndAt time.Time `json:"cooling_end_at"` // 冷静期结束时间
}

// 注销流水状态（与迁移注释一致）
const (
	CancellationStatusCooling   int16 = 1
	CancellationStatusExecuted  int16 = 2
	CancellationStatusWithdrawn int16 = 3
)

func (r *UserRepo) loadUserForCancellationTx(ctx context.Context, tx *sql.Tx, userID int64) (*User, error) {
	if r == nil || tx == nil || userID <= 0 {
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	var u User
	err := tx.QueryRowContext(ctx,
		`SELECT `+sqlUserProfileColumns+`
		 FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`,
		userID,
	).Scan(scanUserProfileArgs(&u)...)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FinalizeCancellationIfDueWithUserLockedTx 在已持有 users 行锁的事务内，检查并执行到期注销（逻辑销户）。
func (r *UserRepo) FinalizeCancellationIfDueWithUserLockedTx(ctx context.Context, tx *sql.Tx, userID int64) (finalized bool, err error) {
	if r == nil || tx == nil || userID <= 0 {
		return false, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	var logID int64
	var coolingEnd time.Time
	err = tx.QueryRowContext(ctx,
		`SELECT id, cooling_end_at FROM user_cancellation_log
		 WHERE user_id = $1 AND status = $2
		 ORDER BY id DESC LIMIT 1 FOR UPDATE`,
		userID, CancellationStatusCooling,
	).Scan(&logID, &coolingEnd)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		logx.WithContext(ctx).Errorf("FinalizeCancellation select log uid=%d: %v", userID, err)
		return false, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if coolingEnd.After(time.Now()) {
		return false, nil
	}
	if err := r.executeLogicalCancellationTx(ctx, tx, userID, logID); err != nil {
		return false, err
	}
	return true, nil
}

func (r *UserRepo) executeLogicalCancellationTx(ctx context.Context, tx *sql.Tx, userID, logID int64) error {
	res, err := tx.ExecContext(ctx,
		`UPDATE user_cancellation_log SET status = $1, updated_at = NOW() WHERE id = $2 AND status = $3`,
		CancellationStatusExecuted, logID, CancellationStatusCooling,
	)
	if err != nil {
		logx.WithContext(ctx).Errorf("executeLogicalCancellation update log: %v", err)
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if n != 1 {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}

	anonEmail := fmt.Sprintf("deleted_%d@cancelled.invalid", userID)
	rnd := make([]byte, 24)
	if _, err := rand.Read(rnd); err != nil {
		logx.WithContext(ctx).Errorf("executeLogicalCancellation rand: %v", err)
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	placeholderPwd := hex.EncodeToString(rnd)
	salt, hash, err := passwd.HashPasswordWithNewSalt(placeholderPwd)
	if err != nil {
		logx.WithContext(ctx).Errorf("executeLogicalCancellation hash: %v", err)
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}

	nick := "已注销用户"
	_, err = tx.ExecContext(ctx,
		`UPDATE users SET
			email = $1, mobile = NULL, nickname = $2, avatar = NULL,
			password = $3, salt = $4, password_algo = $5,
			real_name_status = 0, real_name_time = NULL, real_name_type = NULL,
			cancellation_cooling_until = NULL, account_cancelled_at = NOW(), updated_at = NOW()
		 WHERE id = $6 AND deleted_at IS NULL`,
		anonEmail, nick, hash, salt, passwd.AlgoBcryptConcat, userID,
	)
	if err != nil {
		logx.WithContext(ctx).Errorf("executeLogicalCancellation update user: %v", err)
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_real_name_auth WHERE user_id = $1`, userID); err != nil {
		logx.WithContext(ctx).Errorf("executeLogicalCancellation del realname: %v", err)
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_auth WHERE user_id = $1`, userID); err != nil {
		logx.WithContext(ctx).Errorf("executeLogicalCancellation del oauth: %v", err)
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	logx.WithContext(ctx).Infof("[AUDIT] action=account_cancellation_executed user_id=%d log_id=%d", userID, logID)
	return nil
}

// TryFinalizeCancellationIfDue 尝试对指定用户执行到期注销（独立事务，会先锁 users 行）。
func (r *UserRepo) TryFinalizeCancellationIfDue(ctx context.Context, userID int64) (finalized bool, err error) {
	if r == nil || r.db == nil || userID <= 0 {
		return false, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	tx, err := r.BeginTx(ctx)
	if err != nil {
		logx.WithContext(ctx).Errorf("TryFinalizeCancellationIfDue BeginTx: %v", err)
		return false, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := r.loadUserForCancellationTx(ctx, tx, userID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		logx.WithContext(ctx).Errorf("TryFinalizeCancellationIfDue load user: %v", err)
		return false, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	ok, err := r.FinalizeCancellationIfDueWithUserLockedTx(ctx, tx, userID)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		logx.WithContext(ctx).Errorf("TryFinalizeCancellationIfDue Commit: %v", err)
		return false, errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	return ok, nil
}

// ApplyAccountCancellationTx 提交注销申请并进入冷静期（单事务）。
func (r *UserRepo) ApplyAccountCancellationTx(ctx context.Context, userID int64, reason *string, appliedIP, deviceInfo *string, coolingEndAt time.Time) error {
	if r == nil || r.db == nil || userID <= 0 {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	tx, err := r.BeginTx(ctx)
	if err != nil {
		logx.WithContext(ctx).Errorf("ApplyAccountCancellationTx BeginTx: %v", err)
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err = r.loadUserForCancellationTx(ctx, tx, userID); err != nil {
		if err == sql.ErrNoRows {
			return errorx.NewDefaultError(errorx.CodeUserNotFound)
		}
		logx.WithContext(ctx).Errorf("ApplyAccountCancellationTx load user: %v", err)
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	if _, err := r.FinalizeCancellationIfDueWithUserLockedTx(ctx, tx, userID); err != nil {
		return err
	}

	var u *User
	u, err = r.loadUserForCancellationTx(ctx, tx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errorx.NewDefaultError(errorx.CodeUserNotFound)
		}
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if u.AccountCancelledAt.Valid {
		return errorx.NewDefaultError(errorx.CodeAccountCancelled)
	}
	now := time.Now()
	if u.CancellationCoolingUntil.Valid && u.CancellationCoolingUntil.Time.After(now) {
		return errorx.NewDefaultError(errorx.CodeCancellationAlreadyPending)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO user_cancellation_log (user_id, reason, agreement_signed_at, status, cooling_end_at, applied_ip, device_info)
		 VALUES ($1, $2, NOW(), $3, $4, $5, $6)`,
		userID, nullableStrPtr(reason), CancellationStatusCooling, coolingEndAt, nullableStrPtr(appliedIP), nullableStrPtr(deviceInfo),
	)
	if err != nil {
		logx.WithContext(ctx).Errorf("ApplyAccountCancellationTx insert log: %v", err)
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE users SET cancellation_cooling_until = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`,
		coolingEndAt, userID,
	)
	if err != nil {
		logx.WithContext(ctx).Errorf("ApplyAccountCancellationTx update user: %v", err)
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	if err := tx.Commit(); err != nil {
		logx.WithContext(ctx).Errorf("ApplyAccountCancellationTx Commit: %v", err)
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	logx.WithContext(ctx).Infof("[AUDIT] action=account_cancellation_applied user_id=%d cooling_end=%s", userID, coolingEndAt.Format(time.RFC3339))
	return nil
}

func nullableStrPtr(s *string) interface{} {
	if s == nil {
		return nil
	}
	if *s == "" {
		return nil
	}
	return *s
}

// WithdrawAccountCancellationTx 冷静期内撤销注销申请。
func (r *UserRepo) WithdrawAccountCancellationTx(ctx context.Context, userID int64) error {
	if r == nil || r.db == nil || userID <= 0 {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	tx, err := r.BeginTx(ctx)
	if err != nil {
		logx.WithContext(ctx).Errorf("WithdrawAccountCancellationTx BeginTx: %v", err)
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := r.loadUserForCancellationTx(ctx, tx, userID); err != nil {
		if err == sql.ErrNoRows {
			return errorx.NewDefaultError(errorx.CodeUserNotFound)
		}
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	finalized, err := r.FinalizeCancellationIfDueWithUserLockedTx(ctx, tx, userID)
	if err != nil {
		return err
	}
	if finalized {
		return errorx.NewDefaultError(errorx.CodeCancellationNotInCooling)
	}

	var logID int64
	var coolingEnd time.Time
	err = tx.QueryRowContext(ctx,
		`SELECT id, cooling_end_at FROM user_cancellation_log
		 WHERE user_id = $1 AND status = $2
		 ORDER BY id DESC LIMIT 1 FOR UPDATE`,
		userID, CancellationStatusCooling,
	).Scan(&logID, &coolingEnd)
	if err == sql.ErrNoRows {
		return errorx.NewDefaultError(errorx.CodeCancellationNotInCooling)
	}
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	if !coolingEnd.After(time.Now()) {
		// 已到期应由 Finalize 处理；并发下兜底
		return errorx.NewDefaultError(errorx.CodeCancellationNotInCooling)
	}

	res, err := tx.ExecContext(ctx,
		`UPDATE user_cancellation_log SET status = $1, updated_at = NOW() WHERE id = $2 AND status = $3`,
		CancellationStatusWithdrawn, logID, CancellationStatusCooling,
	)
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	n, err := res.RowsAffected()
	if err != nil || n != 1 {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE users SET cancellation_cooling_until = NULL, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		userID,
	)
	if err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}

	if err := tx.Commit(); err != nil {
		return errorx.NewDefaultError(errorx.CodeDatabaseError)
	}
	logx.WithContext(ctx).Infof("[AUDIT] action=account_cancellation_withdrawn user_id=%d log_id=%d", userID, logID)
	return nil
}
