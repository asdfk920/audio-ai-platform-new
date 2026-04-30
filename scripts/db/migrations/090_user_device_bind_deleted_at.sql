-- 用户设备绑定表：添加 deleted_at 字段（软删除）
SET search_path TO public;

ALTER TABLE public.user_device_bind
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

COMMENT ON COLUMN public.user_device_bind.deleted_at IS '软删除时间，NULL 表示未删除';
