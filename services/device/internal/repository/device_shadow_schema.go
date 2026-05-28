package repository

import (
	"context"
	"database/sql"
	_ "embed"
	"sync"
)

//go:embed schema_device_shadow.sql
var deviceShadowSchemaSQL string

var ensureShadowSchemaOnce sync.Once
var ensureShadowSchemaErr error

// EnsureDeviceShadowSchema 幂等创建/补齐 device_shadow 表（首次注册前调用）
func EnsureDeviceShadowSchema(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return nil
	}
	ensureShadowSchemaOnce.Do(func() {
		_, ensureShadowSchemaErr = db.ExecContext(ctx, deviceShadowSchemaSQL)
	})
	return ensureShadowSchemaErr
}

// EnsureSchema 在 DeviceShadowRepo 上调用 EnsureDeviceShadowSchema
func (r *DeviceShadowRepo) EnsureSchema(ctx context.Context) error {
	return EnsureDeviceShadowSchema(ctx, r.db)
}
