SET search_path TO public;
ALTER TABLE public.device_event_log DROP COLUMN IF EXISTS operator;
