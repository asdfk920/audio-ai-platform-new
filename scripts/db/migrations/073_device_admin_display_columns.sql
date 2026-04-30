-- device 表后台展示字段（与 GORM 模型一致；与 067 同内容，供未跑过 067 的库补齐）
SET search_path TO public;

ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS admin_display_name VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS admin_remark TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN public.device.admin_display_name IS '后台展示用设备名称';
COMMENT ON COLUMN public.device.admin_remark IS '管理员备注';
