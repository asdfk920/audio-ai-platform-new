-- 回滚：stream_channels 扩展字段
SET search_path TO public;

ALTER TABLE public.stream_channels
  DROP COLUMN IF EXISTS status,
  DROP COLUMN IF EXISTS auth_type,
  DROP COLUMN IF EXISTS push_url,
  DROP COLUMN IF EXISTS stream_type;

