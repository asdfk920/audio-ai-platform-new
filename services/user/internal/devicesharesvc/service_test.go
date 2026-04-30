package devicesharesvc

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jacklau/audio-ai-platform/common/errorx"
	"github.com/jacklau/audio-ai-platform/services/user/internal/repo/dao"
	"github.com/jacklau/audio-ai-platform/services/user/internal/svc"
)

func TestNormalizePermissionPayload(t *testing.T) {
	t.Run("full control", func(t *testing.T) {
		got := normalizePermissionPayload(dao.PermissionLevelFullControl, map[string]any{})
		if got["all"] != true {
			t.Fatalf("expected all=true, got %#v", got)
		}
	})

	t.Run("partial control default actions", func(t *testing.T) {
		got := normalizePermissionPayload(dao.PermissionLevelPartial, map[string]any{})
		actions, ok := got["allowed_actions"].([]string)
		if !ok {
			t.Fatalf("expected []string allowed_actions, got %#v", got["allowed_actions"])
		}
		if len(actions) != 0 {
			t.Fatalf("expected empty allowed_actions, got %#v", actions)
		}
	})

	t.Run("view only", func(t *testing.T) {
		got := normalizePermissionPayload("unknown", map[string]any{})
		if got["read_only"] != true {
			t.Fatalf("expected read_only=true, got %#v", got)
		}
	})
}

func TestCanControlDeviceOwner(t *testing.T) {
	svc, mock, cleanup := newMockDeviceShareService(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, device_id, sn, COALESCE(alias,''),
		       status, bound_at, unbound_at
		FROM public.user_device_bind
		WHERE device_id = $1 AND status = 1
		LIMIT 1
	`)).
		WithArgs(int64(88)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "device_id", "sn", "alias", "status", "bound_at", "unbound_at",
		}).AddRow(int64(1), int64(1001), int64(88), "SN-001", "owner-device", int16(1), now, nil))
	mock.ExpectRollback()

	got, err := svc.CanControlDevice(context.Background(), 1001, 88)
	if err != nil {
		t.Fatalf("CanControlDevice error: %v", err)
	}
	if !got.Allowed || got.AccessMode != "owner" || got.Role != dao.FamilyRoleOwner {
		t.Fatalf("unexpected owner decision: %#v", got)
	}
	if got.PermissionLevel != dao.PermissionLevelFullControl {
		t.Fatalf("unexpected permission level: %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations: %v", err)
	}
}

func TestCanControlDeviceSharedMember(t *testing.T) {
	svc, mock, cleanup := newMockDeviceShareService(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, device_id, sn, COALESCE(alias,''),
		       status, bound_at, unbound_at
		FROM public.user_device_bind
		WHERE device_id = $1 AND status = 1
		LIMIT 1
	`)).
		WithArgs(int64(88)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "device_id", "sn", "alias", "status", "bound_at", "unbound_at",
		}))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, family_id, device_id, device_sn, device_name, owner_user_id, shared_user_id, target_account, invite_code,
		       share_type, permission_level, permission_payload, start_at, end_at, status, confirmed_at, revoked_at, created_by, remark, created_at, updated_at
		FROM public.user_device_share
		WHERE device_id = $1 AND shared_user_id = $2
		  AND status IN ($3, $4)
		ORDER BY id DESC
		LIMIT 1
	`)).
		WithArgs(int64(88), int64(2002), dao.DeviceShareStatusPending, dao.DeviceShareStatusActive).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "family_id", "device_id", "device_sn", "device_name", "owner_user_id", "shared_user_id", "target_account", "invite_code",
			"share_type", "permission_level", "permission_payload", "start_at", "end_at", "status", "confirmed_at", "revoked_at", "created_by", "remark", "created_at", "updated_at",
		}).AddRow(
			int64(10), int64(9), int64(88), "SN-002", "shared-device", int64(1001), int64(2002), "13800138000", "SHR001",
			dao.ShareTypeTemporary, dao.PermissionLevelPartial, []byte(`{"allowed_actions":["power"]}`), nil, nil, dao.DeviceShareStatusActive, now, nil, int64(1001), "", now, now,
		))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, family_id, user_id, role, status, joined_at, invited_by, created_at, updated_at
		FROM public.user_family_member
		WHERE family_id = $1 AND user_id = $2
		ORDER BY id DESC
		LIMIT 1
	`)).
		WithArgs(int64(9), int64(2002)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "family_id", "user_id", "role", "status", "joined_at", "invited_by", "created_at", "updated_at",
		}).AddRow(int64(7), int64(9), int64(2002), dao.FamilyRoleMember, dao.FamilyMemberStatusActive, now, int64(1001), now, now))
	mock.ExpectRollback()

	got, err := svc.CanControlDevice(context.Background(), 2002, 88)
	if err != nil {
		t.Fatalf("CanControlDevice error: %v", err)
	}
	if !got.Allowed || got.AccessMode != "shared" || got.Role != dao.FamilyRoleMember {
		t.Fatalf("unexpected shared decision: %#v", got)
	}
	if got.PermissionLevel != dao.PermissionLevelPartial {
		t.Fatalf("unexpected permission level: %#v", got)
	}
	if got.Permission["allowed_actions"] == nil {
		t.Fatalf("expected allowed_actions in permission: %#v", got.Permission)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations: %v", err)
	}
}

