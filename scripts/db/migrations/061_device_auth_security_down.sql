SET search_path TO public;

DROP INDEX IF EXISTS idx_device_auth_audit_sn_time;
DROP TABLE IF EXISTS public.device_auth_audit;

DROP INDEX IF EXISTS idx_device_auth_failures_sn_time;
DROP TABLE IF EXISTS public.device_auth_failures;

ALTER TABLE public.device DROP COLUMN IF EXISTS last_auth_at;
ALTER TABLE public.device DROP COLUMN IF EXISTS activated_firmware_version;
ALTER TABLE public.device DROP COLUMN IF EXISTS activated_ip;
ALTER TABLE public.device DROP COLUMN IF EXISTS activated_at;
