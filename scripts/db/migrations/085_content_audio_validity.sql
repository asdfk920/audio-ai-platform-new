-- 内容音频可播放/展示的有效期（可选；均为 NULL 表示不限制）
SET search_path TO public;

ALTER TABLE public.content ADD COLUMN IF NOT EXISTS audio_valid_from TIMESTAMPTZ NULL;
ALTER TABLE public.content ADD COLUMN IF NOT EXISTS audio_valid_until TIMESTAMPTZ NULL;

COMMENT ON COLUMN public.content.audio_valid_from IS '音频有效期起（含）；NULL 表示不限制起始';
COMMENT ON COLUMN public.content.audio_valid_until IS '音频有效期止（含业务语义上通常校验「当前时间 <= until」）；NULL 表示不限制结束';

CREATE INDEX IF NOT EXISTS idx_content_audio_valid_until ON public.content (audio_valid_until)
    WHERE is_deleted = 0 AND audio_valid_until IS NOT NULL;
