SET search_path TO public;

DROP TABLE IF EXISTS public.device_admin_edit_log;

ALTER TABLE public.device DROP COLUMN IF EXISTS admin_config;
ALTER TABLE public.device DROP COLUMN IF EXISTS admin_tags;
ALTER TABLE public.device DROP COLUMN IF EXISTS admin_group_id;
ALTER TABLE public.device DROP COLUMN IF EXISTS admin_location;
