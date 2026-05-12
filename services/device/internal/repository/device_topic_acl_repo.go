package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jacklau/audio-ai-platform/services/device/internal/model"
)

type DeviceTopicACLRepo struct {
	db *sql.DB
}

func NewDeviceTopicACLRepo(db *sql.DB) *DeviceTopicACLRepo {
	return &DeviceTopicACLRepo{db: db}
}

func (r *DeviceTopicACLRepo) GetDeviceACLRules(ctx context.Context, sn string) ([]model.DeviceTopicACL, error) {
	sn = strings.ToUpper(strings.TrimSpace(sn))
	if sn == "" {
		return nil, nil
	}

	query := `
		SELECT id, sn, topic_pattern, action, permission, description, created_at, updated_at
		FROM device_topic_acl
		WHERE sn = $1
		ORDER BY 
			CASE permission WHEN 1 THEN 0 ELSE 1 END,
			id
	`

	rows, err := r.db.QueryContext(ctx, query, sn)
	if err != nil {
		return nil, fmt.Errorf("查询设备 ACL 规则失败: %v", err)
	}
	defer rows.Close()

	var rules []model.DeviceTopicACL
	for rows.Next() {
		var rule model.DeviceTopicACL
		err := rows.Scan(
			&rule.ID, &rule.Sn, &rule.TopicPattern, &rule.Action,
			&rule.Permission, &rule.Description, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描 ACL 规则失败: %v", err)
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

func (r *DeviceTopicACLRepo) GetGlobalACLRules(ctx context.Context) ([]model.DeviceTopicACL, error) {
	query := `
		SELECT id, sn, topic_pattern, action, permission, description, created_at, updated_at
		FROM device_topic_acl
		WHERE sn = '*'
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询全局 ACL 规则失败: %v", err)
	}
	defer rows.Close()

	var rules []model.DeviceTopicACL
	for rows.Next() {
		var rule model.DeviceTopicACL
		err := rows.Scan(
			&rule.ID, &rule.Sn, &rule.TopicPattern, &rule.Action,
			&rule.Permission, &rule.Description, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描全局 ACL 规则失败: %v", err)
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

func (r *DeviceTopicACLRepo) CreateDefaultRulesForDevice(ctx context.Context, sn string) error {
	sn = strings.ToUpper(strings.TrimSpace(sn))

	defaultRules := []struct {
		topicPattern string
		action       int16
		permission   int16
		description  string
	}{
		{"device/" + sn + "/status", model.TopicActionPublish, model.PermissionAllow, "设备状态上报"},
		{"device/" + sn + "/log", model.TopicActionPublish, model.PermissionAllow, "设备日志上报"},
		{"audio/" + sn + "/upload", model.TopicActionPublish, model.PermissionAllow, "音频数据上传"},
		{"cmd/" + sn + "/down", model.TopicActionSubscribe, model.PermissionAllow, "平台指令下发"},
		{"ota/" + sn + "/info", model.TopicActionSubscribe, model.PermissionAllow, "OTA升级通知"},
		{"diagnose/" + sn + "/cmd", model.TopicActionSubscribe, model.PermissionAllow, "远程诊断指令"},
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始事务失败: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO device_topic_acl (sn, topic_pattern, action, permission, description)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (sn, topic_pattern, action) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("准备插入语句失败: %v", err)
	}
	defer stmt.Close()

	for _, rule := range defaultRules {
		_, err := stmt.ExecContext(ctx, sn, rule.topicPattern, rule.action, rule.permission, rule.description)
		if err != nil {
			return fmt.Errorf("插入默认规则失败 (%s): %v", rule.topicPattern, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %v", err)
	}

	return nil
}