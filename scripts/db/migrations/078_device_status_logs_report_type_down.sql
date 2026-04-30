SET search_path TO public;

ALTER TABLE public.device_status_logs
  DROP CONSTRAINT IF EXISTS device_status_logs_report_type_check;

ALTER TABLE public.device_status_logs
  DROP COLUMN IF EXISTS report_type;
