-- 用户侧设备绑定：扩展展示字段 + 活跃绑定全局唯一（一设备同时仅允许一个用户绑定中）
SET search_path TO public;

ALTER TABLE public.user_device_bind
  ADD COLUMN IF NOT EXISTS device_name VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS device_model VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS system_version VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN public.user_device_bind.status IS '1=绑定中 2=已解绑';

-- 仅对「绑定中」记录约束：同一 device_id 只能有一条 status=1
CREATE UNIQUE INDEX IF NOT EXISTS uk_user_device_bind_device_active
  ON public.user_device_bind (device_id)
  WHERE status = 1;
