-- 实名认证：主表状态 + 历史审核流水（证件号密文存储）

ALTER TABLE users ADD COLUMN IF NOT EXISTS real_name_status SMALLINT NOT NULL DEFAULT 0;
-- 0 未认证 1 已通过 2 审核中 3 认证失败
ALTER TABLE users ADD COLUMN IF NOT EXISTS real_name_time TIMESTAMP NULL;
-- 1 个人身份证 2 企业统一社会信用代码
ALTER TABLE users ADD COLUMN IF NOT EXISTS real_name_type SMALLINT NULL;

COMMENT ON COLUMN users.real_name_status IS '实名状态: 0未认证 1已通过 2审核中 3失败';
COMMENT ON COLUMN users.real_name_time IS '最近一次实名通过时间';
COMMENT ON COLUMN users.real_name_type IS '证件类型: 1个人 2企业';

CREATE INDEX IF NOT EXISTS idx_users_real_name_status ON users(real_name_status) WHERE real_name_status <> 0;

CREATE TABLE IF NOT EXISTS user_real_name_auth (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cert_type SMALLINT NOT NULL,
    real_name_masked VARCHAR(64) NOT NULL,
    id_number_encrypted TEXT NOT NULL,
    id_number_last4 VARCHAR(8) NOT NULL,
    id_photo_ref TEXT,
    face_data_ref TEXT,
    auth_status SMALLINT NOT NULL,
    third_party_flow_no VARCHAR(128),
    third_party_channel VARCHAR(64),
    third_party_raw_response TEXT,
    fail_reason TEXT,
    reviewer_note TEXT,
    reviewed_at TIMESTAMP NULL,
    reviewed_by VARCHAR(64),
    device_info VARCHAR(512),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_real_name_auth_user_id ON user_real_name_auth(user_id);
CREATE INDEX IF NOT EXISTS idx_user_real_name_auth_status ON user_real_name_auth(auth_status);

COMMENT ON TABLE user_real_name_auth IS '实名认证提交与审核流水';
COMMENT ON COLUMN user_real_name_auth.auth_status IS '10待三方 11三方通过 12三方拒绝 20待人工 21人工通过 22人工拒绝 30已取消(接口异常回滚)';
