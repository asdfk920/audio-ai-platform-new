package dao

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jacklau/audio-ai-platform/services/user/internal/userdomain/profile/realname"
)

// ErrRealNameAuthNotReviewable 当前流水状态不允许人工审核。
var ErrRealNameAuthNotReviewable = errors.New("realname: auth record not in pending manual state")

// UserRealNameAuth 实名流水（敏感字段仅存密文与脱敏展示）。
type UserRealNameAuth struct {
	ID                    int64
	UserID                int64
	CertType              int16
	RealNameMasked        string
	IdNumberEncrypted     string
	IdNumberLast4         string
	IdPhotoRef            sql.NullString
	IdCardFrontRef        sql.NullString
	IdCardBackRef         sql.NullString
	FaceDataRef           sql.NullString
	AuthStatus            int16
	ThirdPartyFlowNo      sql.NullString
	ThirdPartyChannel     sql.NullString
	ThirdPartyRawResponse sql.NullString
	FailReason            sql.NullString
	ReviewerNote          sql.NullString
	ReviewedAt            sql.NullTime
	ReviewedBy            sql.NullString
	DeviceInfo            sql.NullString
	CreatedAt             time.Time
}

// RealNameTxLoadUserForUpdate 事务内锁定用户行（换绑/实名等场景）。
// 如果 tx 为 nil，则使用普通查询（不加锁）。
func (r *UserRepo) RealNameTxLoadUserForUpdate(ctx context.Context, tx *sql.Tx, userID int64) (*User, error) {
	var u User
	var err error
	if tx != nil {
		err = tx.QueryRowContext(ctx,
			`SELECT `+sqlUserProfileColumns+` FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`,
			userID,
		).Scan(scanUserProfileArgs(&u)...)
	} else {
		err = r.db.QueryRowContext(ctx,
			`SELECT `+sqlUserProfileColumns+` FROM users WHERE id = $1 AND deleted_at IS NULL`,
			userID,
		).Scan(scanUserProfileArgs(&u)...)
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// RealNameTxInsertAuth 在事务内写入待三方核验流水（id_photo_ref 兼容旧客户端，通常与人像面一致）。
func (r *UserRepo) RealNameTxInsertAuth(ctx context.Context, tx *sql.Tx, userID int64, certType int16, maskedName, idEnc, idLast4 string, idPhotoRef, idFrontRef, idBackRef, faceRef *string, deviceInfo string) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
INSERT INTO user_real_name_auth (
  user_id, cert_type, real_name_masked, id_number_encrypted, id_number_last4,
  id_photo_ref, id_card_front_ref, id_card_back_ref, face_data_ref, auth_status, device_info
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		userID, certType, maskedName, idEnc, idLast4,
		nullableStr(idPhotoRef), nullableStr(idFrontRef), nullableStr(idBackRef), nullableStr(faceRef),
		realname.AuthPendingThirdParty, nullIfEmpty(deviceInfo),
	).Scan(&id)
	return id, err
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// RealNameTxMarkUserInProgress 事务内将用户标为实名审核中并写入证件类型。
func (r *UserRepo) RealNameTxMarkUserInProgress(ctx context.Context, tx *sql.Tx, userID int64, certType int16) error {
	_, err := tx.ExecContext(ctx, `
UPDATE users SET real_name_status = $2, real_name_type = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL`,
		userID, realname.UserRealNameInProgress, certType)
	return err
}

// RealNameUpdateAuthAfterThirdParty 更新流水上的三方结果字段。
func (r *UserRepo) RealNameUpdateAuthAfterThirdParty(ctx context.Context, authID int64, authStatus int16, flowNo, channel, raw string, failReason *string) error {
	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}
	var fr interface{}
	if failReason != nil {
		fr = *failReason
	}
	_, err := r.db.ExecContext(ctx, `
UPDATE user_real_name_auth SET
  auth_status = $2,
  third_party_flow_no = $3,
  third_party_channel = $4,
  third_party_raw_response = $5,
  fail_reason = $6,
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1`,
		authID, authStatus, strOrNil(flowNo), strOrNil(channel), strOrNil(raw), fr)
	return err
}

func strOrNil(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// RealNameUpdateUserStatus 更新用户主表实名状态；verified=true 时写入 real_name_time。
func (r *UserRepo) RealNameUpdateUserStatus(ctx context.Context, userID int64, status int16, certType int16, verified bool) error {
	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}
	if verified {
		_, err := r.db.ExecContext(ctx, `
UPDATE users SET real_name_status = $2, real_name_time = CURRENT_TIMESTAMP, real_name_type = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL`,
			userID, status, certType)
		return err
	}
	if status == realname.UserRealNameNone {
		_, err := r.db.ExecContext(ctx, `
UPDATE users SET real_name_status = 0, real_name_time = NULL, real_name_type = NULL, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL`, userID)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
UPDATE users SET real_name_status = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL`, userID, status)
	return err
}

// RealNameFindLatestAuthByUser 用户最近一次实名流水。
func (r *UserRepo) RealNameFindLatestAuthByUser(ctx context.Context, userID int64) (*UserRealNameAuth, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return nil, nil
	}
	row := r.db.QueryRowContext(ctx, `
SELECT id, user_id, cert_type, real_name_masked, id_number_encrypted, id_number_last4,
  id_photo_ref, id_card_front_ref, id_card_back_ref, face_data_ref, auth_status,
  third_party_flow_no, third_party_channel, third_party_raw_response, fail_reason,
  reviewer_note, reviewed_at, reviewed_by, device_info, created_at
FROM user_real_name_auth WHERE user_id = $1 ORDER BY id DESC LIMIT 1`, userID)
	return scanUserRealNameAuth(row)
}

// RealNameFindLatestSuccessfulPersonalAuth 最近一条已通过核验的个人身份证流水（auth_status 为三方通过或人工通过），用于实名换绑比对证件密文。
func (r *UserRepo) RealNameFindLatestSuccessfulPersonalAuth(ctx context.Context, userID int64) (*UserRealNameAuth, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return nil, nil
	}
	row := r.db.QueryRowContext(ctx, `
SELECT id, user_id, cert_type, real_name_masked, id_number_encrypted, id_number_last4,
  id_photo_ref, id_card_front_ref, id_card_back_ref, face_data_ref, auth_status,
  third_party_flow_no, third_party_channel, third_party_raw_response, fail_reason,
  reviewer_note, reviewed_at, reviewed_by, device_info, created_at
FROM user_real_name_auth
WHERE user_id = $1 AND cert_type = $2 AND auth_status IN ($3, $4)
ORDER BY id DESC LIMIT 1`,
		userID, realname.CertTypePersonal, realname.AuthThirdPartyPass, realname.AuthManualPass)
	return scanUserRealNameAuth(row)
}

// RealNameGetAuthByID 按主键取流水（管理端）。
func (r *UserRepo) RealNameGetAuthByID(ctx context.Context, authID int64) (*UserRealNameAuth, error) {
	if r == nil || r.db == nil || authID <= 0 {
		return nil, nil
	}
	row := r.db.QueryRowContext(ctx, `
SELECT id, user_id, cert_type, real_name_masked, id_number_encrypted, id_number_last4,
  id_photo_ref, id_card_front_ref, id_card_back_ref, face_data_ref, auth_status,
  third_party_flow_no, third_party_channel, third_party_raw_response, fail_reason,
  reviewer_note, reviewed_at, reviewed_by, device_info, created_at
FROM user_real_name_auth WHERE id = $1`, authID)
	return scanUserRealNameAuth(row)
}

func scanUserRealNameAuth(row *sql.Row) (*UserRealNameAuth, error) {
	var a UserRealNameAuth
	err := row.Scan(
		&a.ID, &a.UserID, &a.CertType, &a.RealNameMasked, &a.IdNumberEncrypted, &a.IdNumberLast4,
		&a.IdPhotoRef, &a.IdCardFrontRef, &a.IdCardBackRef, &a.FaceDataRef, &a.AuthStatus,
		&a.ThirdPartyFlowNo, &a.ThirdPartyChannel, &a.ThirdPartyRawResponse, &a.FailReason,
		&a.ReviewerNote, &a.ReviewedAt, &a.ReviewedBy, &a.DeviceInfo, &a.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// RealNameAdminFinalize 人工审核落库：更新流水 + 用户主表状态。
// user-api 不暴露该能力的 HTTP 接口；由独立管理后台（或其它服务）在鉴权后调用本方法或执行等价 SQL。
func (r *UserRepo) RealNameAdminFinalize(ctx context.Context, authID int64, approve bool, adminID, note string) error {
	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}
	tx, err := r.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	a, err := r.realNameGetAuthByIDTx(ctx, tx, authID)
	if err != nil {
		return err
	}
	if a == nil {
		return sql.ErrNoRows
	}
	if a.AuthStatus != realname.AuthPendingManual {
		return ErrRealNameAuthNotReviewable
	}

	newAuth := realname.AuthManualReject
	newUser := realname.UserRealNameFailed
	if approve {
		newAuth = realname.AuthManualPass
		newUser = realname.UserRealNameVerified
	}
	if approve {
		_, err = tx.ExecContext(ctx, `
UPDATE user_real_name_auth SET
  auth_status = $2, reviewer_note = $3, reviewed_by = $4, fail_reason = NULL,
  reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE id = $1`,
			authID, newAuth, nullIfEmpty(note), nullIfEmpty(adminID))
	} else {
		_, err = tx.ExecContext(ctx, `
UPDATE user_real_name_auth SET
  auth_status = $2, reviewer_note = $3, reviewed_by = $4, fail_reason = $3,
  reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE id = $1`,
			authID, newAuth, nullIfEmpty(note), nullIfEmpty(adminID))
	}
	if err != nil {
		return err
	}
	if approve {
		_, err = tx.ExecContext(ctx, `
UPDATE users SET real_name_status = $2, real_name_time = CURRENT_TIMESTAMP, real_name_type = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL`,
			a.UserID, newUser, a.CertType)
	} else {
		_, err = tx.ExecContext(ctx, `
UPDATE users SET real_name_status = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL`,
			a.UserID, newUser)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *UserRepo) realNameGetAuthByIDTx(ctx context.Context, tx *sql.Tx, authID int64) (*UserRealNameAuth, error) {
	row := tx.QueryRowContext(ctx, `
SELECT id, user_id, cert_type, real_name_masked, id_number_encrypted, id_number_last4,
  id_photo_ref, id_card_front_ref, id_card_back_ref, face_data_ref, auth_status,
  third_party_flow_no, third_party_channel, third_party_raw_response, fail_reason,
  reviewer_note, reviewed_at, reviewed_by, device_info, created_at
FROM user_real_name_auth WHERE id = $1`, authID)
	return scanUserRealNameAuth(row)
}

// RealNameUpdateAuthStatusSimple 仅更新流水状态（如三方通过后转待人工）。
func (r *UserRepo) RealNameUpdateAuthStatusSimple(ctx context.Context, authID int64, authStatus int16) error {
	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}
	_, err := r.db.ExecContext(ctx, `
UPDATE user_real_name_auth SET auth_status = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		authID, authStatus)
	return err
}

// RealNameMarkPendingManual 三方已通过，进入人工审核队列。
func (r *UserRepo) RealNameMarkPendingManual(ctx context.Context, authID int64, flowNo, channel, raw string) error {
	if r == nil || r.db == nil {
		return sql.ErrConnDone
	}
	_, err := r.db.ExecContext(ctx, `
UPDATE user_real_name_auth SET
  auth_status = $2,
  third_party_flow_no = $3,
  third_party_channel = $4,
  third_party_raw_response = $5,
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1`,
		authID, realname.AuthPendingManual, strOrNil(flowNo), strOrNil(channel), strOrNil(raw))
	return err
}
