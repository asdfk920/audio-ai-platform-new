package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/pkg/util/userconst"
)

// MemberOrderRepo 会员套餐订单与支付履约（order_master / pay_log / member_pay_log / user_member）。
type MemberOrderRepo struct {
	db *sql.DB
}

func NewMemberOrderRepo(db *sql.DB) *MemberOrderRepo {
	return &MemberOrderRepo{db: db}
}

// BeginTx 开启事务（余额支付履约）。
func (r *MemberOrderRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if r == nil || r.db == nil {
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	return r.db.BeginTx(ctx, nil)
}

// MemberPackageRow 上架套餐一行。
type MemberPackageRow struct {
	PackageCode   string
	PackageName   string
	LevelCode     string
	PriceCent     int64
	ListPriceCent int64
	DurationDays  int
}

// GetMemberPackageByCode 查询启用套餐。
func (r *MemberOrderRepo) GetMemberPackageByCode(ctx context.Context, code string) (*MemberPackageRow, error) {
	if r == nil || r.db == nil {
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	var row MemberPackageRow
	err := r.db.QueryRowContext(ctx, `
SELECT package_code, package_name, level_code, price_cent,
       COALESCE(list_price_cent, price_cent), duration_days
  FROM public.member_package
 WHERE package_code = $1 AND status = 1
`, code).Scan(&row.PackageCode, &row.PackageName, &row.LevelCode, &row.PriceCent, &row.ListPriceCent, &row.DurationDays)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// OrderMasterRow 订单行（查询用）。
type OrderMasterRow struct {
	OrderNo      string
	UserID       int64
	PackageCode  string
	LevelCode    string
	DurationDays int
	AmountCent   int64
	PayType      int16
	PayStatus    int16
}

// InsertMemberOrder 创建待支付会员订单（含原价、优惠、biz_scene）。
func (r *MemberOrderRepo) InsertMemberOrder(ctx context.Context, orderNo string, userID int64, pkg *MemberPackageRow, payType int16, originalAmountCent, discountCent int64, bizScene string) error {
	if r == nil || r.db == nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO public.order_master
  (order_no, user_id, order_type, package_code, level_code, duration_days,
   amount_cent, original_amount_cent, discount_cent, biz_scene, pay_type, pay_status)
VALUES ($1, $2, 'member', $3, $4, $5, $6, $7, $8, $9, $10, 0)
`, orderNo, userID, pkg.PackageCode, pkg.LevelCode, pkg.DurationDays, pkg.PriceCent, originalAmountCent, discountCent, bizScene, payType)
	return err
}

// GetOrderByNoForUser 当前用户的订单。
func (r *MemberOrderRepo) GetOrderByNoForUser(ctx context.Context, orderNo string, userID int64) (*OrderMasterRow, error) {
	if r == nil || r.db == nil {
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	var o OrderMasterRow
	err := r.db.QueryRowContext(ctx, `
SELECT order_no, user_id, package_code, level_code, duration_days, amount_cent, pay_type, pay_status
  FROM public.order_master
 WHERE order_no = $1 AND user_id = $2
`, orderNo, userID).Scan(
		&o.OrderNo, &o.UserID, &o.PackageCode, &o.LevelCode, &o.DurationDays, &o.AmountCent, &o.PayType, &o.PayStatus,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// GetOrderByNo 按单号查询（支付回调）。
func (r *MemberOrderRepo) GetOrderByNo(ctx context.Context, tx *sql.Tx, orderNo string) (*OrderMasterRow, error) {
	if r == nil {
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	q := `
SELECT order_no, user_id, package_code, level_code, duration_days, amount_cent, pay_type, pay_status
  FROM public.order_master
 WHERE order_no = $1`
	var row *sql.Row
	if tx != nil {
		row = tx.QueryRowContext(ctx, q, orderNo)
	} else {
		row = r.db.QueryRowContext(ctx, q, orderNo)
	}
	var o OrderMasterRow
	err := row.Scan(&o.OrderNo, &o.UserID, &o.PackageCode, &o.LevelCode, &o.DurationDays, &o.AmountCent, &o.PayType, &o.PayStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// IsUserActiveForOrder 用户未注销且 status=正常。
func (r *MemberOrderRepo) IsUserActiveForOrder(ctx context.Context, userID int64) (ok bool, err error) {
	if r == nil || r.db == nil {
		return false, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	var st sql.NullInt64
	err = r.db.QueryRowContext(ctx, `SELECT status FROM public.users WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&st)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !st.Valid {
		return false, nil
	}
	return st.Int64 == int64(userconst.UserStatusActive), nil
}

// GetUserMemberRenewalState 读取用于续费计算的 user_member 快照。
func (r *MemberOrderRepo) GetUserMemberRenewalState(ctx context.Context, userID int64) (UserMemberRenewalState, error) {
	if r == nil || r.db == nil {
		return UserMemberRenewalState{}, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	var exp sql.NullTime
	var perm sql.NullInt16
	var st sql.NullInt16
	err := r.db.QueryRowContext(ctx, `
SELECT expire_at, is_permanent, status FROM public.user_member WHERE user_id = $1
`, userID).Scan(&exp, &perm, &st)
	if errors.Is(err, sql.ErrNoRows) {
		return UserMemberRenewalState{HasRow: false}, nil
	}
	if err != nil {
		return UserMemberRenewalState{}, err
	}
	return BuildRenewalState(exp, perm, st), nil
}

// PreviewNewExpireAt 计算若本单支付成功后新的到期时间（与 FulfillPaidOrderTx 一致）。
func (r *MemberOrderRepo) PreviewNewExpireAt(ctx context.Context, userID int64, durationDays int) (time.Time, error) {
	now := time.Now()
	st, err := r.GetUserMemberRenewalState(ctx, userID)
	if err != nil {
		return time.Time{}, err
	}
	newExp, _ := ComputeMemberRenewal(now, st, durationDays)
	return newExp, nil
}

// GetBalanceCent 用户余额（分）；tx 非空时加行锁。
func (r *MemberOrderRepo) GetBalanceCent(ctx context.Context, tx *sql.Tx, userID int64) (int64, error) {
	q := `SELECT balance_cent FROM public.users WHERE id = $1 AND deleted_at IS NULL`
	if tx != nil {
		q += ` FOR UPDATE`
	}
	var row *sql.Row
	if tx != nil {
		row = tx.QueryRowContext(ctx, q, userID)
	} else {
		row = r.db.QueryRowContext(ctx, q, userID)
	}
	var b sql.NullInt64
	if err := row.Scan(&b); err != nil {
		return 0, err
	}
	if !b.Valid {
		return 0, nil
	}
	return b.Int64, nil
}

// DeductBalanceCent 扣减余额（须在事务内 FOR UPDATE 后调用）。
func (r *MemberOrderRepo) DeductBalanceCent(ctx context.Context, tx *sql.Tx, userID int64, cents int64) error {
	res, err := tx.ExecContext(ctx, `
UPDATE public.users SET balance_cent = balance_cent - $2, updated_at = CURRENT_TIMESTAMP
 WHERE id = $1 AND deleted_at IS NULL AND balance_cent >= $2
`, userID, cents)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errorx.NewDefaultError(errorx.CodeMemberInsufficientBalance)
	}
	return nil
}

// FulfillPaidOrderTx 支付成功履约：幂等；更新订单、写流水、延长会员。
func (r *MemberOrderRepo) FulfillPaidOrderTx(ctx context.Context, tx *sql.Tx, orderNo, channelTradeNo string, payType int16, rawNotify *string) error {
	var o OrderMasterRow
	err := tx.QueryRowContext(ctx, `
SELECT order_no, user_id, package_code, level_code, duration_days, amount_cent, pay_type, pay_status
  FROM public.order_master
 WHERE order_no = $1
 FOR UPDATE
`, orderNo).Scan(&o.OrderNo, &o.UserID, &o.PackageCode, &o.LevelCode, &o.DurationDays, &o.AmountCent, &o.PayType, &o.PayStatus)
	if err != nil {
		return err
	}
	if o.PayStatus == 1 {
		return nil
	}
	if o.PayStatus != 0 {
		return errorx.NewDefaultError(errorx.CodeMemberOrderNotPending)
	}

	raw := ""
	if rawNotify != nil {
		raw = *rawNotify
	}
	trade := channelTradeNo
	if trade == "" {
		trade = fmt.Sprintf("BALANCE-%s", orderNo)
	}

	_, err = tx.ExecContext(ctx, `
UPDATE public.order_master
   SET pay_status = 1, channel_trade_no = $2, paid_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
 WHERE order_no = $1 AND pay_status = 0
`, orderNo, nullStr(trade))
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO public.pay_log (order_no, user_id, pay_type, amount_cent, trade_no, pay_status, raw_notify)
VALUES ($1, $2, $3, $4, $5, 1, $6)
`, orderNo, o.UserID, payType, o.AmountCent, trade, nullStr(raw))
	if err != nil {
		return err
	}

	now := time.Now()
	var oldExp sql.NullTime
	var perm sql.NullInt16
	var st sql.NullInt16
	err = tx.QueryRowContext(ctx, `
SELECT expire_at, is_permanent, status FROM public.user_member WHERE user_id = $1 FOR UPDATE
`, o.UserID).Scan(&oldExp, &perm, &st)
	var state UserMemberRenewalState
	if errors.Is(err, sql.ErrNoRows) {
		state = UserMemberRenewalState{HasRow: false}
	} else if err != nil {
		return err
	} else {
		state = BuildRenewalState(oldExp, perm, st)
	}

	var oldPtr *time.Time
	if state.ExpireValid {
		t := state.ExpireAt
		oldPtr = &t
	}

	if ShouldSkipMembershipExtend(state) {
		newForLog := now
		if state.ExpireValid {
			newForLog = state.ExpireAt
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO public.member_pay_log (user_id, order_no, package_code, level_code, duration_days, old_expire_at, new_expire_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`, o.UserID, orderNo, o.PackageCode, o.LevelCode, o.DurationDays, oldPtr, newForLog)
		return err
	}

	newExp, _ := ComputeMemberRenewal(now, state, o.DurationDays)

	_, err = tx.ExecContext(ctx, `
INSERT INTO public.user_member (user_id, level, level_code, expire_at, is_permanent, status, register_type, grant_by, created_at, updated_at)
VALUES ($1, 0, $2, $3, 0, 1, 'pay', 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (user_id) DO UPDATE SET
  level_code = EXCLUDED.level_code,
  expire_at = EXCLUDED.expire_at,
  is_permanent = 0,
  register_type = 'pay',
  status = 1,
  updated_at = CURRENT_TIMESTAMP
`, o.UserID, o.LevelCode, newExp)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO public.member_pay_log (user_id, order_no, package_code, level_code, duration_days, old_expire_at, new_expire_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`, o.UserID, orderNo, o.PackageCode, o.LevelCode, o.DurationDays, oldPtr, newExp)
	return err
}

// UserMemberRow 用户会员信息行。
type UserMemberRow struct {
	ID                   int64
	LevelCode            string
	ExpireAt             time.Time
	IsPermanent          int16
	Status               int16
	CreatedAt            time.Time
	AutoRenew            int16
	AutoRenewPackageCode string
	AutoRenewPayType     int16
	AutoRenewUpdatedAt   sql.NullTime
	CancelPending        int16
	CancelRequestedAt    sql.NullTime
}

// GetUserMemberInfo 查询用户会员信息。
func (r *MemberOrderRepo) GetUserMemberInfo(ctx context.Context, userID int64) (*UserMemberRow, error) {
	if r == nil || r.db == nil {
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	var row UserMemberRow
	var pkg sql.NullString
	var pay sql.NullInt16
	err := r.db.QueryRowContext(ctx, `
SELECT id, level_code, expire_at, is_permanent, status, created_at,
       auto_renew, auto_renew_package_code, auto_renew_pay_type, auto_renew_updated_at,
       cancel_pending, cancel_requested_at
  FROM public.user_member
 WHERE user_id = $1
`, userID).Scan(
		&row.ID, &row.LevelCode, &row.ExpireAt, &row.IsPermanent, &row.Status, &row.CreatedAt,
		&row.AutoRenew, &pkg, &pay, &row.AutoRenewUpdatedAt,
		&row.CancelPending, &row.CancelRequestedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if pkg.Valid {
		row.AutoRenewPackageCode = pkg.String
	}
	if pay.Valid {
		row.AutoRenewPayType = pay.Int16
	}
	return &row, nil
}

// AutoRenewCandidate 定时任务扫描：待关注自动续费的会员行。
type AutoRenewCandidate struct {
	UserID               int64
	LevelCode            string
	ExpireAt             time.Time
	AutoRenewPackageCode string
	AutoRenewPayType     int16
}

// ListAutoRenewCandidates 即将在 withinDays 天内到期、且开启自动续费占位开关的用户（非永久）。
func (r *MemberOrderRepo) ListAutoRenewCandidates(ctx context.Context, withinDays, limit int) ([]AutoRenewCandidate, error) {
	if r == nil || r.db == nil {
		return nil, errorx.NewDefaultError(errorx.CodeSystemError)
	}
	if withinDays <= 0 {
		withinDays = 7
	}
	if limit <= 0 {
		limit = 200
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT user_id, COALESCE(level_code,''), expire_at,
       COALESCE(auto_renew_package_code,''), COALESCE(auto_renew_pay_type, 0)
  FROM public.user_member
 WHERE auto_renew = 1
   AND status = 1
   AND COALESCE(is_permanent, 0) = 0
   AND expire_at IS NOT NULL
   AND expire_at > CURRENT_TIMESTAMP
   AND expire_at <= CURRENT_TIMESTAMP + ($1 * INTERVAL '1 day')
 ORDER BY expire_at ASC
 LIMIT $2
`, withinDays, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []AutoRenewCandidate
	for rows.Next() {
		var c AutoRenewCandidate
		if err := rows.Scan(&c.UserID, &c.LevelCode, &c.ExpireAt, &c.AutoRenewPackageCode, &c.AutoRenewPayType); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// UpdateUserMemberAutoRenew 更新自动续费意向（需已存在 user_member 行）。
func (r *MemberOrderRepo) UpdateUserMemberAutoRenew(ctx context.Context, userID int64, enabled bool, packageCode string, payType int16) error {
	if r == nil || r.db == nil {
		return errorx.NewDefaultError(errorx.CodeSystemError)
	}
	var ar int16
	if enabled {
		ar = 1
	}
	pc := strings.TrimSpace(packageCode)
	var pkg any
	var pt sql.NullInt16
	if enabled {
		if pc == "" {
			return errorx.NewCodeError(errorx.CodeInvalidParam, "开启自动续费时请传入套餐编码")
		}
		pkg = pc
		if payType < 1 || payType > 3 {
			return errorx.NewCodeError(errorx.CodeInvalidParam, "支付方式须为 1–3")
		}
		pt = sql.NullInt16{Int16: payType, Valid: true}
	} else {
		pkg = nil
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE public.user_member
   SET auto_renew = $2,
       auto_renew_package_code = $3,
       auto_renew_pay_type = $4,
       auto_renew_updated_at = CURRENT_TIMESTAMP,
       updated_at = CURRENT_TIMESTAMP
 WHERE user_id = $1
`, userID, ar, pkg, pt)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errorx.NewCodeError(errorx.CodeInvalidParam, "无会员档案，请先开通会员后再设置自动续费")
	}
	return nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
