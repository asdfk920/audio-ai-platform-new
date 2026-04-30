SET search_path TO public;

ALTER TABLE public.user_device_bind
  DROP COLUMN IF EXISTS operator;
