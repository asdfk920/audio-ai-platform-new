package familysvc

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
)

const defaultFamilyInviteTTL = 72 * time.Hour

type Service struct {
	svcCtx *svc.ServiceContext
}

type FamilyView struct {
	ID          int64
	Name        string
	OwnerUserID int64
	CurrentRole string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	MemberCount int
}

type MemberView struct {
	ID        int64
	UserID    int64
	Role      string
	Status    int16
	JoinedAt  time.Time
	InvitedBy int64
	Nickname  string
	Email     string
	Mobile    string
}

type CreateFamilyInput struct {
	OwnerUserID int64
	Name        string
}

type InviteFamilyMemberInput struct {
	OperatorUserID int64
	TargetUserID   int64
	TargetAccount  string
	Role           string
	Remark         string
	ExpiresAt      *time.Time
}

type FamilyInviteView struct {
	FamilyID      int64
	InviteCode    string
	TargetUserID  int64
	TargetAccount string
	Role          string
	Status        string
	ExpiresAt     *time.Time
}

func New(svcCtx *svc.ServiceContext) *Service {
	return &Service{svcCtx: svcCtx}
}

func (s *Service) EnsureFamilyForOwner(ctx context.Context, ownerUserID int64, fallbackName string) (*FamilyView, error) {
	tx, err := s.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	family, member, err := dao.FindCurrentFamilyByUser(ctx, tx, ownerUserID)
	if err != nil {
		return nil, err
	}
	if family != nil && member != nil && member.Role == dao.FamilyRoleOwner {
		view, err := s.buildFamilyView(ctx, tx, family, member)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return view, nil
	}

	created, err := s.createFamilyTx(ctx, tx, CreateFamilyInput{OwnerUserID: ownerUserID, Name: fallbackName})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) CreateFamily(ctx context.Context, in CreateFamilyInput) (*FamilyView, error) {
	tx, err := s.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	view, err := s.createFamilyTx(ctx, tx, in)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return view, nil
}

func (s *Service) createFamilyTx(ctx context.Context, tx *sql.Tx, in CreateFamilyInput) (*FamilyView, error) {
	if in.OwnerUserID <= 0 {
		return nil, errorx.NewCodeError(errorx.CodeTokenInvalid, "登录已过期或无效，请重新登录")
	}
	user, err := dao.FindUserByID(ctx, tx, in.OwnerUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errorx.NewCodeError(errorx.CodeUserNotFound, "")
		}
		return nil, err
	}
	if user.Status != 1 {
		return nil, errorx.NewCodeError(errorx.CodeUserAccountDisabled, "")
	}
	existing, err := dao.FindFamilyByOwner(ctx, tx, in.OwnerUserID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errorx.NewCodeError(errorx.CodeFamilyAlreadyExists, "")
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		if user.Nickname != "" {
			name = user.Nickname + "的家庭"
		} else {
			name = "我的家庭"
		}
	}

	family, err := dao.InsertFamily(ctx, tx, in.OwnerUserID, name)
	if err != nil {
		return nil, err
	}
	member, err := dao.InsertFamilyMember(ctx, tx, family.ID, in.OwnerUserID, in.OwnerUserID, dao.FamilyRoleOwner)
	if err != nil {
		return nil, err
	}
	return s.buildFamilyView(ctx, tx, family, member)
}

