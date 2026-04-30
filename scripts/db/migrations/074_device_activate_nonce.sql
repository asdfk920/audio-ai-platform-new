-- 设备云端认证激活：防随机数重放（与 go-admin POST .../activate-cloud 配套）
CREATE TABLE IF NOT EXISTS device_activate_nonce (
    id BIGSERIAL PRIMARY KEY,
    sn VARCHAR(64) NOT NULL,
    nonce VARCHAR(256) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_device_activate_sn_nonce UNIQUE (sn, nonce)
);
CREATE INDEX IF NOT EXISTS idx_device_activate_nonce_created ON device_activate_nonce (created_at);
