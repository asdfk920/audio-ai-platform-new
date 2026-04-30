package devicesharesvc

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/familysvc"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
)

type Service struct {
	svcCtx    *svc.ServiceContext
	familySvc *familysvc.Service
}

type CreateShareInviteInput struct {
	OperatorUserID  int64
	DeviceSN        string
	TargetUserID    int64
	TargetAccount   string
	ShareType       string
	PermissionLevel string
	Permission      map[string]any
	StartAt         *time.Time
	EndAt           *time.Time
	Remark          string
}

type ShareView struct {
	ID              int64
	FamilyID        int64
	DeviceID        int64
	DeviceSN        string
	DeviceName      string
	OwnerUserID     int64
	OwnerNickname   string
	SharedUserID    int64
	SharedNickname  string
	TargetAccount   string
	InviteCode      string
	ShareType       string
	PermissionLevel string
	Permission      map[string]any
	Status          string
	FamilyName      string
	StartAt         *time.Time
	EndAt           *time.Time
	CreatedAt       time.Time
	ConfirmedAt     *time.Time
}

type AccessDecision struct {
	Allowed         bool
	AccessMode      string
	Role            string
	PermissionLevel string
	Permission      map[string]any
	OwnerUserID     int64
}

func New(svcCtx *svc.ServiceContext) *Service {
	return &Service{
		svcCtx:    svcCtx,
		familySvc: familysvc.New(svcCtx),
	}
}

func (s *Service) CreateShareInvite(ctx context.Context, in CreateShareInviteInput) (*ShareView, error) {
	tx, err := s.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	target, err := dao.FindUserByAccount(ctx, tx, strings.TrimSpace(in.TargetAccount), in.TargetUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errorx.NewCodeError(errorx.CodeUserNotFound, "")
		}
		return nil, err
	}
	if target.ID == in.OperatorUserID {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "不能分享给自己")
	}

	bind, err := dao.FindActiveBindByUserAndSN(ctx, tx, in.OperatorUserID, strings.TrimSpace(in.DeviceSN))
	if err != nil {
		return nil, err
	}

	var family *familysvc.FamilyView
	var operatorRole string
	var ownerUserID int64
	var deviceID int64
	var deviceName string
	if bind != nil {
		family, err = s.familySvc.EnsureFamilyForOwner(ctx, in.OperatorUserID, "")
		if err != nil {
			return nil, err
		}
		txLookup, lookupErr := s.svcCtx.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if lookupErr != nil {
			return nil, lookupErr
		}
		defer func() { _ = txLookup.Rollback() }()
		member, lookupErr := dao.FindFamilyMember(ctx, txLookup, family.ID, in.OperatorUserID)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if member == nil || member.Status != dao.FamilyMemberStatusActive {
			return nil, errorx.NewCodeError(errorx.CodeFamilyNotFound, "设备主人未加入家庭")
		}
		operatorRole = dao.FamilyRoleOwner
		ownerUserID = in.OperatorUserID
		deviceID = bind.DeviceID
		deviceName = bind.DeviceName
	} else {
		familyRow, member, err := dao.FindCurrentFamilyByUser(ctx, tx, in.OperatorUserID)
		if err != nil {
			return nil, err
		}
		if familyRow == nil || member == nil {
			return nil, errorx.NewCodeError(errorx.CodeDeviceShareForbiddenReshare, "当前账号无设备分享资格")
		}
		operatorRole = member.Role
		if operatorRole != dao.FamilyRoleSuperAdmin {
			return nil, errorx.NewCodeError(errorx.CodeDeviceShareForbiddenReshare, "")
		}
		family = &familysvc.FamilyView{ID: familyRow.ID, OwnerUserID: familyRow.OwnerUserID, Name: familyRow.Name}
		device, err := dao.FindDeviceBySN(ctx, tx, strings.TrimSpace(in.DeviceSN))
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, errorx.NewCodeError(errorx.CodeDeviceNotFound, "")
			}
			return nil, err
		}
		deviceID = device.ID
		deviceName = strings.TrimSpace(device.ProductKey)
		ownerBind, err := dao.FindActiveBindByDeviceID(ctx, tx, device.ID)
		if err != nil {
			return nil, err
		}
		if ownerBind == nil {
			return nil, errorx.NewCodeError(errorx.CodeDeviceNotBound, "")
		}
		ownerUserID = ownerBind.UserID
		if family.OwnerUserID != ownerUserID {
			return nil, errorx.NewCodeError(errorx.CodeDeviceShareNoPermission, "仅同家庭超级管理员可继续分享")
		}
	}

	member, err := dao.FindFamilyMember(ctx, tx, family.ID, target.ID)
	if err != nil {
		return nil, err
	}
	if member == nil || member.Status != dao.FamilyMemberStatusActive {
		return nil, errorx.NewCodeError(errorx.CodeFamilyMemberExists, "目标账号尚未加入当前家庭")
	}
	existing, err := dao.FindActiveShareForDeviceUser(ctx, tx, deviceID, target.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errorx.NewCodeError(errorx.CodeDeviceShareExists, "")
	}

	permissionRaw, err := json.Marshal(normalizePermissionPayload(in.PermissionLevel, in.Permission))
	if err != nil {
		return nil, err
	}
	code, err := shareInviteCode()
	if err != nil {
		return nil, err
	}
	shareRow, err := dao.InsertDeviceShare(ctx, tx, dao.DeviceShareRow{
		FamilyID:        family.ID,
		DeviceID:        deviceID,
		DeviceSN:        strings.TrimSpace(in.DeviceSN),
		DeviceName:      deviceName,
		OwnerUserID:     ownerUserID,
		SharedUserID:    target.ID,
		TargetAccount:   firstNonEmpty(in.TargetAccount, target.Mobile.String, target.Email.String),
		InviteCode:      code,
		ShareType:       normalizeShareType(in.ShareType),
		PermissionLevel: normalizePermissionLevel(in.PermissionLevel),
		PermissionRaw:   permissionRaw,
		StartAt:         toNullTime(in.StartAt),
		EndAt:           toNullTime(in.EndAt),
		Status:          dao.DeviceShareStatusPending,
		CreatedBy:       in.OperatorUserID,
		Remark:          strings.TrimSpace(in.Remark),
	})
	if err != nil {
		return nil, err
	}
	if err := dao.InsertDeviceShareLog(ctx, tx, shareRow.ID, shareRow.FamilyID, shareRow.DeviceID, shareRow.DeviceSN, "create", "创建设备共享邀请", in.OperatorUserID, operatorRole); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetShareDetail(ctx, shareRow.ID, in.OperatorUserID)
}