func (s *Service) GetCurrentFamily(ctx context.Context, userID int64) (*FamilyView, error) {
	tx, err := s.svcCtx.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	family, member, err := dao.FindCurrentFamilyByUser(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	if family == nil || member == nil {
		return nil, errorx.NewCodeError(errorx.CodeFamilyNotFound, "")
	}
	view, err := s.buildFamilyView(ctx, tx, family, member)
	if err != nil {
		return nil, err
	}
	return view, nil
}

func (s *Service) ListFamilyMembers(ctx context.Context, userID int64) ([]MemberView, *FamilyView, error) {
	tx, err := s.svcCtx.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	family, member, err := dao.FindCurrentFamilyByUser(ctx, tx, userID)
	if err != nil {
		return nil, nil, err
	}
	if family == nil || member == nil {
		return nil, nil, errorx.NewCodeError(errorx.CodeFamilyNotFound, "")
	}
	view, err := s.buildFamilyView(ctx, tx, family, member)
	if err != nil {
		return nil, nil, err
	}
	rows, err := dao.ListFamilyMembers(ctx, tx, family.ID)
	if err != nil {
		return nil, nil, err
	}
	list := make([]MemberView, 0, len(rows))
	for _, item := range rows {
		user, err := dao.FindUserByID(ctx, tx, item.UserID)
		if err != nil {
			return nil, nil, err
		}
		mv := MemberView{
			ID:        item.ID,
			UserID:    item.UserID,
			Role:      item.Role,
			Status:    item.Status,
			JoinedAt:  item.JoinedAt,
			InvitedBy: item.InvitedBy,
			Nickname:  user.Nickname,
		}
		if user.Email.Valid {
			mv.Email = user.Email.String
		}
		if user.Mobile.Valid {
			mv.Mobile = user.Mobile.String
		}
		list = append(list, mv)
	}
	return list, view, nil
}

func (s *Service) InviteFamilyMember(ctx context.Context, in InviteFamilyMemberInput) (*FamilyInviteView, error) {
	tx, err := s.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	family, operatorMember, err := dao.FindCurrentFamilyByUser(ctx, tx, in.OperatorUserID)
	if err != nil {
		return nil, err
	}
	if family == nil || operatorMember == nil {
		return nil, errorx.NewCodeError(errorx.CodeFamilyNotFound, "")
	}
	if operatorMember.Role != dao.FamilyRoleOwner && operatorMember.Role != dao.FamilyRoleSuperAdmin {
		return nil, errorx.NewCodeError(errorx.CodeFamilyNoPermission, "")
	}

	target, err := dao.FindUserByAccount(ctx, tx, strings.TrimSpace(in.TargetAccount), in.TargetUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errorx.NewCodeError(errorx.CodeUserNotFound, "")
		}
		return nil, err
	}
	if target.ID == in.OperatorUserID {
		return nil, errorx.NewCodeError(errorx.CodeInvalidParam, "不能邀请自己加入家庭")
	}
	existing, err := dao.FindFamilyMember(ctx, tx, family.ID, target.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status == dao.FamilyMemberStatusActive {
		return nil, errorx.NewCodeError(errorx.CodeFamilyMemberExists, "")
	}

	code, err := randomInviteCode("FAM")
	if err != nil {
		return nil, err
	}
	exp := in.ExpiresAt
	if exp == nil {
		t := time.Now().Add(defaultFamilyInviteTTL)
		exp = &t
	}
	targetID := target.ID
	invite, err := dao.InsertFamilyInvite(ctx, tx, family.ID, code, &targetID, firstNonEmpty(in.TargetAccount, target.Mobile.String, target.Email.String), normalizedFamilyRole(in.Role), exp, in.OperatorUserID, in.Remark)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return toFamilyInviteView(invite), nil
}

func (s *Service) AcceptFamilyInvite(ctx context.Context, userID int64, inviteCode string) (*FamilyView, error) {
	tx, err := s.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	invite, err := dao.FindFamilyInviteByCode(ctx, tx, inviteCode, true)
	if err != nil {
		return nil, err
	}
	if invite == nil || invite.Status != dao.FamilyInviteStatusPending {
		return nil, errorx.NewCodeError(errorx.CodeFamilyInviteInvalid, "")
	}
	if invite.ExpiresAt.Valid && invite.ExpiresAt.Time.Before(time.Now()) {
		if err := dao.UpdateFamilyInviteStatus(ctx, tx, invite.ID, dao.FamilyInviteStatusExpired); err != nil {
			return nil, err
		}
		return nil, errorx.NewCodeError(errorx.CodeFamilyInviteInvalid, "")
	}
	if invite.TargetUserID.Valid && invite.TargetUserID.Int64 != userID {
		return nil, errorx.NewCodeError(errorx.CodeFamilyInviteInvalid, "邀请码与当前账号不匹配")
	}
	member, err := dao.FindFamilyMember(ctx, tx, invite.FamilyID, userID)
	if err != nil {
		return nil, err
	}
	if member != nil && member.Status == dao.FamilyMemberStatusActive {
		return nil, errorx.NewCodeError(errorx.CodeFamilyMemberExists, "")
	}
	if member != nil {
		if err := dao.ReactivateFamilyMember(ctx, tx, member.ID, invite.CreatedBy, invite.Role); err != nil {
			return nil, err
		}
	} else {
		if _, err := dao.InsertFamilyMember(ctx, tx, invite.FamilyID, userID, invite.CreatedBy, invite.Role); err != nil {
			return nil, err
		}
	}
	if err := dao.UpdateFamilyInviteStatus(ctx, tx, invite.ID, dao.FamilyInviteStatusAccepted); err != nil {
		return nil, err
	}
	family, currentMember, err := dao.FindCurrentFamilyByUser(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	view, err := s.buildFamilyView(ctx, tx, family, currentMember)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return view, nil
}

func (s *Service) RemoveFamilyMember(ctx context.Context, operatorUserID, targetUserID int64) error {
	tx, err := s.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	family, operator, err := dao.FindCurrentFamilyByUser(ctx, tx, operatorUserID)
	if err != nil {
		return err
	}
	if family == nil || operator == nil {
		return errorx.NewCodeError(errorx.CodeFamilyNotFound, "")
	}
	if operator.Role != dao.FamilyRoleOwner && operator.Role != dao.FamilyRoleSuperAdmin {
		return errorx.NewCodeError(errorx.CodeFamilyNoPermission, "")
	}
	target, err := dao.FindFamilyMember(ctx, tx, family.ID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil || target.Status != dao.FamilyMemberStatusActive {
		return errorx.NewCodeError(errorx.CodeUserNotFound, "目标成员不存在")
	}
	if target.Role == dao.FamilyRoleOwner {
		return errorx.NewCodeError(errorx.CodeFamilyNoPermission, "不能移除家庭创建者")
	}
	if operator.Role == dao.FamilyRoleSuperAdmin && target.Role == dao.FamilyRoleSuperAdmin {
		return errorx.NewCodeError(errorx.CodeFamilyNoPermission, "超级管理员不能移除其他超级管理员")
	}
	if err := dao.RemoveFamilyMember(ctx, tx, target.ID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Service) ChangeMemberRole(ctx context.Context, operatorUserID, targetUserID int64, role string) error {
	tx, err := s.svcCtx.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	family, operator, err := dao.FindCurrentFamilyByUser(ctx, tx, operatorUserID)
	if err != nil {
		return err
	}
	if family == nil || operator == nil {
		return errorx.NewCodeError(errorx.CodeFamilyNotFound, "")
	}
	if operator.Role != dao.FamilyRoleOwner {
		return errorx.NewCodeError(errorx.CodeFamilyNoPermission, "只有家庭创建者可以调整成员角色")
	}
	target, err := dao.FindFamilyMember(ctx, tx, family.ID, targetUserID)
	if err != nil {
		return err
	}
	if target == nil || target.Status != dao.FamilyMemberStatusActive {
		return errorx.NewCodeError(errorx.CodeUserNotFound, "目标成员不存在")
	}
	if target.Role == dao.FamilyRoleOwner {
		return errorx.NewCodeError(errorx.CodeFamilyNoPermission, "不能修改创建者角色")
	}
	if err := dao.UpdateFamilyMemberRole(ctx, tx, target.ID, normalizedFamilyRole(role)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) buildFamilyView(ctx context.Context, tx *sql.Tx, family *dao.FamilyRow, member *dao.FamilyMemberRow) (*FamilyView, error) {
	rows, err := dao.ListFamilyMembers(ctx, tx, family.ID)
	if err != nil {
		return nil, err
	}
	return &FamilyView{
		ID:          family.ID,
		Name:        family.Name,
		OwnerUserID: family.OwnerUserID,
		CurrentRole: member.Role,
		CreatedAt:   family.CreatedAt,
		UpdatedAt:   family.UpdatedAt,
		MemberCount: len(rows),
	}, nil
}

func normalizedFamilyRole(role string) string {
	switch strings.TrimSpace(role) {
	case dao.FamilyRoleSuperAdmin:
		return dao.FamilyRoleSuperAdmin
	default:
		return dao.FamilyRoleMember
	}
}

func randomInviteCode(prefix string) (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("rand invite code: %w", err)
	}
	return strings.ToUpper(prefix) + hex.EncodeToString(buf), nil
}

func toFamilyInviteView(row *dao.FamilyInviteRow) *FamilyInviteView {
	view := &FamilyInviteView{
		FamilyID:      row.FamilyID,
		InviteCode:    row.InviteCode,
		TargetAccount: row.TargetAccount,
		Role:          row.Role,
		Status:        row.Status,
	}
	if row.TargetUserID.Valid {
		view.TargetUserID = row.TargetUserID.Int64
	}
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time
		view.ExpiresAt = &t
	}
	return view
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
