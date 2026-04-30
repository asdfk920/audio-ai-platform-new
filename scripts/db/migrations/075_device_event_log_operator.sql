-- 与 040 对齐：部分环境未执行 040 时补齐 operator，避免 INSERT ... operator 失败
SET search_path TO public;

ALTER TABLE public.device_event_log ADD COLUMN IF NOT EXISTS operator VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN public.device_event_log.operator IS '操作者（管理员账号或系统）';
