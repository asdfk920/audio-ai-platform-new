-- 设备认证失败记录表（防暴力破解）
CREATE TABLE IF NOT EXISTS device_auth_failures (
    id BIGSERIAL PRIMARY KEY,
    sn VARCHAR(64) NOT NULL,
    reason VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 索引：加速查询某设备的失败记录
CREATE INDEX IF NOT EXISTS idx_auth_failures_sn_time 
ON device_auth_failures(sn, created_at);

-- 注释
COMMENT ON TABLE device_auth_failures IS '设备认证失败记录表';
COMMENT ON COLUMN device_auth_failures.sn IS '设备序列号';
COMMENT ON COLUMN device_auth_failures.reason IS '失败原因 device_not_found/secret_mismatch/device_disabled';
COMMENT ON COLUMN device_auth_failures.created_at IS '失败时间';
