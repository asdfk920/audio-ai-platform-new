-- 解绑状态统一为 0；历史 status=2 迁移为 0
SET search_path TO public;

ALTER TABLE public.user_device_bind DROP CONSTRAINT IF EXISTS user_device_bind_status_check;

UPDATE public.user_device_bind SET status = 0 WHERE status = 2;

ALTER TABLE public.user_device_bind
  ADD CONSTRAINT user_device_bind_status_check CHECK (status IN (0, 1));

COMMENT ON COLUMN public.user_device_bind.status IS '0=已解绑 1=绑定中';
