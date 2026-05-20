SET search_path TO public;

ALTER TABLE public.content DROP COLUMN IF EXISTS today_play_count;
ALTER TABLE public.content DROP COLUMN IF EXISTS play_count;
