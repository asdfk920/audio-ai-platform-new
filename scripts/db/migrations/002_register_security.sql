-- 注册安全：密码算法字段 + 注册行为审计表
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_algo VARCHAR(32);
UPDATE users SET password_algo = 'bcrypt_concat' WHERE password IS NOT NULL AND (password_algo IS NULL OR password_algo = '');

CREATE TABLE IF NOT EXISTS user_register_events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(32) NOT NULL DEFAULT 'register',
    outcome VARCHAR(32) NOT NULL,
    email VARCHAR(255),
    mobile VARCHAR(20),
    client_ip VARCHAR(128),
    user_agent TEXT,
    device_id VARCHAR(128),
    err_msg TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_register_events_created ON user_register_events(created_at);
CREATE INDEX IF NOT EXISTS idx_user_register_events_ip ON user_register_events(client_ip);
