SET search_path TO public;

DROP TRIGGER IF EXISTS trg_ota_firmware_set_updated_at ON public.ota_firmware;

DROP INDEX IF EXISTS idx_ota_firmware_fw_status;
DROP INDEX IF EXISTS idx_ota_firmware_created_at;

ALTER TABLE public.ota_firmware DROP CONSTRAINT IF EXISTS ota_firmware_fw_status_check;

ALTER TABLE public.ota_firmware DROP COLUMN IF EXISTS creator;
ALTER TABLE public.ota_firmware DROP COLUMN IF EXISTS fw_status;
ALTER TABLE public.ota_firmware DROP COLUMN IF EXISTS updated_at;
ALTER TABLE public.ota_firmware DROP COLUMN IF EXISTS download_count;
ALTER TABLE public.ota_firmware DROP COLUMN IF EXISTS device_models;
ALTER TABLE public.ota_firmware DROP COLUMN IF EXISTS min_sys_version;
ALTER TABLE public.ota_firmware DROP COLUMN IF EXISTS version_code;
