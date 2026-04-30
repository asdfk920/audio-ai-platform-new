-- 账号注销：冷静期 + 流水表（执行后为逻辑注销，释放联系方式）

ALTER TABLE users ADD COLUMN IF NOT EXISTS cancellation_cooling_until TIMESTAMPTZ NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS account_cancelled_at TIMESTAMPTZ NULL;

COMMENT ON COLUMN users.cancellation_cooling_until IS '注销冷静期结束时间；非空且未到期时禁止登录';
COMMENT ON COLUMN users.account_cancelled_at IS '逻辑注销完成时间';

CREATE TABLE IF NOT EXISTS user_cancellation_log (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT,
    agreement_signed_at TIMESTAMPTZ NOT NULL,
    status SMALLINT NOT NULL,
    cooling_end_at TIMESTAMPTZ NOT NULL,
    applied_ip VARCHAR(45),
    device_info VARCHAR(512),
    audit_admin_id VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_cancellation_log_user_id ON user_cancellation_log(user_id);
CREATE INDEX IF NOT EXISTS idx_user_cancellation_log_due
    ON user_cancellation_log(status, cooling_end_at)
    WHERE status = 1;

COMMENT ON TABLE user_cancellation_log IS '注销申请流水：1冷静中 2已执行 3已撤销';
COMMENT ON COLUMN user_cancellation_log.status IS '1冷静期 2已注销 3已撤销';
