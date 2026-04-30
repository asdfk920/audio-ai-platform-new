-- 用户下载记录表
CREATE TABLE IF NOT EXISTS user_download_records (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    content_id BIGINT NOT NULL,
    file_id BIGINT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    downloaded_size BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    local_path VARCHAR(500) DEFAULT '',
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_user_download_records_user_id ON user_download_records(user_id);
CREATE INDEX IF NOT EXISTS idx_user_download_records_status ON user_download_records(status);
CREATE INDEX IF NOT EXISTS idx_user_download_records_created_at ON user_download_records(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_download_records_user_status ON user_download_records(user_id, status);

-- 注释
COMMENT ON TABLE user_download_records IS '用户下载记录表';
COMMENT ON COLUMN user_download_records.id IS '主键ID';
COMMENT ON COLUMN user_download_records.user_id IS '用户ID';
COMMENT ON COLUMN user_download_records.content_id IS '内容ID';
COMMENT ON COLUMN user_download_records.file_id IS '文件ID';
COMMENT ON COLUMN user_download_records.file_name IS '文件名';
COMMENT ON COLUMN user_download_records.file_size IS '文件大小（字节）';
COMMENT ON COLUMN user_download_records.downloaded_size IS '已下载大小（字节）';
COMMENT ON COLUMN user_download_records.status IS '状态：pending=未开始, downloading=下载中, completed=已完成, paused=已暂停, cancelled=已取消';
COMMENT ON COLUMN user_download_records.local_path IS '本地文件路径';
COMMENT ON COLUMN user_download_records.is_deleted IS '是否删除';
COMMENT ON COLUMN user_download_records.created_at IS '创建时间';
COMMENT ON COLUMN user_download_records.updated_at IS '更新时间';
