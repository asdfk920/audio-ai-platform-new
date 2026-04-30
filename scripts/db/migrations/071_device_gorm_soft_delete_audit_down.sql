SET search_path TO public;

DROP INDEX IF EXISTS idx_device_deleted_at;

ALTER TABLE public.device DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE public.device DROP COLUMN IF EXISTS update_by;
ALTER TABLE public.device DROP COLUMN IF EXISTS create_by;
