-- Migration: 添加 artist_id 字段到 content 表
-- Created: 2026-04-28

-- 添加 artist_id 字段
ALTER TABLE content ADD COLUMN IF NOT EXISTS artist_id BIGINT DEFAULT 0;

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_content_artist_id ON content(artist_id);

-- 更新现有数据：根据 artist 字段匹配 artists 表设置 artist_id
UPDATE content c
SET artist_id = COALESCE(a.id, 0)
FROM artists a
WHERE c.artist = a.name;

-- 对于没有匹配到的，保持默认值 0
