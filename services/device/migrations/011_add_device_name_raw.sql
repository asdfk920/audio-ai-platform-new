-- 设备表增加原始名称（PostgreSQL）
ALTER TABLE public.device
    ADD COLUMN IF NOT EXISTS device_name_raw VARCHAR(100) DEFAULT '';

COMMENT ON COLUMN public.device.device_name_raw IS '设备原始名称：出厂预设或设备注册时上报';

CREATE INDEX IF NOT EXISTS idx_device_name_raw ON public.device(device_name_raw);
