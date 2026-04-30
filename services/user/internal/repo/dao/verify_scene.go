package dao

import (
	"context"
)

// EmailRegistered 是否存在使用该邮箱的用户（用于发码场景的数据库侧判断）。
// 不按 deleted_at 过滤：软删账号仍视为占用，避免同邮箱重复注册；与「活跃用户」查询语义不同。
func EmailRegistered(ctx context.Context, q RowQuerier, email string) (bool, error) {
	var ok bool
	// 不依赖 deleted_at：避免未执行完整迁移的库因缺列报错；软删账号仍视为已占用。
	err := q.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND email IS NOT NULL AND email <> '')`,
		email,
	).Scan(&ok)
	return ok, err
}

// MobileRegistered 是否存在使用该手机号的用户。
func MobileRegistered(ctx context.Context, q RowQuerier, mobile string) (bool, error) {
	var ok bool
	err := q.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE mobile = $1 AND mobile IS NOT NULL AND mobile <> '')`,
		mobile,
	).Scan(&ok)
	return ok, err
}
