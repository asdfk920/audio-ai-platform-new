SET search_path TO public;

ALTER TABLE public.device_event_log DROP COLUMN IF EXISTS operator;

UPDATE public.device SET status = 2 WHERE status = 4;

ALTER TABLE public.device DROP CONSTRAINT IF EXISTS device_status_check;
ALTER TABLE public.device
  ADD CONSTRAINT device_status_check CHECK (status IN (1, 2, 3));

COMMENT ON COLUMN public.device.status IS '1=正常 2=禁用 3=未激活';

ALTER TABLE public.device DROP COLUMN IF EXISTS last_active_at;
