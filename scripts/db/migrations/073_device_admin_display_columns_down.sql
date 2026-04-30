SET search_path TO public;

ALTER TABLE public.device DROP COLUMN IF EXISTS admin_remark;
ALTER TABLE public.device DROP COLUMN IF EXISTS admin_display_name;
