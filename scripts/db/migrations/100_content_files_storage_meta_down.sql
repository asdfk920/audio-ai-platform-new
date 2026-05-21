ALTER TABLE public.content_files DROP COLUMN IF EXISTS file_md5;
ALTER TABLE public.content_files DROP COLUMN IF EXISTS duration_seconds;
ALTER TABLE public.content_files DROP COLUMN IF EXISTS format_tag;
ALTER TABLE public.content_files DROP COLUMN IF EXISTS object_key;
ALTER TABLE public.content_files DROP COLUMN IF EXISTS storage_driver;
