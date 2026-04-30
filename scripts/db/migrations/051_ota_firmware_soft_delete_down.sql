SET search_path TO public;

DROP INDEX IF EXISTS public.uk_ota_firmware_product_version_active;
DROP INDEX IF EXISTS public.idx_ota_firmware_deleted_at;

-- 若存在多条同 product_key+version（含已软删），恢复全局唯一会失败，需人工处理数据后再执行
ALTER TABLE public.ota_firmware
  ADD CONSTRAINT uk_ota_firmware_product_version UNIQUE (product_key, version);

ALTER TABLE public.ota_firmware DROP COLUMN IF EXISTS backup_expires_at;
ALTER TABLE public.ota_firmware DROP COLUMN IF EXISTS backup_path;
ALTER TABLE public.ota_firmware DROP COLUMN IF EXISTS delete_reason;
ALTER TABLE public.ota_firmware DROP COLUMN IF EXISTS deleted_by;
ALTER TABLE public.ota_firmware DROP COLUMN IF EXISTS deleted_at;
