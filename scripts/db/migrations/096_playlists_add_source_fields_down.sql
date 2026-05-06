-- 回滚：删除 playlists 表的 source 和 source_id 字段

-- 删除索引
DROP INDEX IF EXISTS idx_playlists_user_source;
DROP INDEX IF EXISTS idx_playlists_source_id;
DROP INDEX IF EXISTS idx_playlists_source;

-- 删除字段
ALTER TABLE playlists DROP COLUMN IF EXISTS source;
ALTER TABLE playlists DROP COLUMN IF EXISTS source_id;
