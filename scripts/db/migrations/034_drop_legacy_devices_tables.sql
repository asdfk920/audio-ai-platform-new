-- 删除历史设备相关表：devices、device_user_rel、device_commands（已由 public.device + user_device_bind + device_instruction 替代）
-- 先解除 content_play_records 对 public.devices 的外键，再删 devices。
SET search_path TO public;

DO $$
DECLARE
  r RECORD;
BEGIN
  IF EXISTS (
    SELECT 1 FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'public' AND c.relname = 'content_play_records'
  ) AND EXISTS (
    SELECT 1 FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'public' AND c.relname = 'devices'
  ) THEN
    FOR r IN
      SELECT con.conname AS conname
        FROM pg_constraint con
        JOIN pg_class rel ON rel.oid = con.conrelid
        JOIN pg_namespace nsp ON nsp.oid = rel.relnamespace
        JOIN pg_class ref ON ref.oid = con.confrelid
        JOIN pg_namespace rn ON rn.oid = ref.relnamespace
       WHERE nsp.nspname = 'public'
         AND rel.relname = 'content_play_records'
         AND rn.nspname = 'public'
         AND ref.relname = 'devices'
         AND con.contype = 'f'
    LOOP
      EXECUTE format('ALTER TABLE public.content_play_records DROP CONSTRAINT IF EXISTS %I', r.conname);
    END LOOP;
  END IF;
END $$;

DROP TABLE IF EXISTS public.device_user_rel CASCADE;
DROP TABLE IF EXISTS public.device_commands CASCADE;
DROP TABLE IF EXISTS public.devices CASCADE;

COMMENT ON COLUMN public.content_play_records.device_id IS '可选设备主键引用；历史曾外键 devices(id)，已移除；与新 device 表对齐由业务层维护';
