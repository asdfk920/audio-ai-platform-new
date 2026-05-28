package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

// EnsurePlaylistTables 幂等创建歌单相关表
func EnsurePlaylistTables(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db nil")
	}
	candidates := []string{
		filepath.Join("migrations", "001_ensure_playlists.sql"),
		filepath.Join("services", "content", "migrations", "001_ensure_playlists.sql"),
	}
	if exe, err := os.Executable(); err == nil {
		base := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(base, "migrations", "001_ensure_playlists.sql"))
	}
	for _, p := range candidates {
		sqlBytes, err := os.ReadFile(p)
		if err == nil {
			if err := execSQLScript(db, string(sqlBytes)); err != nil {
				return err
			}
			return nil
		}
	}
	return ensurePlaylistTablesInline(db)
}

func ensurePlaylistTablesInline(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS public.playlists (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			sn VARCHAR(64) NOT NULL DEFAULT '',
			name VARCHAR(100) NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			cover_url VARCHAR(1000) NOT NULL DEFAULT '',
			song_count INTEGER NOT NULL DEFAULT 0,
			is_public SMALLINT NOT NULL DEFAULT 1,
			is_default SMALLINT NOT NULL DEFAULT 0,
			status SMALLINT NOT NULL DEFAULT 1,
			source VARCHAR(32) NOT NULL DEFAULT 'local',
			source_id VARCHAR(128) NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_playlists_user_id ON public.playlists(user_id)`,
		`CREATE TABLE IF NOT EXISTS public.playlist_songs (
			id BIGSERIAL PRIMARY KEY,
			playlist_id BIGINT NOT NULL REFERENCES public.playlists(id) ON DELETE CASCADE,
			content_id BIGINT NOT NULL,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT uk_playlist_songs_playlist_content UNIQUE (playlist_id, content_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_playlist_songs_playlist_id ON public.playlist_songs(playlist_id)`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			return err
		}
	}
	return nil
}

func execSQLScript(db *gorm.DB, script string) error {
	for _, block := range strings.Split(script, ";") {
		s := strings.TrimSpace(block)
		if s == "" || strings.HasPrefix(s, "--") {
			continue
		}
		if err := db.Exec(s).Error; err != nil {
			return fmt.Errorf("%w (stmt: %.80s...)", err, s)
		}
	}
	return nil
}
