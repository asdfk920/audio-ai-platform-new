-- 设备后台可编辑扩展信息 + 管理员修改审计
SET search_path TO public;

ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS admin_location VARCHAR(256) NOT NULL DEFAULT '';
ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS admin_group_id VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS admin_tags JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS admin_config JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN public.device.admin_location IS '后台：设备位置';
COMMENT ON COLUMN public.device.admin_group_id IS '后台：设备分组标识';
COMMENT ON COLUMN public.device.admin_tags IS '后台：标签 JSON 数组';
COMMENT ON COLUMN public.device.admin_config IS '后台：扩展配置 JSON（上报间隔、音量等）';

CREATE TABLE IF NOT EXISTS public.device_admin_edit_log (
  id BIGSERIAL PRIMARY KEY,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  sn VARCHAR(64) NOT NULL,
  admin_user_id BIGINT NOT NULL DEFAULT 0,
  admin_account VARCHAR(128) NOT NULL DEFAULT '',
  before_data JSONB NOT NULL DEFAULT '{}'::jsonb,
  after_data JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_fields JSONB NOT NULL DEFAULT '[]'::jsonb,
  ip_address VARCHAR(64) NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_admin_edit_log_device ON public.device_admin_edit_log (device_id, created_at DESC);

COMMENT ON TABLE public.device_admin_edit_log IS '管理员修改设备扩展信息审计';
