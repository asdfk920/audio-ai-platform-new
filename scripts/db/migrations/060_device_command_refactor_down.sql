SET search_path TO public;

DROP INDEX IF EXISTS idx_device_instruction_user;
DROP INDEX IF EXISTS idx_device_instruction_dispatch;
DROP INDEX IF EXISTS idx_device_instruction_schedule_id;

ALTER TABLE public.device_instruction DROP CONSTRAINT IF EXISTS device_instruction_status_check;
ALTER TABLE public.device_instruction
  ADD CONSTRAINT device_instruction_status_check CHECK (status IN (1, 2, 3, 4, 5, 6));

ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS schedule_id;
ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS merged_from_count;
ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS max_retry;
ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS executed_at;
ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS dispatched_at;
ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS expires_at;
ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS priority;
ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS command_code;
ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS instruction_type;

DROP TRIGGER IF EXISTS trg_device_command_schedule_set_updated_at ON public.device_command_schedule;
DROP INDEX IF EXISTS idx_device_command_schedule_user;
DROP INDEX IF EXISTS idx_device_command_schedule_due;
DROP INDEX IF EXISTS idx_device_command_schedule_log_schedule;
DROP TABLE IF EXISTS public.device_command_schedule_log;
DROP TABLE IF EXISTS public.device_command_schedule;
