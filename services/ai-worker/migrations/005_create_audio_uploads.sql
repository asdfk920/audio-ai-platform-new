-- ============================================================
-- Audio Uploads Table - 用户上传音频记录表
-- 用于存储用户上传的音轨分离原始音频信息
-- 版本: 005
-- 日期: 2026-05-12
-- ============================================================

CREATE TABLE IF NOT EXISTS audio_uploads (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,                    -- 上传用户ID
    file_key VARCHAR(500) NOT NULL UNIQUE,      -- OSS/S3 存储路径键（唯一标识）
    file_url VARCHAR(2000) NOT NULL,            -- 完整访问URL
    original_filename VARCHAR(255) NOT NULL,    -- 原始文件名
    file_extension VARCHAR(20) NOT NULL,        -- 文件扩展名 (mp3/wav/flac/aac/ogg/m4a)
    file_size BIGINT NOT NULL DEFAULT 0,        -- 文件大小（字节）
    mime_type VARCHAR(100),                     -- MIME 类型 (audio/mpeg, audio/wav 等)
    storage_driver VARCHAR(20) NOT NULL DEFAULT 'oss',  -- 存储驱动: oss | s3 | local
    storage_bucket VARCHAR(255),                -- 存储桶名称

    -- 音频元数据（可选，后续可添加分析功能）
    duration_seconds FLOAT,                     -- 音频时长（秒）
    sample_rate INTEGER,                       -- 采样率 (Hz)
    channels INTEGER,                          -- 声道数 (1=单声道, 2=立体声)
    bitrate INTEGER,                           -- 比特率 (kbps)

    -- 关联关系
    content_id BIGINT,                         -- 关联的内容ID（如果已创建内容）
    task_id VARCHAR(100),                      -- 关联的分离任务ID（如果已发起分离）

    -- 状态管理
    status VARCHAR(20) NOT NULL DEFAULT 'uploaded',  -- 状态: uploaded(已上传) | processing(处理中) | completed(已完成) | failed(失败) | deleted(已删除)
    is_used BOOLEAN NOT NULL DEFAULT FALSE,   -- 是否已被使用（用于音轨分离）

    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE        -- 软删除时间
);

-- 创建索引以优化查询性能
CREATE INDEX IF NOT EXISTS idx_audio_uploads_user_id ON audio_uploads(user_id);
CREATE INDEX IF NOT EXISTS idx_audio_uploads_file_key ON audio_uploads(file_key);
CREATE INDEX IF NOT EXISTS idx_audio_uploads_status ON audio_uploads(status);
CREATE INDEX IF NOT EXISTS idx_audio_uploads_created_at ON audio_uploads(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audio_uploads_task_id ON audio_uploads(task_id);
CREATE INDEX IF NOT EXISTS idx_audio_uploads_content_id ON audio_uploads(content_id);

-- 添加约束：确保状态值合法
ALTER TABLE audio_uploads ADD CONSTRAINT chk_upload_status
    CHECK (status IN ('uploaded', 'processing', 'completed', 'failed', 'deleted'));

-- 添加注释
COMMENT ON TABLE audio_uploads IS '用户上传音频记录表 - 存储用户上传用于音轨分离的原始音频文件信息';
COMMENT ON COLUMN audio_uploads.id IS '主键ID（自增）';
COMMENT ON COLUMN audio_uploads.user_id IS '上传用户ID（关联 users 表）';
COMMENT ON COLUMN audio_uploads.file_key IS '对象存储路径键（格式: audio/{user_id}/{date}/{uuid}_{filename}.ext）';
COMMENT ON COLUMN audio_uploads.file_url IS '完整可访问URL（OSS/S3/本地地址）';
COMMENT ON COLUMN audio_uploads.original_filename IS '用户上传时的原始文件名';
COMMENT ON COLUMN audio_uploads.file_extension IS '文件扩展名（小写）';
COMMENT ON COLUMN audio_uploads.file_size IS '文件大小（字节）';
COMMENT ON COLUMN audio_uploads.mime_type IS 'MIME媒体类型';
COMMENT ON COLUMN audio_uploads.storage_driver IS '存储后端驱动: oss(阿里云) | s3(AWS/MinIO) | local(本地)';
COMMENT ON COLUMN audio_uploads.storage_bucket IS '存储桶/容器名称';
COMMENT ON COLUMN audio_uploads.duration_seconds IS '音频时长（秒），可选字段';
COMMENT ON COLUMN audio_uploads.sample_rate IS '采样率（Hz），如 44100、48000';
COMMENT ON COLUMN audio_uploads.channels IS '声道数: 1=单声道, 2=立体声';
COMMENT ON COLUMN audio_uploads.bitrate IS '比特率（kbps）';
COMMENT ON COLUMN audio_uploads.content_id IS '关联的内容记录ID（创建内容后回填）';
COMMENT ON COLUMN audio_uploads.task_id IS '关联的音轨分离任务ID（发起分离后回填）';
COMMENT ON COLUMN audio_uploads.status IS '当前状态: uploaded=已上传待用 | processing=正在处理 | completed=分离完成 | failed=处理失败 | deleted=已删除';
COMMENT ON COLUMN audio_uploads.is_used IS '是否已被使用于音轨分离任务';
COMMENT ON COLUMN audio_uploads.created_at IS '创建时间（上传完成时间）';
COMMENT ON COLUMN audio_uploads.updated_at IS '最后更新时间';
COMMENT ON COLUMN audio_uploads.deleted_at IS '软删除时间（NULL表示未删除）';

-- 创建触发器自动更新 updated_at 字段
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_audio_uploads_updated_at
    BEFORE UPDATE ON audio_uploads
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
