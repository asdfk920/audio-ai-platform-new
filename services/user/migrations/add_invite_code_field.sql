-- ============================================================
-- 快速修复：为 user_device_share 表添加 invite_code 字段
-- 用于解决"发起设备共享没有邀请码"的问题
-- 执行方式: psql -U admin -d audio_platform -f add_invite_code_field.sql
-- ============================================================

DO $$
BEGIN
    -- 检查并添加 invite_code 字段（如果不存在）
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'user_device_share'
        AND column_name = 'invite_code'
    ) THEN

        -- 添加邀请码字段
        ALTER TABLE public.user_device_share
        ADD COLUMN invite_code VARCHAR(64) UNIQUE;

        -- 为现有数据生成邀请码（如果有旧数据）
        UPDATE public.user_device_share
        SET invite_code = 'INV' || LPAD(id::TEXT, 8, '0') || '_' || SUBSTRING(MD5(id::TEXT || NOW()::TEXT), 1, 6)
        WHERE invite_code IS NULL;

        -- 添加索引以优化查询性能
        CREATE INDEX idx_user_device_share_invite_code ON public.user_device_share(invite_code);

        -- 添加注释
        COMMENT ON COLUMN public.user_device_share.invite_code IS '设备共享邀请码（唯一标识）';

        RAISE NOTICE '✅ 已成功添加 invite_code 字段到 user_device_share 表';

    ELSE
        RAISE NOTICE 'ℹ️  invite_code 字段已存在';
    END IF;
END $$;

-- 验证字段是否添加成功
SELECT 
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_schema = 'public' 
  AND table_name = 'user_device_share'
  AND column_name = 'invite_code';

RAISE NOTICE '';
RAISE NOTICE '🎉 邀请码字段已就绪！现在可以重启服务测试了。';
