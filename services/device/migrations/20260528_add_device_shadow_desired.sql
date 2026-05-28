-- 补齐 device_shadow.desired（部分环境仅有 reported，缺列会导致查询/写入失败）
ALTER TABLE public.device_shadow ADD COLUMN IF NOT EXISTS desired JSONB DEFAULT '{}'::jsonb;
