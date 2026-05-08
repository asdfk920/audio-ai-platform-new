-- Audio Tracks Table
-- Stores separated audio tracks from AI separation tasks

CREATE TABLE IF NOT EXISTS audio_tracks (
    id BIGSERIAL PRIMARY KEY,
    task_id VARCHAR(100) NOT NULL,
    track_name VARCHAR(50) NOT NULL,
    track_url VARCHAR(2000) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    duration DOUBLE PRECISION NOT NULL DEFAULT 0,
    sample_rate INTEGER NOT NULL DEFAULT 44100,
    channels INTEGER NOT NULL DEFAULT 2,
    format VARCHAR(20) NOT NULL DEFAULT 'wav',
    order_index INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for faster queries
CREATE INDEX IF NOT EXISTS idx_audio_tracks_task_id ON audio_tracks(task_id);
CREATE INDEX IF NOT EXISTS idx_audio_tracks_track_name ON audio_tracks(track_name);

-- Add comments
COMMENT ON TABLE audio_tracks IS '音轨分离结果表 - 存储分离后的各个音轨';
COMMENT ON COLUMN audio_tracks.task_id IS '所属任务 ID';
COMMENT ON COLUMN audio_tracks.track_name IS '音轨名称（vocals/drums/bass/other）';
COMMENT ON COLUMN audio_tracks.track_url IS '音轨文件 URL';
COMMENT ON COLUMN audio_tracks.file_size IS '文件大小（字节）';
COMMENT ON COLUMN audio_tracks.duration IS '时长（秒）';
COMMENT ON COLUMN audio_tracks.sample_rate IS '采样率';
COMMENT ON COLUMN audio_tracks.channels IS '声道数';
COMMENT ON COLUMN audio_tracks.format IS '文件格式';
COMMENT ON COLUMN audio_tracks.order_index IS '排序索引';
