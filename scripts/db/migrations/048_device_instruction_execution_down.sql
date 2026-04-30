SET search_path TO public;

DROP TABLE IF EXISTS public.device_instruction_state_log CASCADE;

ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS error_msg;
ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS retry_count;
ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS timeout_seconds;
ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS completed_at;
ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS received_at;
