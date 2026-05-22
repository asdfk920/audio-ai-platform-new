-- 用户收藏表
-- 记录用户对内容的收藏关系（歌曲、歌单、专辑等）
-- 支持收藏/取消收藏操作，使用事务保证数据一致性

CREATE TABLE IF NOT EXISTS user_favorites (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,                -- 用户ID
    content_id      BIGINT NOT NULL,                -- 内容ID（对应content表）
    favorite_type   VARCHAR(20) NOT NULL DEFAULT 'song', -- 收藏类型：song(歌曲)/playlist(歌单)/album(专辑)
    status          SMALLINT NOT NULL DEFAULT 1,    -- 状态：1=已收藏, 0=已取消
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

-- 创建唯一索引：防止重复收藏同一内容
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_favorites_unique ON user_favorites (user_id, content_id) WHERE status = 1;

-- 创建普通索引：加速按用户查询
CREATE INDEX IF NOT EXISTS idx_user_favorites_user_id ON user_favorites (user_id);

-- 创建普通索引：加速按内容查询
CREATE INDEX IF NOT EXISTS idx_user_favorites_content_id ON user_favorites (content_id);

-- 添加注释
COMMENT ON TABLE user_favorites IS '用户收藏表';
COMMENT ON COLUMN user_favorites.id IS '主键ID';
COMMENT ON COLUMN user_favorites.user_id IS '用户ID';
COMMENT ON COLUMN user_favorites.content_id IS '内容ID（对应content表）';
COMMENT ON COLUMN user_favorites.favorite_type IS '收藏类型：song(歌曲)/playlist(歌单)/album(专辑)';
COMMENT ON COLUMN user_favorites.status IS '状态：1=已收藏, 0=已取消';
COMMENT ON COLUMN user_favorites.created_at IS '创建时间';
COMMENT ON COLUMN user_favorites.updated_at IS '更新时间';

-- 为content表添加favorite_count字段（如果不存在）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'content' AND column_name = 'favorite_count'
    ) THEN
        ALTER TABLE content ADD COLUMN favorite_count BIGINT DEFAULT 0;
        COMMENT ON COLUMN content.favorite_count IS '收藏总数';
        RAISE NOTICE '已添加字段: content.favorite_count';
    ELSE
        RAISE NOTICE '字段已存在: content.favorite_count';
    END IF;
END $$;
