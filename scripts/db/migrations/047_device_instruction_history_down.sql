SET search_path TO public;

DROP INDEX IF EXISTS idx_device_instruction_created_at;

ALTER TABLE public.device_instruction DROP CONSTRAINT IF EXISTS device_instruction_status_check;
-- 回退前需确保无 status 5/6 数据
UPDATE public.device_instruction SET status = 4 WHERE status IN (5, 6);
ALTER TABLE public.device_instruction
  ADD CONSTRAINT device_instruction_status_check CHECK (status IN (1, 2, 3, 4));

ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS reason;
ALTER TABLE public.device_instruction DROP COLUMN IF EXISTS operator;
