-- 为 playlists 表添加 source 和 source_id 字段
-- 用于支持第三方歌单导入功能（Spotify、QQ 音乐等）

-- 添加 source 字段：标识歌单来源 (local, spotify, qq-music)
ALTER TABLE playlists 
ADD COLUMN IF NOT EXISTS source VARCHAR(50) DEFAULT 'local' NOT NULL;

-- 添加 source_id 字段：第三方歌单 ID
ALTER TABLE playlists 
ADD COLUMN IF NOT EXISTS source_id VARCHAR(100);

-- 添加注释
COMMENT ON COLUMN playlists.source IS '歌单来源：local=本地，spotify=Spotify，qq-music=QQ 音乐';
COMMENT ON COLUMN playlists.source_id IS '第三方歌单 ID，本地歌单为 NULL';

-- 创建索引以优化查询性能
CREATE INDEX IF NOT EXISTS idx_playlists_source ON playlists(source);
CREATE INDEX IF NOT EXISTS idx_playlists_source_id ON playlists(source_id);
CREATE INDEX IF NOT EXISTS idx_playlists_user_source ON playlists(user_id, source);
