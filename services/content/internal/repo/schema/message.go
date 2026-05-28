package schema

import (
	"fmt"

	"gorm.io/gorm"
)

// EnsureMessageTables 幂等创建站内消息相关表
func EnsureMessageTables(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db nil")
	}
	return ensureMessageTablesInline(db)
}

func ensureMessageTablesInline(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS public.user_messages (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			message_type VARCHAR(32) NOT NULL DEFAULT 'system',
			title VARCHAR(255) NOT NULL DEFAULT '',
			content TEXT NOT NULL DEFAULT '',
			related_type VARCHAR(32) NOT NULL DEFAULT '',
			related_id BIGINT NOT NULL DEFAULT 0,
			action_url VARCHAR(500) NOT NULL DEFAULT '',
			is_read SMALLINT NOT NULL DEFAULT 0,
			read_at TIMESTAMP NULL,
			expire_at TIMESTAMP NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_messages_user_id ON public.user_messages(user_id) WHERE is_read = 0`,
		`CREATE INDEX IF NOT EXISTS idx_user_messages_user_read ON public.user_messages(user_id, is_read)`,
		`CREATE INDEX IF NOT EXISTS idx_user_messages_type ON public.user_messages(user_id, message_type, created_at DESC)`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			return fmt.Errorf("%w (stmt: %.80s...)", err, s)
		}
	}
	return nil
}
