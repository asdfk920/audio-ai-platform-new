-- 回滚：已解绑恢复为 status=2
SET search_path TO public;

ALTER TABLE public.user_device_bind DROP CONSTRAINT IF EXISTS user_device_bind_status_check;

UPDATE public.user_device_bind SET status = 2 WHERE status = 0;

ALTER TABLE public.user_device_bind
  ADD CONSTRAINT user_device_bind_status_check CHECK (status IN (1, 2));

COMMENT ON COLUMN public.user_device_bind.status IS '1=绑定中 2=已解绑';
