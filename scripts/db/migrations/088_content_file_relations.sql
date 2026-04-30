-- 为 content 表添加文件关联字段
ALTER TABLE public.content ADD COLUMN IF NOT EXISTS audio_id BIGINT DEFAULT NULL;
ALTER TABLE public.content ADD COLUMN IF NOT EXISTS cover_id BIGINT DEFAULT NULL;

COMMENT ON COLUMN public.content.audio_id IS '关联音频文件ID（content_files表）';
COMMENT ON COLUMN public.content.cover_id IS '关联封面文件ID（content_files表）';
