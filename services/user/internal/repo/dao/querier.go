package dao

import (
	"context"
	"database/sql"
)

// RowQuerier 兼容 *sql.DB 与 *sql.Tx，便于注册等事务内与直连共用查询。
type RowQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
