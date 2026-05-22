-- Migration: Unique index on content_download_stats.content_id for UPSERT
-- Version: 102
-- Description: Fixes SQLSTATE 42P10 (ON CONFLICT requires unique/exclusion constraint on target)

SET search_path TO public;
SET client_encoding TO 'UTF8';

-- 若执行失败且提示 duplicate key，说明存在重复 content_id，需先合并/删除重复行后再执行，例如：
-- DELETE FROM content_download_stats a USING content_download_stats b
--   WHERE a.id > b.id AND a.content_id = b.content_id;

CREATE UNIQUE INDEX IF NOT EXISTS ux_content_download_stats_content_id ON content_download_stats (content_id);
