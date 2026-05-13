-- ============================================================
-- 快速修复：删除 user_device_share 表的 invite_code 字段
-- 用于移除设备共享邀请码功能，简化为基于SN的接受流程
-- 执行方式: psql -U admin -d audio_platform -f remove_invite_code_field.sql
-- ============================================================

DO $$
BEGIN
    -- 检查并删除 invite_code 字段（如果存在）
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'user_device_share'
        AND column_name = 'invite_code'
    ) THEN

        -- 删除相关索引（如果存在）
        DROP INDEX IF EXISTS idx_user_device_share_invite_code;

        -- 删除 invite_code 字段
        ALTER TABLE public.user_device_share
        DROP COLUMN IF EXISTS invite_code;

        RAISE NOTICE '✅ 已成功删除 user_device_share 表的 invite_code 字段';

    ELSE
        RAISE NOTICE 'ℹ️  invite_code 字段不存在，无需删除';
    END IF;
END $$;

-- 验证字段是否已删除
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = 'user_device_share'
  AND column_name = 'invite_code';

RAISE NOTICE '';
RAISE NOTICE '🎉 邀请码字段已删除！现在可以重启服务测试了。';
