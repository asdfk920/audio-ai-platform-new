-- 用户订阅艺术家表（内容微服务启动时可幂等执行）
-- 用于存储用户与艺术家的订阅关系，支持粉丝数统计、去重订阅等功能

CREATE TABLE IF NOT EXISTS public.artist_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,                    -- 用户ID
    artist_id BIGINT NOT NULL,                  -- 艺术家ID
    artist_name VARCHAR(255) NOT NULL DEFAULT '', -- 艺术家名称（冗余字段，避免频繁关联查询）
    status SMALLINT NOT NULL DEFAULT 1,         -- 订阅状态：1=已订阅 0=已取消
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- 唯一约束：同一用户不能重复订阅同一艺术家
    CONSTRAINT uk_artist_subscriptions_user_artist UNIQUE (user_id, artist_id)
);

-- 索引优化：按用户查询订阅列表
CREATE INDEX IF NOT EXISTS idx_artist_subscriptions_user_id ON public.artist_subscriptions(user_id) WHERE status = 1;

-- 索引优化：按艺术家查询粉丝列表
CREATE INDEX IF NOT EXISTS idx_artist_subscriptions_artist_id ON public.artist_subscriptions(artist_id) WHERE status = 1;

-- 复合索引：用户+状态（用于查询某用户的所有订阅）
CREATE INDEX IF NOT EXISTS idx_artist_subscriptions_user_status ON public.artist_subscriptions(user_id, status);

COMMENT ON TABLE public.artist_subscriptions IS '用户订阅艺术家表';
COMMENT ON COLUMN public.artist_subscriptions.user_id IS '用户ID';
COMMENT ON COLUMN public.artist_subscriptions.artist_id IS '艺术家ID';
COMMENT ON COLUMN public.artist_subscriptions.artist_name IS '艺术家名称（冗余）';
COMMENT ON COLUMN public.artist_subscriptions.status IS '订阅状态：1=已订阅 0=已取消';

-- 确保content表有subscribe_count字段（用于记录艺术家订阅数/粉丝数）
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'artists'
        AND column_name = 'fan_count'
    ) THEN
        -- 如果存在独立的artists表
        ALTER TABLE public.artists ADD COLUMN IF NOT EXISTS fan_count BIGINT NOT NULL DEFAULT 0;
        COMMENT ON COLUMN public.artists.fan_count IS '粉丝数/订阅数';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'content'
        AND column_name = 'subscribe_count'
    ) THEN
        -- 如果使用content表存储艺术家信息
        ALTER TABLE public.content ADD COLUMN IF NOT EXISTS subscribe_count BIGINT NOT NULL DEFAULT 0;
        COMMENT ON COLUMN public.content.subscribe_count IS '订阅数/粉丝数（当该条目为艺术家时有效）';
    END IF;
END $$;
