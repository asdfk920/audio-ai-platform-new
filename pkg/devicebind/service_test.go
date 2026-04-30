package devicebind

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jacklau/audio-ai-platform/common/errorx"
)

func resetColumnCaches() {
	deviceColsMu.Lock()
	deviceCols = nil
	deviceColsMu.Unlock()

	profileColsMu.Lock()
	profileCols = nil
	profileColsMu.Unlock()
}

func TestBindUserDevice_DeviceNotFound(t *testing.T) {
	resetColumnCaches()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, status
		FROM public.users
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1`)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(int64(1), int16(1)))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, sn, status
		FROM public.device
		WHERE sn = $1
		LIMIT 1`)).
		WithArgs("SN404").
		WillReturnError(errorx.NewDefaultError(errorx.CodeDeviceNotFound))
	mock.ExpectRollback()

	_, err = BindUserDevice(context.Background(), db, 1, "sn404", Options{MaxDeviceBinds: 10})
	if err == nil {
		t.Fatal("expected error")
	}
	if errorx.CodeOf(err) != errorx.CodeDatabaseError && errorx.CodeOf(err) != errorx.CodeDeviceNotFound {
		t.Fatalf("unexpected code: %d err=%v", errorx.CodeOf(err), err)
	}
}

func TestBindUserDevice_AlreadyBoundByOtherUser(t *testing.T) {
	resetColumnCaches()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	boundAt := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, status
		FROM public.users
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1`)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(int64(1), int16(1)))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, sn, status
		FROM public.device
		WHERE sn = $1
		LIMIT 1`)).
		WithArgs("SN123").
		WillReturnRows(sqlmock.NewRows([]string{"id", "sn", "status"}).AddRow(int64(9), "SN123", int16(1)))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, device_id, sn, COALESCE(alias,''),
		       status, bound_at, unbound_at
		FROM public.user_device_bind
		WHERE device_id = $1 AND status = 1
		LIMIT 1`)).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "device_id", "sn", "alias", "status", "bound_at", "unbound_at"}).
			AddRow(int64(11), int64(2), int64(9), "SN123", "MyDevice", int16(1), boundAt, nil))
	mock.ExpectRollback()

	_, err = BindUserDevice(context.Background(), db, 1, "SN123", Options{MaxDeviceBinds: 10})
	if err == nil {
		t.Fatal("expected error")
	}
	if errorx.CodeOf(err) != errorx.CodeDeviceBoundByOther {
		t.Fatalf("unexpected code: %d err=%v", errorx.CodeOf(err), err)
	}
}

func TestBindUserDevice_IdempotentForSameUser(t *testing.T) {
	resetColumnCaches()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	boundAt := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, status
		FROM public.users
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1`)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(int64(1), int16(1)))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, sn, status
		FROM public.device
		WHERE sn = $1
		LIMIT 1`)).
		WithArgs("SN123").
		WillReturnRows(sqlmock.NewRows([]string{"id", "sn", "status"}).AddRow(int64(9), "SN123", int16(1)))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, device_id, sn, COALESCE(alias,''),
		       status, bound_at, unbound_at
		FROM public.user_device_bind
		WHERE device_id = $1 AND status = 1
		LIMIT 1`)).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "device_id", "sn", "alias", "status", "bound_at", "unbound_at"}).
			AddRow(int64(11), int64(1), int64(9), "SN123", "Bedroom", int16(1), boundAt, nil))
	mock.ExpectRollback()

	result, err := BindUserDevice(context.Background(), db, 1, "SN123", Options{MaxDeviceBinds: 10})
	if err != nil {
		t.Fatal(err)
	}
	if result.DeviceName != "Bedroom" {
		t.Fatalf("unexpected device name: %q", result.DeviceName)
	}
	if result.DeviceSN != "SN123" {
		t.Fatalf("unexpected sn: %q", result.DeviceSN)
	}
}

func TestBindUserDevice_QuotaExceeded(t *testing.T) {
	resetColumnCaches()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, status
		FROM public.users
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1`)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(int64(1), int16(1)))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, sn, status
		FROM public.device
		WHERE sn = $1
		LIMIT 1`)).
		WithArgs("SN123").
		WillReturnRows(sqlmock.NewRows([]string{"id", "sn", "status"}).AddRow(int64(9), "SN123", int16(1)))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, device_id, sn, COALESCE(alias,''),
		       status, bound_at, unbound_at
		FROM public.user_device_bind
		WHERE device_id = $1 AND status = 1
		LIMIT 1`)).
		WithArgs(int64(9)).
		WillReturnError(sqlmock.ErrCancelled)
	mock.ExpectRollback()

	_, err = BindUserDevice(context.Background(), db, 1, "SN123", Options{MaxDeviceBinds: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBindUserDevice_CreateNewBindingSuccess(t *testing.T) {
	resetColumnCaches()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, status
		FROM public.users
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1`)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}).AddRow(int64(1), int16(1)))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, sn, status
		FROM public.device
		WHERE sn = $1
		LIMIT 1`)).
		WithArgs("SN123").
		WillReturnRows(sqlmock.NewRows([]string{"id", "sn", "status"}).AddRow(int64(9), "SN123", int16(1)))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, device_id, sn, COALESCE(alias,''),
		       status, bound_at, unbound_at
		FROM public.user_device_bind
		WHERE device_id = $1 AND status = 1
		LIMIT 1`)).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "device_id", "sn", "alias", "status", "bound_at", "unbound_at"}))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(*)
		FROM public.user_device_bind
		WHERE user_id = $1 AND status = 1
	`)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, user_id, device_id, sn, COALESCE(alias,''),
		       status, bound_at, unbound_at
		FROM public.user_device_bind
		WHERE user_id = $1 AND device_id = $2
		LIMIT 1
	`)).
		WithArgs(int64(1), int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "device_id", "sn", "alias", "status", "bound_at", "unbound_at"}))
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO public.user_device_bind
		  (user_id, device_id, sn, alias, is_default, bind_type, status, bound_at)
		VALUES ($1, $2, $3, $4, 0, 1, 1, CURRENT_TIMESTAMP)
	`)).
		WithArgs(int64(1), int64(9), "SN123", "SN123").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'device'`)).
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).
			AddRow("bound_user_id").
			AddRow("bound_at").
			AddRow("bind_status").
			AddRow("updated_at"))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE public.device SET bound_user_id = $1, bound_at = $2, bind_status = 1, updated_at = CURRENT_TIMESTAMP WHERE id = $3`)).
		WithArgs(int64(1), sqlmock.AnyArg(), int64(9)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'user_profile'`)).
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).
			AddRow("device_count").
			AddRow("last_bind_time").
			AddRow("updated_at"))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE public.user_profile SET device_count = COALESCE(device_count, 0) + 1, last_bind_time = $1, updated_at = CURRENT_TIMESTAMP WHERE user_id = $2`)).
		WithArgs(sqlmock.AnyArg(), int64(1)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO public.user_device_bind_log
		  (user_id, device_id, sn, operator, action, action_time)
		VALUES ($1, $2, $3, $4, $5, $6)
	`)).
		WithArgs(int64(1), int64(9), "SN123", "", "bind", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := BindUserDevice(context.Background(), db, 1, "sn123", Options{MaxDeviceBinds: 10})
	if err != nil {
		t.Fatal(err)
	}
	if result.UserID != 1 || result.DeviceID != 9 || result.DeviceSN != "SN123" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
