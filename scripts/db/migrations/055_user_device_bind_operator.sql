-- 为用户设备绑定表补充 operator 字段
SET search_path TO public;

ALTER TABLE public.user_device_bind
  ADD COLUMN IF NOT EXISTS operator VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN public.user_device_bind.operator IS '绑定操作人标识';
