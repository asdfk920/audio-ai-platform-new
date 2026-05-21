-- Verification: Check device auxiliary info fields in user_device_bind table
-- Date: 2026-05-21
-- Description: 验证 user_device_bind 表是否包含辅助信息字段

SELECT 
    column_name,
    data_type,
    character_maximum_length,
    is_nullable,
    column_default,
    pg_catalog.col_description((table_schema || '.' || table_name)::regclass::oid, ordinal_position) as comment
FROM information_schema.columns 
WHERE table_schema = 'public' 
  AND table_name = 'user_device_bind'
  AND column_name IN ('location', 'group_name', 'scene')
ORDER BY ordinal_position;

-- 预期结果：应该返回3行记录（location, group_name, scene）
-- 如果返回0行，说明字段不存在，需要执行迁移脚本
