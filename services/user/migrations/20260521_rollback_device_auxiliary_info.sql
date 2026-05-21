-- Rollback: Remove device auxiliary info fields from user_device_bind table
-- Date: 2026-05-21
-- Description: 回滚：删除 user_device_bind 表的辅助信息字段（如果需要）

DO $$
BEGIN
    -- 删除 location 字段（如果存在）
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'user_device_bind' 
        AND column_name = 'location'
    ) THEN
        ALTER TABLE public.user_device_bind DROP COLUMN location;
        RAISE NOTICE '✅ 已删除 location 字段';
    ELSE
        RAISE NOTICE 'ℹ️  location 字段不存在，无需删除';
    END IF;

    -- 删除 group_name 字段（如果存在）
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'user_device_bind' 
        AND column_name = 'group_name'
    ) THEN
        ALTER TABLE public.user_device_bind DROP COLUMN group_name;
        RAISE NOTICE '✅ 已删除 group_name 字段';
    ELSE
        RAISE NOTICE 'ℹ️  group_name 字段不存在，无需删除';
    END IF;

    -- 删除 scene 字段（如果存在）
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'user_device_bind' 
        AND column_name = 'scene'
    ) THEN
        ALTER TABLE public.user_device_bind DROP COLUMN scene;
        RAISE NOTICE '✅ 已删除 scene 字段';
    ELSE
        RAISE NOTICE 'ℹ️  scene 字段不存在，无需删除';
    END IF;

    RAISE NOTICE '✅ 回滚完成：user_device_bind 表辅助信息字段已删除';
END $$;
