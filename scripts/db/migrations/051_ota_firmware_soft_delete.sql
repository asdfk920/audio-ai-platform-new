SET search_path TO public;

-- 软删除与备份元数据
ALTER TABLE public.ota_firmware
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP NULL;
ALTER TABLE public.ota_firmware
  ADD COLUMN IF NOT EXISTS deleted_by VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE public.ota_firmware
  ADD COLUMN IF NOT EXISTS delete_reason TEXT NULL;
ALTER TABLE public.ota_firmware
  ADD COLUMN IF NOT EXISTS backup_path VARCHAR(512) NOT NULL DEFAULT '';
ALTER TABLE public.ota_firmware
  ADD COLUMN IF NOT EXISTS backup_expires_at TIMESTAMP NULL;

COMMENT ON COLUMN public.ota_firmware.deleted_at IS '软删除时间，非空表示已删除';
COMMENT ON COLUMN public.ota_firmware.deleted_by IS '删除操作人';
COMMENT ON COLUMN public.ota_firmware.delete_reason IS '删除原因';
COMMENT ON COLUMN public.ota_firmware.backup_path IS '固件文件备份相对路径';
COMMENT ON COLUMN public.ota_firmware.backup_expires_at IS '备份文件建议清理时间';

-- 允许同 product_key+version 在软删后重新上传：仅对未删除行唯一
ALTER TABLE public.ota_firmware DROP CONSTRAINT IF EXISTS uk_ota_firmware_product_version;

CREATE UNIQUE INDEX IF NOT EXISTS uk_ota_firmware_product_version_active
  ON public.ota_firmware (product_key, version)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ota_firmware_deleted_at ON public.ota_firmware (deleted_at DESC);
