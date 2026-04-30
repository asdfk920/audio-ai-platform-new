SET search_path TO public;

ALTER TABLE public.ota_firmware DROP COLUMN IF EXISTS sha256;
