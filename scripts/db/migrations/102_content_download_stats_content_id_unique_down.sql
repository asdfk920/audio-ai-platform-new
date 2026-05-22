-- Down migration for 102

SET search_path TO public;

DROP INDEX IF EXISTS ux_content_download_stats_content_id;
