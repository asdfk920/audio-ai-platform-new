package schema

import (
	"fmt"

	"gorm.io/gorm"
)

// EnsureArtistSubscriptionTables 幂等创建艺术家订阅相关表
func EnsureArtistSubscriptionTables(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db nil")
	}
	return ensureArtistSubscriptionTablesInline(db)
}

func ensureArtistSubscriptionTablesInline(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS public.artist_subscriptions (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			artist_id BIGINT NOT NULL,
			artist_name VARCHAR(255) NOT NULL DEFAULT '',
			status SMALLINT NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT uk_artist_subscriptions_user_artist UNIQUE (user_id, artist_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_artist_subscriptions_user_id ON public.artist_subscriptions(user_id) WHERE status = 1`,
		`CREATE INDEX IF NOT EXISTS idx_artist_subscriptions_artist_id ON public.artist_subscriptions(artist_id) WHERE status = 1`,
		`CREATE INDEX IF NOT EXISTS idx_artist_subscriptions_user_status ON public.artist_subscriptions(user_id, status)`,
		`ALTER TABLE public.content ADD COLUMN IF NOT EXISTS subscribe_count BIGINT NOT NULL DEFAULT 0`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			return fmt.Errorf("%w (stmt: %.80s...)", err, s)
		}
	}
	return nil
}