func (s *Service) AcceptShareInvite(ctx context.Context, userID int64, inviteCode string) (*ShareView, error) {
	tx, err := s.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	share, err := dao.FindDeviceShareByInviteCode(ctx, tx, inviteCode, true)
	if err != nil {
		return nil, err
	}
	if share == nil || share.Status != dao.DeviceShareStatusPending {
		return nil, errorx.NewCodeError(errorx.CodeDeviceShareInvalid, "")
	}
	if share.SharedUserID != userID {
		return nil, errorx.NewCodeError(errorx.CodeDeviceShareInvalid, "邀请码与当前账号不匹配")
	}
	if share.EndAt.Valid && share.EndAt.Time.Before(time.Now()) {
		if err := dao.UpdateDeviceShareStatus(ctx, tx, share.ID, dao.DeviceShareStatusExpired); err != nil {
			return nil, err
		}
		return nil, errorx.NewCodeError(errorx.CodeDeviceShareExpired, "")
	}
	member, err := dao.FindFamilyMember(ctx, tx, share.FamilyID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil || member.Status != dao.FamilyMemberStatusActive {
		return nil, errorx.NewCodeError(errorx.CodeFamilyNotFound, "请先加入家庭后再接受共享")
	}
	if err := dao.UpdateDeviceShareAccepted(ctx, tx, share.ID); err != nil {
		return nil, err
	}
	if err := dao.InsertDeviceShareLog(ctx, tx, share.ID, share.FamilyID, share.DeviceID, share.DeviceSN, "accept", "接受设备共享邀请", userID, member.Role); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetShareDetail(ctx, share.ID, userID)
}

func (s *Service) RevokeShare(ctx context.Context, operatorUserID, shareID int64) error {
	tx, err := s.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	share, err := dao.FindDeviceShareByID(ctx, tx, shareID)
	if err != nil {
		return err
	}
	if share == nil {
		return errorx.NewCodeError(errorx.CodeDeviceShareInvalid, "")
	}
	operatorRole, err := s.resolveOperatorRoleForShare(ctx, tx, operatorUserID, share)
	if err != nil {
		return err
	}
	if err := dao.UpdateDeviceShareStatus(ctx, tx, shareID, dao.DeviceShareStatusRevoked); err != nil {
		return err
	}
	if err := dao.InsertDeviceShareLog(ctx, tx, shareID, share.FamilyID, share.DeviceID, share.DeviceSN, "revoke", "撤销设备共享", operatorUserID, operatorRole); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) QuitShare(ctx context.Context, userID, shareID int64) error {
	tx, err := s.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	share, err := dao.FindDeviceShareByID(ctx, tx, shareID)
	if err != nil {
		return err
	}
	if share == nil || share.SharedUserID != userID {
		return errorx.NewCodeError(errorx.CodeDeviceShareNoPermission, "")
	}
	member, err := dao.FindFamilyMember(ctx, tx, share.FamilyID, userID)
	if err != nil {
		return err
	}
	role := dao.FamilyRoleMember
	if member != nil {
		role = member.Role
	}
	if err := dao.UpdateDeviceShareStatus(ctx, tx, shareID, dao.DeviceShareStatusQuit); err != nil {
		return err
	}
	if err := dao.InsertDeviceShareLog(ctx, tx, shareID, share.FamilyID, share.DeviceID, share.DeviceSN, "quit", "主动退出设备共享", userID, role); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) ListSharesByOwner(ctx context.Context, userID int64) ([]ShareView, error) {
	tx, err := s.svcCtx.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := dao.ListSharesByOwner(ctx, tx, userID, "")
	if err != nil {
		return nil, err
	}
	return toShareViews(rows), nil
}

func (s *Service) ListSharesForReceiver(ctx context.Context, userID int64) ([]ShareView, error) {
	tx, err := s.svcCtx.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := dao.ListSharesForReceiver(ctx, tx, userID, "")
	if err != nil {
		return nil, err
	}
	return toShareViews(rows), nil
}

func (s *Service) GetShareDetail(ctx context.Context, shareID, requesterUserID int64) (*ShareView, error) {
	tx, err := s.svcCtx.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	row, err := dao.FindDeviceShareByID(ctx, tx, shareID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, errorx.NewCodeError(errorx.CodeDeviceShareInvalid, "")
	}
	if requesterUserID != row.OwnerUserID && requesterUserID != row.SharedUserID && requesterUserID != row.CreatedBy {
		member, err := dao.FindFamilyMember(ctx, tx, row.FamilyID, requesterUserID)
		if err != nil {
			return nil, err
		}
		if member == nil || (member.Role != dao.FamilyRoleOwner && member.Role != dao.FamilyRoleSuperAdmin) {
			return nil, errorx.NewCodeError(errorx.CodeDeviceShareNoPermission, "")
		}
	}
	view := toShareView(row)
	return &view, nil
}

func (s *Service) ExpireShares(ctx context.Context, limit int) (int, error) {
	tx, err := s.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := dao.ExpireDueShares(ctx, tx, time.Now(), limit)
	if err != nil {
		return 0, err
	}
	for _, row := range rows {
		if err := dao.InsertDeviceShareLog(ctx, tx, row.ID, row.FamilyID, row.DeviceID, row.DeviceSN, "expire", "共享到期自动失效", row.OwnerUserID, dao.FamilyRoleOwner); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(rows), nil
}

func (s *Service) RevokeSharesByDevice(ctx context.Context, ownerUserID, deviceID int64, reason string) (int, error) {
	tx, err := s.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := dao.RevokeSharesByDevice(ctx, tx, ownerUserID, deviceID, reason)
	if err != nil {
		return 0, err
	}
	for _, row := range rows {
		if err := dao.InsertDeviceShareLog(ctx, tx, row.ID, row.FamilyID, row.DeviceID, row.DeviceSN, "revoke", firstNonEmpty(reason, "设备解绑导致共享失效"), ownerUserID, dao.FamilyRoleOwner); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(rows), nil
}

func (s *Service) CanControlDevice(ctx context.Context, userID, deviceID int64) (*AccessDecision, error) {
	tx, err := s.svcCtx.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	ownerBind, err := dao.FindActiveBindByDeviceID(ctx, tx, deviceID)
	if err != nil {
		return nil, err
	}
	if ownerBind != nil && ownerBind.UserID == userID {
		return &AccessDecision{
			Allowed:         true,
			AccessMode:      "owner",
			Role:            dao.FamilyRoleOwner,
			PermissionLevel: dao.PermissionLevelFullControl,
			Permission:      map[string]any{"all": true},
			OwnerUserID:     ownerBind.UserID,
		}, nil
	}
	share, err := dao.FindActiveShareForDeviceUser(ctx, tx, deviceID, userID)
	if err != nil {
		return nil, err
	}
	if share == nil || share.Status != dao.DeviceShareStatusActive {
		return &AccessDecision{Allowed: false}, nil
	}
	member, err := dao.FindFamilyMember(ctx, tx, share.FamilyID, userID)
	if err != nil {
		return nil, err
	}
	if member == nil || member.Status != dao.FamilyMemberStatusActive {
		return &AccessDecision{Allowed: false}, nil
	}
	return &AccessDecision{
		Allowed:         true,
		AccessMode:      "shared",
		Role:            member.Role,
		PermissionLevel: share.PermissionLevel,
		Permission:      decodePermission(share.PermissionRaw),
		OwnerUserID:     share.OwnerUserID,
	}, nil
}

func (s *Service) resolveOperatorRoleForShare(ctx context.Context, tx *sql.Tx, operatorUserID int64, share *dao.DeviceShareViewRow) (string, error) {
	if operatorUserID == share.OwnerUserID {
		return dao.FamilyRoleOwner, nil
	}
	member, err := dao.FindFamilyMember(ctx, tx, share.FamilyID, operatorUserID)
	if err != nil {
		return "", err
	}
	if member == nil || member.Status != dao.FamilyMemberStatusActive {
		return "", errorx.NewCodeError(errorx.CodeDeviceShareNoPermission, "")
	}
	if member.Role != dao.FamilyRoleSuperAdmin {
		return "", errorx.NewCodeError(errorx.CodeDeviceShareNoPermission, "")
	}
	return member.Role, nil
}

func toShareViews(rows []*dao.DeviceShareViewRow) []ShareView {
	out := make([]ShareView, 0, len(rows))
	for _, row := range rows {
		out = append(out, toShareView(row))
	}
	return out
}

func toShareView(row *dao.DeviceShareViewRow) ShareView {
	view := ShareView{
		ID:              row.ID,
		FamilyID:        row.FamilyID,
		DeviceID:        row.DeviceID,
		DeviceSN:        row.DeviceSN,
		DeviceName:      row.DeviceName,
		OwnerUserID:     row.OwnerUserID,
		OwnerNickname:   row.OwnerNickname,
		SharedUserID:    row.SharedUserID,
		SharedNickname:  row.SharedNickname,
		TargetAccount:   row.TargetAccount,
		InviteCode:      row.InviteCode,
		ShareType:       row.ShareType,
		PermissionLevel: row.PermissionLevel,
		Permission:      decodePermission(row.PermissionRaw),
		Status:          row.Status,
		FamilyName:      row.FamilyName,
		CreatedAt:       row.CreatedAt,
	}
	if row.StartAt.Valid {
		t := row.StartAt.Time
		view.StartAt = &t
	}
	if row.EndAt.Valid {
		t := row.EndAt.Time
		view.EndAt = &t
	}
	if row.ConfirmedAt.Valid {
		t := row.ConfirmedAt.Time
		view.ConfirmedAt = &t
	}
	return view
}

func normalizeShareType(v string) string {
	switch strings.TrimSpace(v) {
	case dao.ShareTypeTemporary:
		return dao.ShareTypeTemporary
	case dao.ShareTypeTimeWindow:
		return dao.ShareTypeTimeWindow
	default:
		return dao.ShareTypePermanent
	}
}

func normalizePermissionLevel(v string) string {
	switch strings.TrimSpace(v) {
	case dao.PermissionLevelFullControl:
		return dao.PermissionLevelFullControl
	case dao.PermissionLevelPartial:
		return dao.PermissionLevelPartial
	default:
		return dao.PermissionLevelViewOnly
	}
}

func normalizePermissionPayload(level string, raw map[string]any) map[string]any {
	if raw == nil {
		raw = map[string]any{}
	}
	switch normalizePermissionLevel(level) {
	case dao.PermissionLevelPartial:
		if _, ok := raw["allowed_actions"]; !ok {
			raw["allowed_actions"] = []string{}
		}
	case dao.PermissionLevelFullControl:
		raw["all"] = true
	default:
		raw["read_only"] = true
	}
	return raw
}

func decodePermission(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func shareInviteCode() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "SHR" + strings.ToUpper(hex.EncodeToString(buf)), nil
}
