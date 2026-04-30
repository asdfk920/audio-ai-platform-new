-- 设备后台管理：扩展状态（含报废）、最后活跃时间、事件日志操作人
SET search_path TO public;

ALTER TABLE public.device ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMP NULL;

COMMENT ON COLUMN public.device.last_active_at IS '最后活跃/上报时间，用于离线判断';

ALTER TABLE public.device DROP CONSTRAINT IF EXISTS device_status_check;
ALTER TABLE public.device
  ADD CONSTRAINT device_status_check CHECK (status IN (1, 2, 3, 4));

COMMENT ON COLUMN public.device.status IS '1=正常 2=禁用 3=未激活 4=报废';

ALTER TABLE public.device_event_log ADD COLUMN IF NOT EXISTS operator VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN public.device_event_log.operator IS '操作者（管理员账号或系统）';
