// Package uploadsysconfig merges OSS / upload driver settings from go-admin sys_config（配置中心）。
// 与 admin 约定的 config_key：upload_storage_driver、upload_storage_oss_*、upload_storage_public_base_url。
package uploadsysconfig

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jacklau/audio-ai-platform/services/content/internal/config"

	_ "github.com/lib/pq" // Postgres driver for one-shot startup query.
)

var uploadSysConfigKeys = []string{
	"upload_storage_driver",
	"upload_storage_oss_endpoint",
	"upload_storage_oss_bucket",
	"upload_storage_public_base_url",
	"upload_storage_oss_access_key_id",
	"upload_storage_oss_access_key_secret",
}

// MergeFromPostgreSQL fills cfg.Storage from public.sys_config (non-empty values only).
// Call after YAML load, before applyStorageEnvOverrides so env still wins.
func MergeFromPostgreSQL(ctx context.Context, dsn string, cfg *config.Config) error {
	if strings.TrimSpace(dsn) == "" || cfg == nil {
		return nil
	}

	db, err := sql.Open("postgres", strings.TrimSpace(dsn))
	if err != nil {
		return fmt.Errorf("uploadsysconfig: open db: %w", err)
	}
	defer db.Close()

	qctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	qs := `
SELECT config_key, config_value FROM public.sys_config
WHERE deleted_at IS NULL AND config_key IN ($1,$2,$3,$4,$5,$6)`

	r, err := db.QueryContext(qctx, qs,
		uploadSysConfigKeys[0], uploadSysConfigKeys[1], uploadSysConfigKeys[2],
		uploadSysConfigKeys[3], uploadSysConfigKeys[4], uploadSysConfigKeys[5],
	)
	if err != nil {
		return fmt.Errorf("uploadsysconfig: query sys_config: %w", err)
	}
	defer r.Close()

	rows := map[string]string{}
	for r.Next() {
		var k, v string
		if scanErr := r.Scan(&k, &v); scanErr != nil {
			return fmt.Errorf("uploadsysconfig: scan: %w", scanErr)
		}
		k = strings.TrimSpace(k)
		if k != "" {
			rows[k] = strings.TrimSpace(v)
		}
	}
	if err := r.Err(); err != nil {
		return fmt.Errorf("uploadsysconfig: rows: %w", err)
	}

	if v := rows["upload_storage_driver"]; v != "" {
		cfg.Storage.Driver = strings.ToLower(strings.TrimSpace(v))
	}
	if v := rows["upload_storage_oss_endpoint"]; v != "" {
		cfg.Storage.Endpoint = normalizeOSSHost(v)
	}
	if v := rows["upload_storage_oss_bucket"]; v != "" {
		cfg.Storage.Bucket = v
	}
	if v := rows["upload_storage_public_base_url"]; v != "" {
		cfg.Storage.CdnBaseUrl = strings.TrimRight(v, "/")
	}
	if v := rows["upload_storage_oss_access_key_id"]; v != "" {
		cfg.Storage.AccessKey = v
	}
	if v := rows["upload_storage_oss_access_key_secret"]; v != "" {
		cfg.Storage.SecretKey = v
	}
	return nil
}

func normalizeOSSHost(ep string) string {
	ep = strings.TrimSpace(ep)
	ep = strings.TrimPrefix(ep, "https://")
	ep = strings.TrimPrefix(ep, "http://")
	return strings.Trim(ep, "/")
}
