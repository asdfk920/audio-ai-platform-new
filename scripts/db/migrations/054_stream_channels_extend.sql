-- 扩展 stream_channels 字段以支撑推流业务（与接口设计对齐）
SET search_path TO public;

ALTER TABLE public.stream_channels
  ADD COLUMN IF NOT EXISTS stream_type VARCHAR(16) NOT NULL DEFAULT 'live',
  ADD COLUMN IF NOT EXISTS push_url TEXT,
  ADD COLUMN IF NOT EXISTS auth_type VARCHAR(32) NOT NULL DEFAULT 'token',
  ADD COLUMN IF NOT EXISTS status SMALLINT NOT NULL DEFAULT 1;

COMMENT ON COLUMN public.stream_channels.stream_type IS '流类型（live/record等）';
COMMENT ON COLUMN public.stream_channels.push_url IS '推流地址（含鉴权 query）';
COMMENT ON COLUMN public.stream_channels.auth_type IS '鉴权类型（token/hmac等）';
COMMENT ON COLUMN public.stream_channels.status IS '通道状态（1激活/0禁用等）';

