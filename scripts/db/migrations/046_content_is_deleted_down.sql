SET search_path TO public;

DROP INDEX IF EXISTS idx_content_list_alive;
ALTER TABLE public.content DROP COLUMN IF EXISTS is_deleted;
