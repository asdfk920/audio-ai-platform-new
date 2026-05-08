-- Audio Separation Tasks Table
-- Stores information about user audio separation tasks

CREATE TABLE IF NOT EXISTS audio_separation_tasks (
    id BIGSERIAL PRIMARY KEY,
    task_id VARCHAR(100) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL,
    content_id BIGINT NOT NULL,
    audio_url VARCHAR(2000) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    progress INTEGER NOT NULL DEFAULT 0,
    message TEXT,
    result_url VARCHAR(2000),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for faster queries
CREATE INDEX IF NOT EXISTS idx_audio_tasks_task_id ON audio_separation_tasks(task_id);
CREATE INDEX IF NOT EXISTS idx_audio_tasks_user_id ON audio_separation_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_audio_tasks_status ON audio_separation_tasks(status);
CREATE INDEX IF NOT EXISTS idx_audio_tasks_created_at ON audio_separation_tasks(created_at);
