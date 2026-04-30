-- 创建 content_files 表存储加密文件信息
CREATE TABLE IF NOT EXISTS public.content_files (
    id SERIAL PRIMARY KEY,
    url VARCHAR(500) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    file_type SMALLINT NOT NULL DEFAULT 1,
    original_name VARCHAR(255) NOT NULL,
    original_size BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_file_type CHECK (file_type IN (1, 2, 3, 4))
);

COMMENT ON TABLE public.content_files IS '内容文件表（存储加密后的私有格式文件信息）';
COMMENT ON COLUMN public.content_files.id IS '主键ID';
COMMENT ON COLUMN public.content_files.url IS '文件访问地址';
COMMENT ON COLUMN public.content_files.key_hash IS '密钥bcrypt哈希';
COMMENT ON COLUMN public.content_files.file_type IS '文件类型：1音频 2视频 3图片 4封面';
COMMENT ON COLUMN public.content_files.original_name IS '原始文件名';
COMMENT ON COLUMN public.content_files.original_size IS '原始文件大小（字节）';
COMMENT ON COLUMN public.content_files.created_at IS '创建时间';
