-- ============================================================
-- 快速修复：仅创建 user_device_share 表
-- 用于解决 "关系 public.user_device_share 不存在 (SQLSTATE 42P01)" 错误
-- 执行方式: psql -U admin -d audio_platform -f create_user_device_share_table.sql
-- ============================================================

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'user_device_share') THEN

        -- 创建设备共享表
        CREATE TABLE public.user_device_share (
            id              BIGSERIAL PRIMARY KEY,
            sharer_user_id  BIGINT NOT NULL,                    -- 分享者用户ID
            receiver_user_id BIGINT NOT NULL,                   -- 接收者用户ID
            device_id       BIGINT NOT NULL,                    -- 设备ID
            sn              VARCHAR(64) NOT NULL,               -- 设备SN
            status          SMALLINT NOT NULL DEFAULT 0,         -- 状态：0=待接受，1=已接受，2=已拒绝，3=已撤销
            share_type      SMALLINT NOT NULL DEFAULT 1,         -- 共享类型：1=只读，2=可控制
            expire_at       TIMESTAMP WITH TIME ZONE,           -- 过期时间（NULL表示永久）
            created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            accepted_at     TIMESTAMP WITH TIME ZONE,            -- 接受时间
            revoked_at      TIMESTAMP WITH TIME ZONE             -- 撤销时间
        );

        -- 创建索引以优化查询性能
        CREATE INDEX idx_user_device_share_sharer ON public.user_device_share(sharer_user_id);
        CREATE INDEX idx_user_device_share_receiver ON public.user_device_share(receiver_user_id);
        CREATE INDEX idx_user_device_share_device ON public.user_device_share(device_id);
        CREATE INDEX idx_user_device_share_status ON public.user_device_share(status) WHERE status IN (0, 1);
        CREATE INDEX idx_user_device_share_expire ON public.user_device_share(expire_at) WHERE expire_at IS NOT NULL;

        -- 添加表注释
        COMMENT ON TABLE public.user_device_share IS '设备共享表';
        COMMENT ON COLUMN public.user_device_share.status IS '状态：0=待接受，1=已接受，2=已拒绝，3=已撤销';
        COMMENT ON COLUMN public.user_device_share.share_type IS '共享类型：1=只读，2=可控制';

        RAISE NOTICE '✅ 已成功创建 user_device_share 表';

    ELSE
        RAISE NOTICE 'ℹ️  user_device_share 表已存在，无需重复创建';
    END IF;
END $$;

-- 验证表是否创建成功
SELECT 
    table_name,
    (SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'user_device_share') as column_count,
    obj_description(oid) as table_comment
FROM information_schema.tables 
WHERE table_schema = 'public' AND table_name = 'user_device_share';

RAISE NOTICE '';
RAISE NOTICE '🎉 表创建完成！现在可以重启服务并测试设备共享功能了。';
