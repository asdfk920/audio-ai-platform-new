-- 回滚：流媒体相关表
SET search_path TO public;

DROP TABLE IF EXISTS public.stream_auth_configs;
DROP TABLE IF EXISTS public.stream_stats;
DROP TABLE IF EXISTS public.stream_logs;
DROP TABLE IF EXISTS public.play_quality_logs;
DROP TABLE IF EXISTS public.play_records;
DROP TABLE IF EXISTS public.cdn_nodes;
DROP TABLE IF EXISTS public.push_records;
DROP TABLE IF EXISTS public.stream_channels;

