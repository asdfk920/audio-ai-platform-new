package dao

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
)

// RequestMemberUnsubscribe 用户主动退订：到期前标记 + 关自动续费 + 审计日志（幂等：已 cancel_pending 则直接成功）。
func (r *MemberOrderRepo) RequestMemberUnsubscribe(ctx context.Context, userID int64, reasonCode, feedback string) error {
	if r == nil || r.db == nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	tx, err := r.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := r.requestMemberUnsubscribeTx(ctx, tx, userID, reasonCode, feedback); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *MemberOrderRepo) requestMemberUnsubscribeTx(ctx context.Context, tx *sql.Tx, userID int64, reasonCode, feedback string) error {
	now := time.Now()
	var memberID int64
	var exp sql.NullTime
	var perm sql.NullInt16
	var st int16
	var cancelP int16
	err := tx.QueryRowContext(ctx, `
SELECT id, expire_at, is_permanent, status, COALESCE(cancel_pending, 0)
  FROM public.user_member
 WHERE user_id = $1
 FOR UPDATE
`, userID).Scan(&memberID, &exp, &perm, &st, &cancelP)
	if errors.Is(err, sql.ErrNoRows) {
		return errorx.NewCodeError(errorx.CodeMemberNoSubscription, "")
	}
	if err != nil {
		return err
	}
	if st != 1 {
		return errorx.NewCodeError(errorx.CodeMemberNoSubscription, "")
	}
	if perm.Valid && perm.Int16 == 1 {
		return errorx.NewCodeError(errorx.CodeMemberPermanentNoUnsubscribe, "")
	}
	if !exp.Valid || !exp.Time.After(now) {
		return errorx.NewCodeError(errorx.CodeMemberExpiredNoUnsubscribe, "")
	}
	if cancelP == 1 {
		return nil
	}

	fb := strings.TrimSpace(feedback)
	if len(fb) > 4000 {
		fb = fb[:4000]
	}

	_, err = tx.ExecContext(ctx, `
UPDATE public.user_member
   SET cancel_pending = 1,
       cancel_requested_at = CURRENT_TIMESTAMP,
       auto_renew = 0,
       auto_renew_package_code = NULL,
       auto_renew_pay_type = NULL,
       auto_renew_updated_at = CURRENT_TIMESTAMP,
       updated_at = CURRENT_TIMESTAMP
 WHERE user_id = $1 AND id = $2
`, userID, memberID)
	if err != nil {
		return err
	}
	var fbArg any = nil
	if fb != "" {
		fbArg = fb
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO public.member_unsubscribe_log
  (user_id, user_member_id, unsubscribe_type, reason_code, feedback, scheduled_expire_at)
VALUES ($1, $2, 'user_initiated', $3, $4, $5)
`, userID, memberID, reasonCode, fbArg, exp.Time)
	return err
}

// RevokeMemberUnsubscribe 撤销「到期退订」标记（不恢复自动续费）。
func (r *MemberOrderRepo) RevokeMemberUnsubscribe(ctx context.Context, userID int64) error {
	if r == nil || r.db == nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	tx, err := r.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := r.revokeMemberUnsubscribeTx(ctx, tx, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *MemberOrderRepo) revokeMemberUnsubscribeTx(ctx context.Context, tx *sql.Tx, userID int64) error {
	now := time.Now()
	var memberID int64
	var exp sql.NullTime
	var perm sql.NullInt16
	var st int16
	var cancelP int16
	err := tx.QueryRowContext(ctx, `
SELECT id, expire_at, is_permanent, status, COALESCE(cancel_pending, 0)
  FROM public.user_member
 WHERE user_id = $1
 FOR UPDATE
`, userID).Scan(&memberID, &exp, &perm, &st, &cancelP)
	if errors.Is(err, sql.ErrNoRows) {
		return errorx.NewCodeError(errorx.CodeMemberNoSubscription, "")
	}
	if err != nil {
		return err
	}
	if st != 1 {
		return errorx.NewCodeError(errorx.CodeMemberUnsubscribeNotPending, "")
	}
	if cancelP != 1 {
		return errorx.NewCodeError(errorx.CodeMemberUnsubscribeNotPending, "")
	}
	if perm.Valid && perm.Int16 == 1 {
		return errorx.NewCodeError(errorx.CodeMemberUnsubscribeNotPending, "")
	}
	if !exp.Valid || !exp.Time.After(now) {
		return errorx.NewCodeError(errorx.CodeMemberExpiredNoUnsubscribe, "")
	}

	_, err = tx.ExecContext(ctx, `
UPDATE public.user_member
   SET cancel_pending = 0,
       cancel_requested_at = NULL,
       updated_at = CURRENT_TIMESTAMP
 WHERE user_id = $1 AND id = $2
`, userID, memberID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO public.member_unsubscribe_log
  (user_id, user_member_id, unsubscribe_type, reason_code, feedback, scheduled_expire_at)
VALUES ($1, $2, 'revoke', '', NULL, $3)
`, userID, memberID, exp.Time)
	return err
}
