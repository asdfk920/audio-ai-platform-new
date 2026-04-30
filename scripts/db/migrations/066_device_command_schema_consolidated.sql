-- 一次性补齐 device 指令域表结构（等价于按序执行 047 + 048 + 060）。
-- 适用于本地/测试库未跑全量迁移、出现「字段 expires_at 不存在」等错误时使用。
-- 幂等，可重复执行。
SET search_path TO public;

-- ========== 047_device_instruction_history.sql ==========
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

-- ========== 048_device_instruction_execution.sql ==========
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

-- ========== 060_device_command_refactor.sql ==========
CREATE TABLE IF NOT EXISTS public.device_command_schedule (
  id BIGSERIAL PRIMARY KEY,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  device_sn VARCHAR(64) NOT NULL,
  user_id BIGINT NOT NULL DEFAULT 0,
  schedule_type VARCHAR(16) NOT NULL DEFAULT 'once',
  desired_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  command_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  merge_desired BOOLEAN NOT NULL DEFAULT TRUE,
  cron_expr VARCHAR(128) NULL,
  timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
  next_execute_at TIMESTAMP NULL,
  last_execute_at TIMESTAMP NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  expires_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT device_command_schedule_type_check CHECK (schedule_type IN ('once', 'cron')),
  CONSTRAINT device_command_schedule_status_check CHECK (status IN ('active', 'paused', 'cancelled', 'completed'))
);

CREATE INDEX IF NOT EXISTS idx_device_command_schedule_due
  ON public.device_command_schedule (status, next_execute_at ASC);
CREATE INDEX IF NOT EXISTS idx_device_command_schedule_user
  ON public.device_command_schedule (user_id, device_id, created_at DESC);

DROP TRIGGER IF EXISTS trg_device_command_schedule_set_updated_at ON public.device_command_schedule;
CREATE TRIGGER trg_device_command_schedule_set_updated_at
BEFORE UPDATE ON public.device_command_schedule
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TABLE IF NOT EXISTS public.device_command_schedule_log (
  id BIGSERIAL PRIMARY KEY,
  schedule_id BIGINT NOT NULL REFERENCES public.device_command_schedule(id) ON DELETE CASCADE,
  instruction_id BIGINT NULL REFERENCES public.device_instruction(id) ON DELETE SET NULL,
  run_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  status VARCHAR(16) NOT NULL DEFAULT 'created',
  note TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT device_command_schedule_log_status_check CHECK (status IN ('created', 'executed', 'failed', 'cancelled', 'completed'))
);

CREATE INDEX IF NOT EXISTS idx_device_command_schedule_log_schedule
  ON public.device_command_schedule_log (schedule_id, id DESC);

ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS instruction_type VARCHAR(16) NOT NULL DEFAULT 'manual';
ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS command_code VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS priority INT NOT NULL DEFAULT 100;
ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP NULL;
ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS dispatched_at TIMESTAMP NULL;
ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS executed_at TIMESTAMP NULL;
ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS max_retry INT NOT NULL DEFAULT 3;
ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS merged_from_count INT NOT NULL DEFAULT 0;
ALTER TABLE public.device_instruction
  ADD COLUMN IF NOT EXISTS schedule_id BIGINT NULL REFERENCES public.device_command_schedule(id) ON DELETE SET NULL;

UPDATE public.device_instruction
SET command_code = cmd
WHERE COALESCE(command_code, '') = '';

ALTER TABLE public.device_instruction DROP CONSTRAINT IF EXISTS device_instruction_status_check;
ALTER TABLE public.device_instruction
  ADD CONSTRAINT device_instruction_status_check CHECK (status IN (1, 2, 3, 4, 5, 6));

CREATE INDEX IF NOT EXISTS idx_device_instruction_schedule_id
  ON public.device_instruction (schedule_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_device_instruction_dispatch
  ON public.device_instruction (device_id, status, priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_device_instruction_user
  ON public.device_instruction (user_id, created_at DESC);

COMMENT ON COLUMN public.device_instruction.instruction_type IS 'manual / scheduled';
COMMENT ON COLUMN public.device_instruction.command_code IS '统一命令码，兼容旧 cmd';
COMMENT ON COLUMN public.device_instruction.priority IS '优先级，数值越大越先下发';
COMMENT ON COLUMN public.device_instruction.expires_at IS '未执行前过期时间';
COMMENT ON COLUMN public.device_instruction.dispatched_at IS '成功投递到设备/MQTT 时间';
COMMENT ON COLUMN public.device_instruction.executed_at IS '设备执行进入终态时间';
COMMENT ON COLUMN public.device_instruction.max_retry IS '最大重试次数';
COMMENT ON COLUMN public.device_instruction.merged_from_count IS '被新命令合并的旧 pending 数量';
COMMENT ON COLUMN public.device_instruction.schedule_id IS '若由定时任务生成，则关联 schedule';
