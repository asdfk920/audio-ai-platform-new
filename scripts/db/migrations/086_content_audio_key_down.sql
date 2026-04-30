-- 回滚：删除 content 表的 audio_key 字段
ALTER TABLE public.content DROP COLUMN IF EXISTS audio_key;
