-- device 表与 GORM 模型对齐：软删除 + 创建人/更新人（models.Device 含 DeletedAt / CreateBy / UpdateBy）
SET search_path TO public;
SET client_encoding TO 'UTF8';

ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS create_by BIGINT NOT NULL DEFAULT 0;

ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS update_by BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_device_deleted_at ON public.device (deleted_at);

COMMENT ON COLUMN public.device.deleted_at IS '软删除时间（GORM DeletedAt）';
COMMENT ON COLUMN public.device.create_by IS '创建人（后台用户 ID）';
COMMENT ON COLUMN public.device.update_by IS '更新人';
