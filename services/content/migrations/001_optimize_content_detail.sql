-- 内容详情接口优化 - 数据库迁移脚本
-- 添加缺失字段以支持完整的内容详情返回

-- =====================================================
-- 1. 为 content 表添加缺失字段（如果不存在）
-- =====================================================

-- 添加 description 字段（描述/简介）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'content' AND column_name = 'description'
    ) THEN
        ALTER TABLE content ADD COLUMN description TEXT DEFAULT '';
        RAISE NOTICE '✅ 已添加 description 字段';
    ELSE
        RAISE NOTICE 'ℹ️  description 字段已存在';
    END IF;
END $$;

-- 添加 updated_at 字段（更新时间）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'content' AND column_name = 'updated_at'
    ) THEN
        ALTER TABLE content ADD COLUMN updated_at TIMESTAMP DEFAULT NOW();
        RAISE NOTICE '✅ 已添加 updated_at 字段';

        -- 将现有数据的 updated_at 设置为 created_at
        UPDATE content SET updated_at = created_at WHERE updated_at IS NULL;
        RAISE NOTICE '✅ 已初始化 updated_at 值';
    ELSE
        RAISE NOTICE 'ℹ️  updated_at 字段已存在';
    END IF;
END $$;

-- 添加 play_count 字段（播放量统计）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'content' AND column_name = 'play_count'
    ) THEN
        ALTER TABLE content ADD COLUMN play_count BIGINT DEFAULT 0;
        RAISE NOTICE '✅ 已添加 play_count 字段';
    ELSE
        RAISE NOTICE 'ℹ️  play_count 字段已存在';
    END IF;
END $$;

-- =====================================================
-- 2. 创建分类表（如果不存在）
-- =====================================================
CREATE TABLE IF NOT EXISTS content_category (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT DEFAULT '',
    icon_url VARCHAR(500) DEFAULT '',
    sort_order INT DEFAULT 0,
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_content_category_status ON content_category(status);

RAISE NOTICE '✅ content_category 表就绪';

-- 插入默认分类（如果为空）
INSERT INTO content_category (name, description, sort_order) VALUES
    ('空间音频', '空间音频/3D环绕音效', 1),
    ('沉浸式', '沉浸式音频体验', 2),
    ('助眠', '助眠/冥想/放松音乐', 3),
    ('古典', '古典音乐', 4),
    ('流行', '流行音乐', 5),
    ('摇滚', '摇滚音乐', 6),
    ('电子', '电子/舞曲音乐', 7),
    ('爵士', '爵士音乐', 8)
ON CONFLICT (name) DO NOTHING;

RAISE NOTICE '✅ 默认分类数据已插入';

-- =====================================================
-- 3. 创建标签表（如果不存在）
-- =====================================================
CREATE TABLE IF NOT EXISTS tag (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    color VARCHAR(20) DEFAULT '#1890ff',
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(name)
);

CREATE INDEX IF NOT EXISTS idx_tag_status ON tag(status);

RAISE NOTICE '✅ tag 表就绪';

-- 插入默认标签（如果为空）
INSERT INTO tag (name, color) VALUES
    ('3D', '#ff4d4f'),
    ('沉浸式', '#722ed1'),
    ('助眠', '#52c41a'),
    ('空间音效', '#1890ff'),
    ('高音质', '#faad14'),
    ('无损', '#eb2f96'),
    ('热门', '#f5222d'),
    ('新歌', '#13c2c2'),
    ('推荐', '#2f54eb')
ON CONFLICT (name) DO NOTHING;

RAISE NOTICE '✅ 默认标签数据已插入';

-- =====================================================
-- 4. 创建内容-标签关联表（如果不存在）
-- =====================================================
CREATE TABLE IF NOT EXISTS content_tag (
    id SERIAL PRIMARY KEY,
    content_id BIGINT NOT NULL,
    tag_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(content_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_content_tag_content_id ON content_tag(content_id);
CREATE INDEX IF NOT EXISTS idx_content_tag_tag_id ON content_tag(tag_id);

RAISE NOTICE '✅ content_tag 表就绪';

-- =====================================================
-- 5. 创建评论表（如果不存在）
-- =====================================================
CREATE TABLE IF NOT EXISTS comments (
    id BIGSERIAL PRIMARY KEY,
    content_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    parent_id BIGINT DEFAULT 0,
    content TEXT NOT NULL,
    like_count INT DEFAULT 0,
    status SMALLINT DEFAULT 1,
    is_deleted SMALLINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comments_content_id ON comments(content_id, status, is_deleted);
CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments(user_id);
CREATE INDEX IF NOT EXISTS idx_comments_parent_id ON comments(parent_id);

RAISE NOTICE '✅ comments 表就绪';

-- =====================================================
-- 6. 创建用户点赞表（如果不存在）
-- =====================================================
CREATE TABLE IF NOT EXISTS user_likes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    content_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, content_id)
);

CREATE INDEX IF NOT EXISTS idx_user_likes_user_id ON user_likes(user_id);
CREATE INDEX IF NOT EXISTS idx_user_likes_content_id ON user_likes(content_id);

RAISE NOTICE '✅ user_likes 表就绪';

-- =====================================================
-- 7. 创建触发器：自动更新 updated_at
-- =====================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为 content 表创建更新触发器
DROP TRIGGER IF EXISTS update_content_updated_at ON content;
CREATE TRIGGER update_content_updated_at
    BEFORE UPDATE ON content
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

RAISE NOTICE '✅ updated_at 自动更新触发器已创建';

-- =====================================================
-- 完成
-- =====================================================
RAISE NOTICE '';
RAISE NOTICE '========================================';
RAISE NOTICE '🎉 数据库迁移完成！';
RAISE NOTICE '========================================';
RAISE NOTICE '新增功能:';
RAISE NOTICE '  ✅ content.description (描述)';
RAISE NOTICE '  ✅ content.updated_at (更新时间)';
RAISE NOTICE '  ✅ content.play_count (播放量)';
RAISE NOTICE '  ✅ content_category (分类表)';
RAISE NOTE '  ✅ tag + content_tag (标签系统)';
RAISE NOTICE '  ✅ comments (评论表)';
RAISE NOTICE '  ✅ user_likes (点赞表)';
RAISE NOTICE '  ✅ 自动更新 updated_at 触发器';
RAISE NOTICE '========================================';