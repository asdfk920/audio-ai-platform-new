package dao

import (
	"context"
	"strings"
	"sync"
)

// usersTableColumns 反映当前库 users 表已存在的列（用于兼容未跑全量迁移的库）。
type usersTableColumns struct {
	Avatar                   bool
	PasswordAlgo             bool
	RegisterIP               bool
	RegisterChannel          bool
	InviteCode               bool
	DeviceID                 bool
	DeletedAt                bool
	CancellationCoolingUntil bool
	AccountCancelledAt       bool
}

var (
	usersColMu sync.RWMutex
	usersCol   *usersTableColumns
)

func getUsersTableColumns(ctx context.Context, q RowQuerier) (*usersTableColumns, error) {
	usersColMu.RLock()
	cached := usersCol
	usersColMu.RUnlock()
	if cached != nil {
		return cached, nil
	}

	usersColMu.Lock()
	defer usersColMu.Unlock()
	if usersCol != nil {
		return usersCol, nil
	}

	rows, err := q.QueryContext(ctx, `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'users'`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	m := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		m[strings.ToLower(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	col := &usersTableColumns{
		Avatar:                   mapHas(m, "avatar"),
		PasswordAlgo:             mapHas(m, "password_algo"),
		RegisterIP:               mapHas(m, "register_ip"),
		RegisterChannel:          mapHas(m, "register_channel"),
		InviteCode:               mapHas(m, "invite_code"),
		DeviceID:                 mapHas(m, "device_id"),
		DeletedAt:                mapHas(m, "deleted_at"),
		CancellationCoolingUntil: mapHas(m, "cancellation_cooling_until"),
		AccountCancelledAt:       mapHas(m, "account_cancelled_at"),
	}
	usersCol = col
	return col, nil
}

func mapHas(m map[string]struct{}, k string) bool {
	_, ok := m[k]
	return ok
}

// userLoginSelectWithPassword 登录用 SELECT：缺列时用 NULL 占位，保证 Scan 字段数一致。
func userLoginSelectWithPassword(cols *usersTableColumns) string {
	canc := "NULL::timestamp with time zone"
	if cols.CancellationCoolingUntil {
		canc = "cancellation_cooling_until"
	}
	acc := "NULL::timestamp with time zone"
	if cols.AccountCancelledAt {
		acc = "account_cancelled_at"
	}
	return "SELECT id, email, mobile, nickname, avatar, status, password, salt, " + canc + ", " + acc + " FROM users "
}

func userWhereActiveByField(cols *usersTableColumns, field string) string {
	w := field + " = $1"
	if cols.DeletedAt {
		w += " AND deleted_at IS NULL"
	}
	return w
}

func userWhereActiveByID(cols *usersTableColumns) string {
	w := "id = $1"
	if cols.DeletedAt {
		w += " AND deleted_at IS NULL"
	}
	return w
}

// ResetUsersTableColumnsCache 仅测试用：模拟迁移前后列变化。
func ResetUsersTableColumnsCache() {
	usersColMu.Lock()
	usersCol = nil
	usersColMu.Unlock()
}
