package dao

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	FamilyStatusActive   int16 = 1
	FamilyStatusInactive int16 = 2

	FamilyMemberStatusActive  int16 = 1
	FamilyMemberStatusRemoved int16 = 2

	FamilyRoleOwner      = "owner"
	FamilyRoleSuperAdmin = "super_admin"
	FamilyRoleMember     = "member"

	FamilyInviteStatusPending   = "pending"
	FamilyInviteStatusAccepted  = "accepted"
	FamilyInviteStatusExpired   = "expired"
	FamilyInviteStatusCancelled = "cancelled"

	ShareTypePermanent  = "permanent"
	ShareTypeTemporary  = "temporary"
	ShareTypeTimeWindow = "time_window"

	PermissionLevelFullControl = "full_control"
	PermissionLevelPartial     = "partial_control"
	PermissionLevelViewOnly    = "view_only"

	DeviceShareStatusPending = "pending"
	DeviceShareStatusActive  = "active"
	DeviceShareStatusRevoked = "revoked"
	DeviceShareStatusExpired = "expired"
	DeviceShareStatusQuit    = "quit"
)

type FamilyRow struct {
	ID          int64
	OwnerUserID int64
	Name        string
	Status      int16
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type FamilyMemberRow struct {
	ID        int64
	FamilyID  int64
	UserID    int64
	Role      string
	Status    int16
	JoinedAt  time.Time
	InvitedBy int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FamilyInviteRow struct {
	ID            int64
	FamilyID      int64
	InviteCode    string
	TargetUserID  sql.NullInt64
	TargetAccount string
	Role          string
	Status        string
	ExpiresAt     sql.NullTime
	CreatedBy     int64
	Remark        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type DeviceShareRow struct {
	ID              int64
	FamilyID        int64
	DeviceID        int64
	DeviceSN        string
	DeviceName      string
	OwnerUserID     int64
	SharedUserID    int64
	TargetAccount   string
	InviteCode      string
	ShareType       string
	PermissionLevel string
	PermissionRaw   []byte
	StartAt         sql.NullTime
	EndAt           sql.NullTime
	Status          string
	ConfirmedAt     sql.NullTime
	RevokedAt       sql.NullTime
	CreatedBy       int64
	Remark          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type DeviceShareViewRow struct {
	DeviceShareRow
	OwnerNickname  string
	SharedNickname string
	FamilyName     string
}

type SharedDeviceListRow struct {
	ShareID         int64
	DeviceID        int64
	DeviceSN        string
	DeviceName      string
	DeviceModel     string
	SystemVersion   string
	FirmwareVersion string
	HardwareVersion string
	OnlineStatus    int16
	DeviceStatus    int16
	LastActiveAt    *time.Time
	FamilyID        int64
	FamilyRole      string
	PermissionLevel string
	PermissionRaw   []byte
	ShareType       string
	StartAt         *time.Time
	EndAt           *time.Time
	OwnerUserID     int64
	OwnerNickname   string
	AccessGrantedAt time.Time
}

func FindUserByAccount(ctx context.Context, tx *sql.Tx, account string, userID int64) (*UserRow, error) {
	account = strings.TrimSpace(account)
	switch {
	case userID > 0:
		return FindUserByID(ctx, tx, userID)
	case account == "":
		return nil, sql.ErrNoRows
	}

	var row UserRow
	err := tx.QueryRowContext(ctx, `
		SELECT id, email, mobile, COALESCE(nickname, ''), COALESCE(avatar, ''), status, created_at
		FROM public.users
		WHERE deleted_at IS NULL
		  AND (mobile = $1 OR email = $1)
		ORDER BY id ASC
		LIMIT 1
	`, account).Scan(
		&row.ID, &row.Email, &row.Mobile, &row.Nickname, &row.Avatar,
		&row.Status, &row.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func FindFamilyByOwner(ctx context.Context, tx *sql.Tx, ownerUserID int64) (*FamilyRow, error) {
	var row FamilyRow
	err := tx.QueryRowContext(ctx, `
		SELECT id, owner_user_id, name, status, created_at, updated_at
		FROM public.user_family
		WHERE owner_user_id = $1 AND status = $2
		ORDER BY id DESC
		LIMIT 1
	`, ownerUserID, FamilyStatusActive).Scan(
		&row.ID, &row.OwnerUserID, &row.Name, &row.Status, &row.CreatedAt, &row.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func FindCurrentFamilyByUser(ctx context.Context, tx *sql.Tx, userID int64) (*FamilyRow, *FamilyMemberRow, error) {
	var family FamilyRow
	var member FamilyMemberRow
	err := tx.QueryRowContext(ctx, `
		SELECT f.id, f.owner_user_id, f.name, f.status, f.created_at, f.updated_at,
		       m.id, m.family_id, m.user_id, m.role, m.status, m.joined_at, m.invited_by, m.created_at, m.updated_at
		FROM public.user_family_member m
		JOIN public.user_family f ON f.id = m.family_id
		WHERE m.user_id = $1
		  AND m.status = $2
		  AND f.status = $3
		ORDER BY CASE WHEN m.role = 'owner' THEN 0 ELSE 1 END, m.id ASC
		LIMIT 1
	`, userID, FamilyMemberStatusActive, FamilyStatusActive).Scan(
		&family.ID, &family.OwnerUserID, &family.Name, &family.Status, &family.CreatedAt, &family.UpdatedAt,
		&member.ID, &member.FamilyID, &member.UserID, &member.Role, &member.Status, &member.JoinedAt,
		&member.InvitedBy, &member.CreatedAt, &member.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return &family, &member, nil
}

func FindFamilyMember(ctx context.Context, tx *sql.Tx, familyID, userID int64) (*FamilyMemberRow, error) {
	var row FamilyMemberRow
	err := tx.QueryRowContext(ctx, `
		SELECT id, family_id, user_id, role, status, joined_at, invited_by, created_at, updated_at
		FROM public.user_family_member
		WHERE family_id = $1 AND user_id = $2
		ORDER BY id DESC
		LIMIT 1
	`, familyID, userID).Scan(
		&row.ID, &row.FamilyID, &row.UserID, &row.Role, &row.Status, &row.JoinedAt,
		&row.InvitedBy, &row.CreatedAt, &row.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func InsertFamily(ctx context.Context, tx *sql.Tx, ownerUserID int64, name string) (*FamilyRow, error) {
	row := &FamilyRow{}
	err := tx.QueryRowContext(ctx, `
		INSERT INTO public.user_family (owner_user_id, name, status)
		VALUES ($1, $2, $3)
		RETURNING id, owner_user_id, name, status, created_at, updated_at
	`, ownerUserID, strings.TrimSpace(name), FamilyStatusActive).Scan(
		&row.ID, &row.OwnerUserID, &row.Name, &row.Status, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return row, nil
}

func InsertFamilyMember(ctx context.Context, tx *sql.Tx, familyID, userID, invitedBy int64, role string) (*FamilyMemberRow, error) {
	row := &FamilyMemberRow{}
	err := tx.QueryRowContext(ctx, `
		INSERT INTO public.user_family_member (family_id, user_id, role, status, invited_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, family_id, user_id, role, status, joined_at, invited_by, created_at, updated_at
	`, familyID, userID, role, FamilyMemberStatusActive, invitedBy).Scan(
		&row.ID, &row.FamilyID, &row.UserID, &row.Role, &row.Status, &row.JoinedAt,
		&row.InvitedBy, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return row, nil
}

func ReactivateFamilyMember(ctx context.Context, tx *sql.Tx, memberID, invitedBy int64, role string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE public.user_family_member
		SET role = $1,
		    status = $2,
		    invited_by = $3,
		    joined_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $4
	`, role, FamilyMemberStatusActive, invitedBy, memberID)
	return err
}

func InsertFamilyInvite(ctx context.Context, tx *sql.Tx, familyID int64, inviteCode string, targetUserID *int64, targetAccount, role string, expiresAt *time.Time, createdBy int64, remark string) (*FamilyInviteRow, error) {
	row := &FamilyInviteRow{}
	var target sql.NullInt64
	if targetUserID != nil && *targetUserID > 0 {
		target.Valid = true
		target.Int64 = *targetUserID
	}
	var expire sql.NullTime
	if expiresAt != nil {
		expire.Valid = true
		expire.Time = *expiresAt
	}
	err := tx.QueryRowContext(ctx, `
		INSERT INTO public.user_family_invite (family_id, invite_code, target_user_id, target_account, role, status, expires_at, created_by, remark)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, family_id, invite_code, target_user_id, target_account, role, status, expires_at, created_by, remark, created_at, updated_at
	`, familyID, inviteCode, target, strings.TrimSpace(targetAccount), role, FamilyInviteStatusPending, expire, createdBy, strings.TrimSpace(remark)).Scan(
		&row.ID, &row.FamilyID, &row.InviteCode, &row.TargetUserID, &row.TargetAccount,
		&row.Role, &row.Status, &row.ExpiresAt, &row.CreatedBy, &row.Remark, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return row, nil
}

func FindFamilyInviteByCode(ctx context.Context, tx *sql.Tx, inviteCode string, forUpdate bool) (*FamilyInviteRow, error) {
	query := `
		SELECT id, family_id, invite_code, target_user_id, target_account, role, status, expires_at, created_by, remark, created_at, updated_at
		FROM public.user_family_invite
		WHERE invite_code = $1
		LIMIT 1`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	row := &FamilyInviteRow{}
	err := tx.QueryRowContext(ctx, query, strings.TrimSpace(inviteCode)).Scan(
		&row.ID, &row.FamilyID, &row.InviteCode, &row.TargetUserID, &row.TargetAccount,
		&row.Role, &row.Status, &row.ExpiresAt, &row.CreatedBy, &row.Remark, &row.CreatedAt, &row.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

func UpdateFamilyInviteStatus(ctx context.Context, tx *sql.Tx, inviteID int64, status string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE public.user_family_invite
		SET status = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, status, inviteID)
	return err
}

func ListFamilyMembers(ctx context.Context, tx *sql.Tx, familyID int64) ([]*FamilyMemberRow, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id, family_id, user_id, role, status, joined_at, invited_by, created_at, updated_at
		FROM public.user_family_member
		WHERE family_id = $1 AND status = $2
		ORDER BY CASE role WHEN 'owner' THEN 0 WHEN 'super_admin' THEN 1 ELSE 2 END, joined_at ASC
	`, familyID, FamilyMemberStatusActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*FamilyMemberRow
	for rows.Next() {
		row := &FamilyMemberRow{}
		if err := rows.Scan(&row.ID, &row.FamilyID, &row.UserID, &row.Role, &row.Status, &row.JoinedAt, &row.InvitedBy, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func UpdateFamilyMemberRole(ctx context.Context, tx *sql.Tx, memberID int64, role string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE public.user_family_member
		SET role = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, role, memberID)
	return err
}

func RemoveFamilyMember(ctx context.Context, tx *sql.Tx, memberID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE public.user_family_member
		SET status = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, FamilyMemberStatusRemoved, memberID)
	return err
}

func FindActiveBindByUserAndSN(ctx context.Context, tx *sql.Tx, userID int64, sn string) (*UserDeviceBindRow, error) {
	var row UserDeviceBindRow
	err := tx.QueryRowContext(ctx, `
		SELECT id, user_id, device_id, sn, COALESCE(alias,''),
		       status, bound_at, unbound_at
		FROM public.user_device_bind
		WHERE user_id = $1 AND sn = $2 AND status = 1
		LIMIT 1
	`, userID, strings.TrimSpace(sn)).Scan(
		&row.ID, &row.UserID, &row.DeviceID, &row.SN, &row.Alias, &row.Status, &row.BoundAt, &row.UnboundAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.DeviceName = row.Alias
	return &row, nil
}

func InsertDeviceShare(ctx context.Context, tx *sql.Tx, in DeviceShareRow) (*DeviceShareRow, error) {
	row := &DeviceShareRow{}
	var startAt, endAt sql.NullTime
	if in.StartAt.Valid {
		startAt = in.StartAt
	}
	if in.EndAt.Valid {
		endAt = in.EndAt
	}
	err := tx.QueryRowContext(ctx, `
		INSERT INTO public.user_device_share
		  (family_id, device_id, device_sn, device_name, owner_user_id, shared_user_id, target_account, invite_code, share_type,
		   permission_level, permission_payload, start_at, end_at, status, created_by, remark)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
		        $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, family_id, device_id, device_sn, device_name, owner_user_id, shared_user_id, target_account, invite_code,
		          share_type, permission_level, permission_payload, start_at, end_at, status, confirmed_at, revoked_at, created_by, remark, created_at, updated_at
	`, in.FamilyID, in.DeviceID, in.DeviceSN, in.DeviceName, in.OwnerUserID, in.SharedUserID, in.TargetAccount, in.InviteCode,
		in.ShareType, in.PermissionLevel, json.RawMessage(in.PermissionRaw), startAt, endAt, in.Status, in.CreatedBy, in.Remark).Scan(
		&row.ID, &row.FamilyID, &row.DeviceID, &row.DeviceSN, &row.DeviceName, &row.OwnerUserID, &row.SharedUserID,
		&row.TargetAccount, &row.InviteCode, &row.ShareType, &row.PermissionLevel, &row.PermissionRaw,
		&row.StartAt, &row.EndAt, &row.Status, &row.ConfirmedAt, &row.RevokedAt, &row.CreatedBy, &row.Remark, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return row, nil
}

func FindDeviceShareByInviteCode(ctx context.Context, tx *sql.Tx, inviteCode string, forUpdate bool) (*DeviceShareRow, error) {
	query := `
		SELECT id, family_id, device_id, device_sn, device_name, owner_user_id, shared_user_id, target_account, invite_code,
		       share_type, permission_level, permission_payload, start_at, end_at, status, confirmed_at, revoked_at, created_by, remark, created_at, updated_at
		FROM public.user_device_share
		WHERE invite_code = $1
		LIMIT 1`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	row := &DeviceShareRow{}
	err := tx.QueryRowContext(ctx, query, strings.TrimSpace(inviteCode)).Scan(
		&row.ID, &row.FamilyID, &row.DeviceID, &row.DeviceSN, &row.DeviceName, &row.OwnerUserID, &row.SharedUserID,
		&row.TargetAccount, &row.InviteCode, &row.ShareType, &row.PermissionLevel, &row.PermissionRaw,
		&row.StartAt, &row.EndAt, &row.Status, &row.ConfirmedAt, &row.RevokedAt, &row.CreatedBy, &row.Remark, &row.CreatedAt, &row.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

func FindDeviceShareByID(ctx context.Context, tx *sql.Tx, shareID int64) (*DeviceShareViewRow, error) {
	row := &DeviceShareViewRow{}
	err := tx.QueryRowContext(ctx, `
		SELECT s.id, s.family_id, s.device_id, s.device_sn, s.device_name, s.owner_user_id, s.shared_user_id, s.target_account, s.invite_code,
		       s.share_type, s.permission_level, s.permission_payload, s.start_at, s.end_at, s.status, s.confirmed_at, s.revoked_at, s.created_by, s.remark, s.created_at, s.updated_at,
		       COALESCE(ou.nickname, ''), COALESCE(su.nickname, ''), COALESCE(f.name, '')
		FROM public.user_device_share s
		JOIN public.user_family f ON f.id = s.family_id
		LEFT JOIN public.users ou ON ou.id = s.owner_user_id
		LEFT JOIN public.users su ON su.id = s.shared_user_id
		WHERE s.id = $1
		LIMIT 1
	`, shareID).Scan(
		&row.ID, &row.FamilyID, &row.DeviceID, &row.DeviceSN, &row.DeviceName, &row.OwnerUserID, &row.SharedUserID, &row.TargetAccount, &row.InviteCode,
		&row.ShareType, &row.PermissionLevel, &row.PermissionRaw, &row.StartAt, &row.EndAt, &row.Status, &row.ConfirmedAt, &row.RevokedAt, &row.CreatedBy, &row.Remark, &row.CreatedAt, &row.UpdatedAt,
		&row.OwnerNickname, &row.SharedNickname, &row.FamilyName,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

func FindActiveShareForDeviceUser(ctx context.Context, tx *sql.Tx, deviceID, sharedUserID int64) (*DeviceShareRow, error) {
	row := &DeviceShareRow{}
	err := tx.QueryRowContext(ctx, `
		SELECT id, family_id, device_id, device_sn, device_name, owner_user_id, shared_user_id, target_account, invite_code,
		       share_type, permission_level, permission_payload, start_at, end_at, status, confirmed_at, revoked_at, created_by, remark, created_at, updated_at
		FROM public.user_device_share
		WHERE device_id = $1 AND shared_user_id = $2
		  AND status IN ($3, $4)
		ORDER BY id DESC
		LIMIT 1
	`, deviceID, sharedUserID, DeviceShareStatusPending, DeviceShareStatusActive).Scan(
		&row.ID, &row.FamilyID, &row.DeviceID, &row.DeviceSN, &row.DeviceName, &row.OwnerUserID, &row.SharedUserID,
		&row.TargetAccount, &row.InviteCode, &row.ShareType, &row.PermissionLevel, &row.PermissionRaw,
		&row.StartAt, &row.EndAt, &row.Status, &row.ConfirmedAt, &row.RevokedAt, &row.CreatedBy, &row.Remark, &row.CreatedAt, &row.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

func UpdateDeviceShareAccepted(ctx context.Context, tx *sql.Tx, shareID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE public.user_device_share
		SET status = $1,
		    confirmed_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, DeviceShareStatusActive, shareID)
	return err
}

func UpdateDeviceShareStatus(ctx context.Context, tx *sql.Tx, shareID int64, status string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE public.user_device_share
		SET status = $1,
		    revoked_at = CASE WHEN $1 IN ('revoked', 'expired', 'quit') THEN CURRENT_TIMESTAMP ELSE revoked_at END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, status, shareID)
	return err
}

func InsertDeviceShareLog(ctx context.Context, tx *sql.Tx, shareID, familyID, deviceID int64, deviceSN, opType, opContent string, operatorUserID int64, operatorRole string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO public.user_device_share_log
		  (share_id, family_id, device_id, device_sn, op_type, op_content, operator_user_id, operator_role)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, shareID, familyID, deviceID, deviceSN, opType, opContent, operatorUserID, operatorRole)
	return err
}

func ListSharesByOwner(ctx context.Context, tx *sql.Tx, ownerUserID int64, status string) ([]*DeviceShareViewRow, error) {
	args := []any{ownerUserID}
	query := `
		SELECT s.id, s.family_id, s.device_id, s.device_sn, s.device_name, s.owner_user_id, s.shared_user_id, s.target_account, s.invite_code,
		       s.share_type, s.permission_level, s.permission_payload, s.start_at, s.end_at, s.status, s.confirmed_at, s.revoked_at, s.created_by, s.remark, s.created_at, s.updated_at,
		       COALESCE(ou.nickname, ''), COALESCE(su.nickname, ''), COALESCE(f.name, '')
		FROM public.user_device_share s
		JOIN public.user_family f ON f.id = s.family_id
		LEFT JOIN public.users ou ON ou.id = s.owner_user_id
		LEFT JOIN public.users su ON su.id = s.shared_user_id
		WHERE s.owner_user_id = $1`
	if strings.TrimSpace(status) != "" {
		query += ` AND s.status = $2`
		args = append(args, strings.TrimSpace(status))
	}
	query += ` ORDER BY s.created_at DESC`
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DeviceShareViewRow
	for rows.Next() {
		row := &DeviceShareViewRow{}
		if err := rows.Scan(
			&row.ID, &row.FamilyID, &row.DeviceID, &row.DeviceSN, &row.DeviceName, &row.OwnerUserID, &row.SharedUserID, &row.TargetAccount, &row.InviteCode,
			&row.ShareType, &row.PermissionLevel, &row.PermissionRaw, &row.StartAt, &row.EndAt, &row.Status, &row.ConfirmedAt, &row.RevokedAt, &row.CreatedBy, &row.Remark, &row.CreatedAt, &row.UpdatedAt,
			&row.OwnerNickname, &row.SharedNickname, &row.FamilyName,
		); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func ListSharesForReceiver(ctx context.Context, tx *sql.Tx, sharedUserID int64, status string) ([]*DeviceShareViewRow, error) {
	args := []any{sharedUserID}
	query := `
		SELECT s.id, s.family_id, s.device_id, s.device_sn, s.device_name, s.owner_user_id, s.shared_user_id, s.target_account, s.invite_code,
		       s.share_type, s.permission_level, s.permission_payload, s.start_at, s.end_at, s.status, s.confirmed_at, s.revoked_at, s.created_by, s.remark, s.created_at, s.updated_at,
		       COALESCE(ou.nickname, ''), COALESCE(su.nickname, ''), COALESCE(f.name, '')
		FROM public.user_device_share s
		JOIN public.user_family f ON f.id = s.family_id
		LEFT JOIN public.users ou ON ou.id = s.owner_user_id
		LEFT JOIN public.users su ON su.id = s.shared_user_id
		WHERE s.shared_user_id = $1`
	if strings.TrimSpace(status) != "" {
		query += ` AND s.status = $2`
		args = append(args, strings.TrimSpace(status))
	}
	query += ` ORDER BY s.created_at DESC`
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DeviceShareViewRow
	for rows.Next() {
		row := &DeviceShareViewRow{}
		if err := rows.Scan(
			&row.ID, &row.FamilyID, &row.DeviceID, &row.DeviceSN, &row.DeviceName, &row.OwnerUserID, &row.SharedUserID, &row.TargetAccount, &row.InviteCode,
			&row.ShareType, &row.PermissionLevel, &row.PermissionRaw, &row.StartAt, &row.EndAt, &row.Status, &row.ConfirmedAt, &row.RevokedAt, &row.CreatedBy, &row.Remark, &row.CreatedAt, &row.UpdatedAt,
			&row.OwnerNickname, &row.SharedNickname, &row.FamilyName,
		); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func ListSharedDevicesForUser(ctx context.Context, tx *sql.Tx, userID int64, deviceSn, deviceName, deviceModel string) ([]*SharedDeviceListRow, error) {
	cols, err := getDeviceTableColumns(ctx, tx)
	if err != nil {
		return nil, err
	}
	lastActiveAtExpr, _, _, _ := buildDeviceOptionalSelects(cols, "d")
	args := []any{userID, DeviceShareStatusActive, FamilyMemberStatusActive}
	where := []string{
		"s.shared_user_id = $1",
		"s.status = $2",
		"m.status = $3",
	}
	argIndex := 4
	if v := strings.TrimSpace(deviceSn); v != "" {
		where = append(where, fmt.Sprintf("s.device_sn ILIKE $%d", argIndex))
		args = append(args, "%"+v+"%")
		argIndex++
	}
	if v := strings.TrimSpace(deviceName); v != "" {
		where = append(where, fmt.Sprintf("s.device_name ILIKE $%d", argIndex))
		args = append(args, "%"+v+"%")
		argIndex++
	}
	if v := strings.TrimSpace(deviceModel); v != "" {
		where = append(where, fmt.Sprintf("d.product_key ILIKE $%d", argIndex))
		args = append(args, "%"+v+"%")
		argIndex++
	}
	query := fmt.Sprintf(`
		SELECT s.id, s.device_id, s.device_sn, s.device_name, d.product_key, d.firmware_version, d.firmware_version, d.hardware_version,
		       d.online_status, d.status, %s, s.family_id, m.role, s.permission_level, s.permission_payload, s.share_type,
		       s.start_at, s.end_at, s.owner_user_id, COALESCE(ou.nickname, ''), COALESCE(s.confirmed_at, s.created_at)
		FROM public.user_device_share s
		JOIN public.user_family_member m ON m.family_id = s.family_id AND m.user_id = s.shared_user_id
		JOIN public.device d ON d.id = s.device_id
		LEFT JOIN public.users ou ON ou.id = s.owner_user_id
		WHERE %s
		ORDER BY COALESCE(s.confirmed_at, s.created_at) DESC
	`, lastActiveAtExpr, strings.Join(where, " AND "))
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*SharedDeviceListRow
	for rows.Next() {
		row := &SharedDeviceListRow{}
		var lastActive sql.NullTime
		var startAt, endAt sql.NullTime
		if err := rows.Scan(
			&row.ShareID, &row.DeviceID, &row.DeviceSN, &row.DeviceName, &row.DeviceModel, &row.SystemVersion, &row.FirmwareVersion, &row.HardwareVersion,
			&row.OnlineStatus, &row.DeviceStatus, &lastActive, &row.FamilyID, &row.FamilyRole, &row.PermissionLevel, &row.PermissionRaw, &row.ShareType,
			&startAt, &endAt, &row.OwnerUserID, &row.OwnerNickname, &row.AccessGrantedAt,
		); err != nil {
			return nil, err
		}
		if lastActive.Valid {
			t := lastActive.Time
			row.LastActiveAt = &t
		}
		if startAt.Valid {
			t := startAt.Time
			row.StartAt = &t
		}
		if endAt.Valid {
			t := endAt.Time
			row.EndAt = &t
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func ExpireDueShares(ctx context.Context, tx *sql.Tx, now time.Time, limit int) ([]*DeviceShareRow, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := tx.QueryContext(ctx, `
		UPDATE public.user_device_share
		SET status = $1,
		    revoked_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id IN (
			SELECT id
			FROM public.user_device_share
			WHERE status = $2
			  AND end_at IS NOT NULL
			  AND end_at < $3
			ORDER BY end_at ASC
			LIMIT $4
			FOR UPDATE
		)
		RETURNING id, family_id, device_id, device_sn, device_name, owner_user_id, shared_user_id, target_account, invite_code,
		          share_type, permission_level, permission_payload, start_at, end_at, status, confirmed_at, revoked_at, created_by, remark, created_at, updated_at
	`, DeviceShareStatusExpired, DeviceShareStatusActive, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DeviceShareRow
	for rows.Next() {
		row := &DeviceShareRow{}
		if err := rows.Scan(
			&row.ID, &row.FamilyID, &row.DeviceID, &row.DeviceSN, &row.DeviceName, &row.OwnerUserID, &row.SharedUserID,
			&row.TargetAccount, &row.InviteCode, &row.ShareType, &row.PermissionLevel, &row.PermissionRaw,
			&row.StartAt, &row.EndAt, &row.Status, &row.ConfirmedAt, &row.RevokedAt, &row.CreatedBy, &row.Remark, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func RevokeSharesByDevice(ctx context.Context, tx *sql.Tx, ownerUserID, deviceID int64, reason string) ([]*DeviceShareRow, error) {
	rows, err := tx.QueryContext(ctx, `
		UPDATE public.user_device_share
		SET status = $1,
		    revoked_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP,
		    remark = CASE WHEN remark = '' THEN $2 ELSE remark END
		WHERE owner_user_id = $3
		  AND device_id = $4
		  AND status IN ($5, $6)
		RETURNING id, family_id, device_id, device_sn, device_name, owner_user_id, shared_user_id, target_account, invite_code,
		          share_type, permission_level, permission_payload, start_at, end_at, status, confirmed_at, revoked_at, created_by, remark, created_at, updated_at
	`, DeviceShareStatusRevoked, strings.TrimSpace(reason), ownerUserID, deviceID, DeviceShareStatusPending, DeviceShareStatusActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DeviceShareRow
	for rows.Next() {
		row := &DeviceShareRow{}
		if err := rows.Scan(
			&row.ID, &row.FamilyID, &row.DeviceID, &row.DeviceSN, &row.DeviceName, &row.OwnerUserID, &row.SharedUserID,
			&row.TargetAccount, &row.InviteCode, &row.ShareType, &row.PermissionLevel, &row.PermissionRaw,
			&row.StartAt, &row.EndAt, &row.Status, &row.ConfirmedAt, &row.RevokedAt, &row.CreatedBy, &row.Remark, &row.CreatedAt, &row.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}
