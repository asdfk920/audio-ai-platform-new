-- 回滚设备模块表结构（PostgreSQL）
SET search_path TO public;

DROP TABLE IF EXISTS public.device_group CASCADE;
DROP TABLE IF EXISTS public.device_event_log CASCADE;
DROP TABLE IF EXISTS public.device_certificate CASCADE;
DROP TABLE IF EXISTS public.ota_upgrade_task CASCADE;
DROP TABLE IF EXISTS public.ota_firmware CASCADE;
DROP TABLE IF EXISTS public.device_instruction CASCADE;
DROP TABLE IF EXISTS public.device_state_log CASCADE;
DROP TABLE IF EXISTS public.device_shadow CASCADE;
DROP TABLE IF EXISTS public.user_device_bind CASCADE;
DROP TABLE IF EXISTS public.device CASCADE;

-- 若其它表仍引用该函数，可在后续迁移中再创建；这里回滚时一并移除。
DROP FUNCTION IF EXISTS public.set_updated_at();

