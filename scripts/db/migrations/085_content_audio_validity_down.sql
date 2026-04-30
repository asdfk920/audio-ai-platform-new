SET search_path TO public;

DROP INDEX IF EXISTS public.idx_content_audio_valid_until;
ALTER TABLE public.content DROP COLUMN IF EXISTS audio_valid_until;
ALTER TABLE public.content DROP COLUMN IF EXISTS audio_valid_from;
