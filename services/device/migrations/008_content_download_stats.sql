-- 内容下载统计表
-- 用于统计每个内容的下载次数（总下载、今日下载、本周下载）
-- 并记录最后一次下载的详细状态（成功/失败/下载中）
CREATE TABLE IF NOT EXISTS content_download_stats (
    id SERIAL PRIMARY KEY,
    content_id BIGINT UNIQUE NOT NULL,                     -- 内容ID（对应content.content_catalog）

    -- 下载次数统计
    total_downloads BIGINT DEFAULT 0,                      -- 总下载次数
    today_downloads BIGINT DEFAULT 0,                      -- 今日下载次数
    week_downloads BIGINT DEFAULT 0,                       -- 本周下载次数

    -- 最后一次下载详情（回调接口更新）
    last_status VARCHAR(20) DEFAULT '',                    -- 最后状态: downloading/success/failed
    last_error_msg TEXT DEFAULT '',                        -- 最后错误信息（失败时记录原因）
    last_local_path VARCHAR(500) DEFAULT '',               -- 最后本地存储路径（成功时记录）
    last_file_size BIGINT DEFAULT 0,                       -- 最后文件大小（字节，成功时记录）
    last_duration_ms BIGINT DEFAULT 0,                     -- 最后下载耗时（毫秒）

    last_download_at TIMESTAMP,                            -- 最后一次下载时间

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,        -- 创建时间
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP         -- 最后更新时间
);

-- 索引优化查询性能
CREATE INDEX idx_cds_content_id ON content_download_stats(content_id);
CREATE INDEX idx_cds_total_downloads ON content_download_stats(total_downloads DESC);
CREATE INDEX idx_cds_last_download ON content_download_stats(last_download_at DESC);
CREATE INDEX idx_cds_last_status ON content_download_stats(last_status);

-- 注释
COMMENT ON TABLE content_download_stats IS '内容下载统计表';
COMMENT ON COLUMN content_download_stats.id IS '主键ID';
COMMENT ON COLUMN content_download_stats.content_id IS '内容ID（关联content.content_catalog）';
COMMENT ON COLUMN content_download_stats.total_downloads IS '累计总下载次数';
COMMENT ON COLUMN content_download_stats.today_downloads IS '今日下载次数';
COMMENT ON COLUMN content_download_stats.week_downloads IS '本周下载次数';
COMMENT ON COLUMN content_download_stats.last_status IS '最后下载状态: downloading=下载中/success=成功/failed=失败';
COMMENT ON COLUMN content_download_stats.last_error_msg IS '最后错误信息（last_status=failed时必填）';
COMMENT ON COLUMN content_download_stats.last_local_path IS '最后本地存储路径（last_status=success时有值）';
COMMENT ON COLUMN content_download_stats.last_file_size IS '最后文件大小（字节）';
COMMENT ON COLUMN content_download_stats.last_duration_ms IS '最后下载耗时（毫秒）';
COMMENT ON COLUMN content_download_stats.last_download_at IS '最后一次下载时间';
COMMENT ON COLUMN content_download_stats.created_at IS '创建时间';
COMMENT ON COLUMN content_download_stats.updated_at IS '最后更新时间';
