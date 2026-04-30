CREATE TABLE IF NOT EXISTS user_real_name (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    real_name VARCHAR(64) NOT NULL,
    id_card VARCHAR(255) NOT NULL,
    status INT NOT NULL DEFAULT 0,
    submit_time BIGINT,
    audit_time BIGINT,
    audit_admin_id INT,
    audit_remark VARCHAR(255),
    created_by INT,
    updated_by INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_real_name_user_id ON user_real_name(user_id);
CREATE INDEX IF NOT EXISTS idx_user_real_name_status ON user_real_name(status);
