-- 设备歌曲下载记录表
-- 用于记录用户通过设备下载歌曲的完整生命周期
CREATE TABLE IF NOT EXISTS device_song_downloads (
    id SERIAL PRIMARY KEY,
    task_id VARCHAR(64) UNIQUE NOT NULL,                    -- 任务ID（UUID，用于追踪）
    user_id BIGINT NOT NULL,                                 -- 用户ID
    device_sn VARCHAR(16) NOT NULL,                         -- 目标设备序列号（16位）
    content_id BIGINT NOT NULL,                             -- 内容ID（对应content.content_catalog）
    song_name VARCHAR(200),                                 -- 歌曲名称
    download_url TEXT,                                      -- CDN下载地址
    status VARCHAR(20) DEFAULT 'pending',                   -- 状态：pending/downloading/success/failed/cancelled
    progress FLOAT DEFAULT 0,                               -- 进度百分比（0-100）

    -- 结果信息
    local_path VARCHAR(500),                                -- 设备本地存储路径
    file_size BIGINT DEFAULT 0,                             -- 文件大小（字节）
    error_msg TEXT,                                         -- 失败原因

    -- 时间信息
    duration_ms BIGINT DEFAULT 0,                           -- 下载耗时（毫秒）
    instruction_id BIGINT,                                  -- 设备服务指令ID
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,         -- 创建时间
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,         -- 最后更新时间
    finished_at TIMESTAMP                                   -- 完成时间
);

-- 索引优化查询性能
CREATE INDEX idx_dsd_task_id ON device_song_downloads(task_id);
CREATE INDEX idx_dsd_user_device ON device_song_downloads(user_id, device_sn);
CREATE INDEX idx_dsd_content ON device_song_downloads(content_id);
CREATE INDEX idx_dsd_status ON device_song_downloads(status);
CREATE INDEX idx_dsd_created_at ON device_song_downloads(created_at DESC);

-- 注释
COMMENT ON TABLE device_song_downloads IS '设备歌曲下载记录表';
COMMENT ON COLUMN device_song_downloads.id IS '主键ID';
COMMENT ON COLUMN device_song_downloads.task_id IS '任务唯一标识（UUID）';
COMMENT ON COLUMN device_song_downloads.user_id IS '发起下载的用户ID';
COMMENT ON COLUMN device_song_downloads.device_sn IS '目标设备序列号';
COMMENT ON COLUMN device_song_downloads.content_id IS '内容ID（关联content.content_catalog）';
COMMENT ON COLUMN device_song_downloads.song_name IS '歌曲名称';
COMMENT ON COLUMN device_song_downloads.download_url IS 'CDN下载地址';
COMMENT ON COLUMN device_song_downloads.status IS '状态：pending=待处理/downloading=下载中/success=成功/failed=失败/cancelled=已取消';
COMMENT ON COLUMN device_song_downloads.progress IS '下载进度百分比（0-100）';
COMMENT ON COLUMN device_song_downloads.local_path IS '设备端本地存储路径';
COMMENT ON COLUMN device_song_downloads.file_size IS '文件大小（字节）';
COMMENT ON COLUMN device_song_downloads.error_msg IS '错误信息（status=failed时必填）';
COMMENT ON COLUMN device_song_downloads.duration_ms IS '下载耗时（毫秒）';
COMMENT ON COLUMN device_song_downloads.instruction_id IS '指令ID';
COMMENT ON COLUMN device_song_downloads.created_at IS '创建时间';
COMMENT ON COLUMN device_song_downloads.updated_at IS '最后更新时间';
COMMENT ON COLUMN device_song_downloads.finished_at IS '完成时间';
