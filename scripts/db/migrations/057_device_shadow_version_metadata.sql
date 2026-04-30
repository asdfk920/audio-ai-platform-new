SET search_path TO public;

ALTER TABLE public.device_shadow
  ADD COLUMN IF NOT EXISTS version BIGINT NOT NULL DEFAULT 0;

ALTER TABLE public.device_shadow
  ADD COLUMN IF NOT EXISTS metadata JSONB;

COMMENT ON COLUMN public.device_shadow.version IS '影子版本号，每次 reported/desired 更新递增';
COMMENT ON COLUMN public.device_shadow.metadata IS '影子元数据（字段级时间戳等）';
