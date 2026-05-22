-- Down migration for 101: optional rollback (rename column back only if safe)

SET search_path TO public;

DO $$
BEGIN
	IF EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = 'content_download_stats'
			AND column_name = 'last_download_at'
	) AND NOT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = 'content_download_stats'
			AND column_name = 'last_download_time'
	) THEN
		ALTER TABLE content_download_stats RENAME COLUMN last_download_at TO last_download_time;
	END IF;
END $$;
