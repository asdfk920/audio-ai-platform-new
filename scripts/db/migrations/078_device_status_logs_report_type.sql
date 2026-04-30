-- 状态上报类型：自动 / 手动 / 网络恢复补发
SET search_path TO public;

ALTER TABLE public.device_status_logs
  ADD COLUMN IF NOT EXISTS report_type VARCHAR(16) NOT NULL DEFAULT 'auto';

ALTER TABLE public.device_status_logs
  DROP CONSTRAINT IF EXISTS device_status_logs_report_type_check;

ALTER TABLE public.device_status_logs
  ADD CONSTRAINT device_status_logs_report_type_check
  CHECK (report_type IN ('auto', 'manual', 'sync'));

COMMENT ON COLUMN public.device_status_logs.report_type IS '上报类型：auto 定时、manual 手动、sync 补发';
