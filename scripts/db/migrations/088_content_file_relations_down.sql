-- 回滚：删除 content 表的文件关联字段
ALTER TABLE public.content DROP COLUMN IF EXISTS audio_id;
ALTER TABLE public.content DROP COLUMN IF EXISTS cover_id;
