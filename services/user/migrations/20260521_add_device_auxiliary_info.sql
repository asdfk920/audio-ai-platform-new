-- Migration: Add device auxiliary info fields to user_device_bind table
-- Date: 2026-05-21
-- Description: 为 user_device_bind 表添加用户自定义的辅助信息字段
--              支持用户设置设备的位置、分组、常用场景等信息

DO $$
BEGIN
    -- 检查 location 字段是否已存在，不存在则添加
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'user_device_bind' 
        AND column_name = 'location'
    ) THEN
        ALTER TABLE public.user_device_bind 
        ADD COLUMN location VARCHAR(100) DEFAULT '' NOT NULL;
        
        COMMENT ON COLUMN public.user_device_bind.location IS '设备位置信息（如：卧室、客厅、书房等）';
        
        RAISE NOTICE '✅ 已添加 location 字段';
    ELSE
        RAISE NOTICE 'ℹ️  location 字段已存在';
    END IF;

    -- 检查 group_name 字段是否已存在，不存在则添加
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'user_device_bind' 
        AND column_name = 'group_name'
    ) THEN
        ALTER TABLE public.user_device_bind 
        ADD COLUMN group_name VARCHAR(100) DEFAULT '' NOT NULL;
        
        COMMENT ON COLUMN public.user_device_bind.group_name IS '设备分组名称（如：家庭、办公室、客厅音响等）';
        
        RAISE NOTICE '✅ 已添加 group_name 字段';
    ELSE
        RAISE NOTICE 'ℹ️  group_name 字段已存在';
    END IF;

    -- 检查 scene 字段是否已存在，不存在则添加
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'user_device_bind' 
        AND column_name = 'scene'
    ) THEN
        ALTER TABLE public.user_device_bind 
        ADD COLUMN scene VARCHAR(200) DEFAULT '' NOT NULL;
        
        COMMENT ON COLUMN public.user_device_bind.scene IS '常用使用场景（如：晨间唤醒、睡前音乐、聚会背景音等）';
        
        RAISE NOTICE '✅ 已添加 scene 字段';
    ELSE
        RAISE NOTICE 'ℹ️  scene 字段已存在';
    END IF;

    RAISE NOTICE '✅ user_device_bind 表辅助信息字段迁移完成';
END $$;
