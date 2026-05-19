SET search_path TO public;

ALTER TABLE public.device DROP COLUMN IF EXISTS register_timestamp;
ALTER TABLE public.device DROP COLUMN IF EXISTS register_signature;

-- 回滚约束：若库内已无 status=5 可执行；否则请先清理数据
ALTER TABLE public.device DROP CONSTRAINT IF EXISTS device_status_check;
ALTER TABLE public.device
  ADD CONSTRAINT device_status_check CHECK (status IN (1, 2, 3, 4));

COMMENT ON COLUMN public.device.status IS '1=正常 2=禁用 3=未激活 4=报废（回滚后语义）';