func TestCreateShareInviteRejectsMemberReshare(t *testing.T) {
	svc, mock, cleanup := newMockDeviceShareService(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, email, mobile, COALESCE(nickname, ''), COALESCE(avatar, ''), status, created_at
		FROM public.users
		WHERE deleted_at IS NULL
		  AND (mobile = $1 OR email = $1)
		ORDER BY id ASC
		LIMIT 1
	`)).
		WithArgs("13800138001").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "email", "mobile", "nickname", "avatar", "status", "created_at",
		}).AddRow(int64(3003), "target@example.com", "13800138001", "target", "", int16(1), now))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, device_id, sn, COALESCE(alias,''),
		       status, bound_at, unbound_at
		FROM public.user_device_bind
		WHERE user_id = $1 AND sn = $2 AND status = 1
		LIMIT 1
	`)).
		WithArgs(int64(2002), "SN-003").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "device_id", "sn", "alias", "status", "bound_at", "unbound_at",
		}))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT f.id, f.owner_user_id, f.name, f.status, f.created_at, f.updated_at,
		       m.id, m.family_id, m.user_id, m.role, m.status, m.joined_at, m.invited_by, m.created_at, m.updated_at
		FROM public.user_family_member m
		JOIN public.user_family f ON f.id = m.family_id
		WHERE m.user_id = $1
		  AND m.status = $2
		  AND f.status = $3
		ORDER BY CASE WHEN m.role = 'owner' THEN 0 ELSE 1 END, m.id ASC
		LIMIT 1
	`)).
		WithArgs(int64(2002), dao.FamilyMemberStatusActive, dao.FamilyStatusActive).
		WillReturnRows(sqlmock.NewRows([]string{
			"family_id", "owner_user_id", "family_name", "family_status", "family_created_at", "family_updated_at",
			"member_id", "member_family_id", "member_user_id", "member_role", "member_status", "member_joined_at", "member_invited_by", "member_created_at", "member_updated_at",
		}).AddRow(
			int64(9), int64(1001), "family-a", dao.FamilyStatusActive, now, now,
			int64(7), int64(9), int64(2002), dao.FamilyRoleMember, dao.FamilyMemberStatusActive, now, int64(1001), now, now,
		))
	mock.ExpectRollback()

	_, err := svc.CreateShareInvite(context.Background(), CreateShareInviteInput{
		OperatorUserID: 2002,
		DeviceSN:       "SN-003",
		TargetAccount:  "13800138001",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	ce, ok := err.(*errorx.CodeError)
	if !ok || ce.GetCode() != errorx.CodeDeviceShareForbiddenReshare {
		t.Fatalf("unexpected error: %#v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations: %v", err)
	}
}

func newMockDeviceShareService(t *testing.T) (*Service, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	svcCtx := &svc.ServiceContext{DB: db}
	return New(svcCtx), mock, func() { _ = db.Close() }
}

var _ = sql.LevelDefault
