-- Migration: Align content_download_stats columns with device service upsert SQL
-- Version: 101
-- Description: Rename legacy last_download_time; ensure created_at/updated_at exist for UPSERT INSERT

SET search_path TO public;
SET client_encoding TO 'UTF8';

ALTER TABLE IF EXISTS content_download_stats
	ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE IF EXISTS content_download_stats
	ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

DO $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = 'content_download_stats'
			AND column_name = 'last_download_time'
	) AND NOT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = 'content_download_stats'
			AND column_name = 'last_download_at'
	) THEN
		ALTER TABLE content_download_stats RENAME COLUMN last_download_time TO last_download_at;
	END IF;
END $$;

-- INSERT ... ON CONFLICT (content_id) 需要 content_id 上的唯一约束（部分环境建表时漏了 UNIQUE）
CREATE UNIQUE INDEX IF NOT EXISTS ux_content_download_stats_content_id ON content_download_stats (content_id);

COMMENT ON COLUMN content_download_stats.last_download_at IS '最后一次下载时间';
