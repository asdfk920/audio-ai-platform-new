-- 固件管理列表扩展：版本码、适用型号、下载次数、启用状态、创建人等
SET search_path TO public;

ALTER TABLE public.ota_firmware
  ADD COLUMN IF NOT EXISTS version_code INT NOT NULL DEFAULT 0;

ALTER TABLE public.ota_firmware
  ADD COLUMN IF NOT EXISTS min_sys_version VARCHAR(64) NOT NULL DEFAULT '';

ALTER TABLE public.ota_firmware
  ADD COLUMN IF NOT EXISTS device_models TEXT NOT NULL DEFAULT '';

ALTER TABLE public.ota_firmware
  ADD COLUMN IF NOT EXISTS download_count BIGINT NOT NULL DEFAULT 0;

ALTER TABLE public.ota_firmware
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE public.ota_firmware
  ADD COLUMN IF NOT EXISTS creator VARCHAR(128) NOT NULL DEFAULT '';

-- 1=启用 2=禁用（管理端筛选）
ALTER TABLE public.ota_firmware
  ADD COLUMN IF NOT EXISTS fw_status SMALLINT NOT NULL DEFAULT 1;

ALTER TABLE public.ota_firmware DROP CONSTRAINT IF EXISTS ota_firmware_fw_status_check;
ALTER TABLE public.ota_firmware
  ADD CONSTRAINT ota_firmware_fw_status_check CHECK (fw_status IN (1, 2));

COMMENT ON COLUMN public.ota_firmware.version_code IS '整型版本码，如 1002003';
COMMENT ON COLUMN public.ota_firmware.min_sys_version IS '最低系统/固件基线要求';
COMMENT ON COLUMN public.ota_firmware.device_models IS '适用设备型号，逗号分隔或 JSON';
COMMENT ON COLUMN public.ota_firmware.download_count IS '下载/拉取次数';
COMMENT ON COLUMN public.ota_firmware.fw_status IS '1=启用 2=禁用';
COMMENT ON COLUMN public.ota_firmware.creator IS '上传人';

DROP TRIGGER IF EXISTS trg_ota_firmware_set_updated_at ON public.ota_firmware;
CREATE TRIGGER trg_ota_firmware_set_updated_at
BEFORE UPDATE ON public.ota_firmware
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE INDEX IF NOT EXISTS idx_ota_firmware_created_at ON public.ota_firmware (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ota_firmware_fw_status ON public.ota_firmware (fw_status);
