-- Rollback 076_device_shadow_normalized
SET search_path TO public;

DROP TABLE IF EXISTS public.device_shadow_config CASCADE;
DROP TABLE IF EXISTS public.device_shadow_location CASCADE;
DROP TABLE IF EXISTS public.device_shadow_battery CASCADE;
DROP TABLE IF EXISTS public.device_shadow_profile CASCADE;
