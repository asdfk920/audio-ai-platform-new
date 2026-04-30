-- 内容软删：列表/鉴权需 is_deleted = 0
SET search_path TO public;

ALTER TABLE public.content ADD COLUMN IF NOT EXISTS is_deleted SMALLINT NOT NULL DEFAULT 0;
COMMENT ON COLUMN public.content.is_deleted IS '0 未删除 1 已删除（软删）';

CREATE INDEX IF NOT EXISTS idx_content_list_alive ON public.content (status, is_deleted, vip_level)
    WHERE is_deleted = 0 AND status = 1;
