-- 用户歌单表（与 services/content/migrations/001_ensure_playlists.sql 一致）
CREATE TABLE IF NOT EXISTS public.playlists (
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
    deleted_at TIMESTAMP NULL,
    CONSTRAINT playlists_status_check CHECK (status IN (0, 1, 2)),
    CONSTRAINT playlists_is_public_check CHECK (is_public IN (0, 1))
);

CREATE INDEX IF NOT EXISTS idx_playlists_user_id ON public.playlists(user_id);
CREATE INDEX IF NOT EXISTS idx_playlists_user_status ON public.playlists(user_id, status) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS public.playlist_songs (
    id BIGSERIAL PRIMARY KEY,
    playlist_id BIGINT NOT NULL REFERENCES public.playlists(id) ON DELETE CASCADE,
    content_id BIGINT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_playlist_songs_playlist_content UNIQUE (playlist_id, content_id)
);

CREATE INDEX IF NOT EXISTS idx_playlist_songs_playlist_id ON public.playlist_songs(playlist_id);
