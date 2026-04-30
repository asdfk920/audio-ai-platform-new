-- 单条指令执行状态：时间字段、超时/重试/错误信息 + 状态流水表
SET search_path TO public;

ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS received_at TIMESTAMP NULL;
ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS completed_at TIMESTAMP NULL;
ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS timeout_seconds INT NOT NULL DEFAULT 300;
ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0;
ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS error_msg TEXT;

COMMENT ON COLUMN public.device_instruction.received_at IS '设备已接收/开始执行（MQTT 送达等）';
COMMENT ON COLUMN public.device_instruction.completed_at IS '终态完成时间（成功/失败/超时/取消）';
COMMENT ON COLUMN public.device_instruction.timeout_seconds IS '超时判定秒数';
COMMENT ON COLUMN public.device_instruction.retry_count IS '重试次数';
COMMENT ON COLUMN public.device_instruction.error_msg IS '失败或超时说明';

CREATE TABLE IF NOT EXISTS public.device_instruction_state_log (
  id BIGSERIAL PRIMARY KEY,
  instruction_id BIGINT NOT NULL REFERENCES public.device_instruction(id) ON DELETE CASCADE,
  from_status SMALLINT NULL,
  to_status SMALLINT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  operator VARCHAR(128) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_instruction_state_log_instruction_id
  ON public.device_instruction_state_log (instruction_id, id ASC);

COMMENT ON TABLE public.device_instruction_state_log IS '指令状态变化时序（审计）';
