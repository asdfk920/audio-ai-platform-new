-- 私有格式 / 对象存储元数据备份（可与 content 上传接口写入配合）
ALTER TABLE public.content_files
    ADD COLUMN IF NOT EXISTS storage_driver VARCHAR(32),
    ADD COLUMN IF NOT EXISTS object_key VARCHAR(512),
    ADD COLUMN IF NOT EXISTS format_tag VARCHAR(128),
    ADD COLUMN IF NOT EXISTS duration_seconds DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS file_md5 VARCHAR(64);

COMMENT ON COLUMN public.content_files.storage_driver IS '对象存储驱动：local|s3|oss';
COMMENT ON COLUMN public.content_files.object_key IS '对象存储内路径（不含 CDN 前缀）';
COMMENT ON COLUMN public.content_files.format_tag IS '私有格式标识，如 spatial_audio_v1';
COMMENT ON COLUMN public.content_files.duration_seconds IS '音频时长（秒），可选';
COMMENT ON COLUMN public.content_files.file_md5 IS '上传端提供的整文件 MD5（hex）';
