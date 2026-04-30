-- 设备指令历史：扩展状态（executing/timeout/cancelled）、操作人、原因
SET search_path TO public;

ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS operator VARCHAR(128) NOT NULL DEFAULT '';

ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS reason TEXT;

COMMENT ON COLUMN public.device_instruction.operator IS '操作人（管理员账号或系统）';
COMMENT ON COLUMN public.device_instruction.reason IS '下发原因说明';

ALTER TABLE public.device_instruction DROP CONSTRAINT IF EXISTS device_instruction_status_check;
ALTER TABLE public.device_instruction
  ADD CONSTRAINT device_instruction_status_check CHECK (status IN (1, 2, 3, 4, 5, 6));

COMMENT ON COLUMN public.device_instruction.status IS '1=pending 2=executing 3=success 4=failed 5=timeout 6=cancelled';

CREATE INDEX IF NOT EXISTS idx_device_instruction_created_at ON public.device_instruction (created_at DESC);
