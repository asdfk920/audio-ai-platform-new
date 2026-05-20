-- 内容主表播放量字段（列表排序 sort=2、详情 ViewCount、流媒体计数依赖）
SET search_path TO public;

ALTER TABLE public.content ADD COLUMN IF NOT EXISTS play_count BIGINT NOT NULL DEFAULT 0;
COMMENT ON COLUMN public.content.play_count IS '累计播放量（列表「最热」排序使用）';

ALTER TABLE public.content ADD COLUMN IF NOT EXISTS today_play_count BIGINT NOT NULL DEFAULT 0;
COMMENT ON COLUMN public.content.today_play_count IS '当日播放增量（流媒体上报时递增，可按业务定时清零）';
