-- 回滚 045：内容服务目录相关表

SET search_path TO public;

DROP TABLE IF EXISTS public.content_tag_relation CASCADE;
DROP TABLE IF EXISTS public.content_tag CASCADE;
DROP TABLE IF EXISTS public.user_favorite CASCADE;
DROP TABLE IF EXISTS public.user_play_record CASCADE;
DROP TABLE IF EXISTS public.user_download CASCADE;
DROP TABLE IF EXISTS public.content CASCADE;
DROP TABLE IF EXISTS public.content_category CASCADE;
