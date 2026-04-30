SET search_path TO public;

ALTER TABLE public.ota_firmware
  ADD COLUMN IF NOT EXISTS sha256 VARCHAR(128) NOT NULL DEFAULT '';

COMMENT ON COLUMN public.ota_firmware.sha256 IS '固件包 SHA256 校验';
