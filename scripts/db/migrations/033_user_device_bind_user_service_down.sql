SET search_path TO public;

DROP INDEX IF EXISTS uk_user_device_bind_device_active;

ALTER TABLE public.user_device_bind DROP COLUMN IF EXISTS system_version;
ALTER TABLE public.user_device_bind DROP COLUMN IF EXISTS device_model;
ALTER TABLE public.user_device_bind DROP COLUMN IF EXISTS device_name;
