-- 内容服务：目录/分类/下载/播放/收藏/标签（与 001 中 contents + content_play_records 流水线表并存）
-- 本套表面向「运营上架内容」场景：OSS/MinIO URL、会员等级、空间音频参数等

SET search_path TO public;

-- ---------------------------------------------------------------------------
-- 1) 内容分类
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.content_category (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    parent_id       BIGINT NULL REFERENCES public.content_category(id) ON DELETE SET NULL,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    status          SMALLINT NOT NULL DEFAULT 1,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT content_category_status_check CHECK (status IN (0, 1))
);

COMMENT ON TABLE public.content_category IS '内容分类；支持 parent_id 二级分类';
COMMENT ON COLUMN public.content_category.status IS '0 禁用 1 启用';
COMMENT ON COLUMN public.content_category.sort_order IS '排序权重，越小越靠前';

CREATE INDEX IF NOT EXISTS idx_content_category_parent_id ON public.content_category(parent_id);
CREATE INDEX IF NOT EXISTS idx_content_category_status_sort ON public.content_category(status, sort_order);

DROP TRIGGER IF EXISTS trg_content_category_set_updated_at ON public.content_category;
CREATE TRIGGER trg_content_category_set_updated_at
    BEFORE UPDATE ON public.content_category
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- ---------------------------------------------------------------------------
-- 2) 内容主表（目录元数据）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.content (
    id              BIGSERIAL PRIMARY KEY,
    title           VARCHAR(500) NOT NULL DEFAULT '',
    cover_url       VARCHAR(1000),
    audio_url       VARCHAR(2000),
    duration_sec    INTEGER NOT NULL DEFAULT 0,
    artist          VARCHAR(255),
    category_id     BIGINT NULL REFERENCES public.content_category(id) ON DELETE SET NULL,
    vip_level       SMALLINT NOT NULL DEFAULT 0,
    size_bytes      BIGINT,
    format          VARCHAR(32),
    spatial_params  JSONB,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    status          SMALLINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT content_status_check CHECK (status IN (0, 1, 2, 3)),
    CONSTRAINT content_vip_level_check CHECK (vip_level >= 0)
);

COMMENT ON TABLE public.content IS '内容元数据主表：上架音频/空间音频（与 public.contents 流水线表区分）';
COMMENT ON COLUMN public.content.vip_level IS '所需会员等级：0 免费，其余与业务会员档位对齐';
COMMENT ON COLUMN public.content.spatial_params IS '空间音频参数 JSON';
COMMENT ON COLUMN public.content.status IS '0 草稿 1 上架 2 下架 3 审核中';
COMMENT ON COLUMN public.content.sort_order IS '排序权重';
COMMENT ON COLUMN public.content.duration_sec IS '时长（秒）';
COMMENT ON COLUMN public.content.size_bytes IS '文件大小（字节）';

CREATE INDEX IF NOT EXISTS idx_content_category_id ON public.content(category_id);
CREATE INDEX IF NOT EXISTS idx_content_status_sort ON public.content(status, sort_order DESC);
CREATE INDEX IF NOT EXISTS idx_content_vip_level ON public.content(vip_level);
CREATE INDEX IF NOT EXISTS idx_content_created_at ON public.content(created_at DESC);

DROP TRIGGER IF EXISTS trg_content_set_updated_at ON public.content;
CREATE TRIGGER trg_content_set_updated_at
    BEFORE UPDATE ON public.content
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- ---------------------------------------------------------------------------
-- 3) 用户下载记录
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.user_download (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    content_id          BIGINT NOT NULL REFERENCES public.content(id) ON DELETE CASCADE,
    download_status     SMALLINT NOT NULL DEFAULT 0,
    progress_pct        SMALLINT,
    progress_bytes      BIGINT,
    download_count      INTEGER NOT NULL DEFAULT 1,
    started_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at         TIMESTAMP,
    created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_download_status_check CHECK (download_status IN (0, 1, 2, 3)),
    CONSTRAINT user_download_progress_pct_check CHECK (progress_pct IS NULL OR (progress_pct >= 0 AND progress_pct <= 100))
);

COMMENT ON TABLE public.user_download IS '用户下载记录；支持断点续传字段 progress_*';
COMMENT ON COLUMN public.user_download.download_status IS '0 进行中 1 完成 2 失败 3 暂停';

CREATE INDEX IF NOT EXISTS idx_user_download_user_id ON public.user_download(user_id);
CREATE INDEX IF NOT EXISTS idx_user_download_content_id ON public.user_download(content_id);
CREATE INDEX IF NOT EXISTS idx_user_download_status ON public.user_download(user_id, download_status);

DROP TRIGGER IF EXISTS trg_user_download_set_updated_at ON public.user_download;
CREATE TRIGGER trg_user_download_set_updated_at
    BEFORE UPDATE ON public.user_download
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- ---------------------------------------------------------------------------
-- 4) 用户播放记录（统计 / 热度；指向 content 目录表）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.user_play_record (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    content_id          BIGINT NOT NULL REFERENCES public.content(id) ON DELETE CASCADE,
    play_duration_sec   INTEGER NOT NULL DEFAULT 0,
    device_sn           VARCHAR(64),
    played_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE public.user_play_record IS '用户播放行为记录（内容服务统计）';
COMMENT ON COLUMN public.user_play_record.played_at IS '播放发生时间';

CREATE INDEX IF NOT EXISTS idx_user_play_record_user_played ON public.user_play_record(user_id, played_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_play_record_content ON public.user_play_record(content_id);
CREATE INDEX IF NOT EXISTS idx_user_play_record_played_at ON public.user_play_record(played_at DESC);

-- ---------------------------------------------------------------------------
-- 5) 内容收藏
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.user_favorite (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    content_id  BIGINT NOT NULL REFERENCES public.content(id) ON DELETE CASCADE,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_favorite_user_content_unique UNIQUE (user_id, content_id)
);

COMMENT ON TABLE public.user_favorite IS '用户收藏内容';

CREATE INDEX IF NOT EXISTS idx_user_favorite_user_id ON public.user_favorite(user_id);
CREATE INDEX IF NOT EXISTS idx_user_favorite_content_id ON public.user_favorite(content_id);

-- ---------------------------------------------------------------------------
-- 6) 标签与关联
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS public.content_tag (
    id          BIGSERIAL PRIMARY KEY,
    tag_name    VARCHAR(64) NOT NULL,
    status      SMALLINT NOT NULL DEFAULT 1,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT content_tag_status_check CHECK (status IN (0, 1))
);

COMMENT ON TABLE public.content_tag IS '内容标签';
COMMENT ON COLUMN public.content_tag.status IS '0 禁用 1 启用';

CREATE UNIQUE INDEX IF NOT EXISTS idx_content_tag_name_lower ON public.content_tag (lower(tag_name));

DROP TRIGGER IF EXISTS trg_content_tag_set_updated_at ON public.content_tag;
CREATE TRIGGER trg_content_tag_set_updated_at
    BEFORE UPDATE ON public.content_tag
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TABLE IF NOT EXISTS public.content_tag_relation (
    id          BIGSERIAL PRIMARY KEY,
    content_id  BIGINT NOT NULL REFERENCES public.content(id) ON DELETE CASCADE,
    tag_id      BIGINT NOT NULL REFERENCES public.content_tag(id) ON DELETE CASCADE,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT content_tag_relation_unique UNIQUE (content_id, tag_id)
);

COMMENT ON TABLE public.content_tag_relation IS '内容-标签多对多';

CREATE INDEX IF NOT EXISTS idx_content_tag_relation_tag_id ON public.content_tag_relation(tag_id);
