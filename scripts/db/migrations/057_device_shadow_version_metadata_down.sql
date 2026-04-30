SET search_path TO public;

ALTER TABLE public.device_shadow DROP COLUMN IF EXISTS metadata;
ALTER TABLE public.device_shadow DROP COLUMN IF EXISTS version;
