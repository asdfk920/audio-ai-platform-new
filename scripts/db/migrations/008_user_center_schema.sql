-- 用户中心设计对齐（AI 空间音频 ER）：在保留物理表 users / user_auth / roles / user_role_rel 的前提下，
-- 新增会员、验证码流水、登录审计、JWT 黑名单，并通过视图暴露 sys_user / sys_role / sys_user_role。
-- 说明：业务代码仍读写 users，避免大规模改 SQL；BI/文档可查询 sys_* 视图。

-- ---------------------------------------------------------------------------
-- 1) 第三方绑定表 user_auth：对齐 user_auth_third（identity = auth_type + auth_id）
-- ---------------------------------------------------------------------------
ALTER TABLE user_auth ADD COLUMN IF NOT EXISTS unionid VARCHAR(128);
ALTER TABLE user_auth ADD COLUMN IF NOT EXISTS nickname VARCHAR(128);
ALTER TABLE user_auth ADD COLUMN IF NOT EXISTS avatar TEXT;
ALTER TABLE user_auth ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_user_auth_user_id_auth_type ON user_auth(user_id, auth_type);

COMMENT ON TABLE user_auth IS '第三方登录绑定；auth_type=wechat|google 等，auth_id=openid，unionid 微信可选';
COMMENT ON COLUMN user_auth.unionid IS '微信 unionid，可选';

-- ---------------------------------------------------------------------------
-- 2) 验证码发送流水（与 Redis 短期验证码配合；可用于审计，非必填写入）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_verify_code (
    id BIGSERIAL PRIMARY KEY,
    target VARCHAR(255) NOT NULL,
    code VARCHAR(32) NOT NULL,
    scene VARCHAR(32) NOT NULL,
    expired_at TIMESTAMP NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_verify_code_target_scene ON user_verify_code(target, scene, created_at DESC);

COMMENT ON TABLE user_verify_code IS '验证码流水：scene 如 register/login/reset_pwd/bind；生产建议 code 存哈希或脱敏';
COMMENT ON COLUMN user_verify_code.scene IS 'register|login|reset_pwd|bind 等';

-- ---------------------------------------------------------------------------
-- 3) 会员 / 订阅（user_id 一对一扩展）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_member (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    level INT NOT NULL DEFAULT 0,
    expired_at TIMESTAMP NULL,
    status SMALLINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_member_level_status ON user_member(level, status);

COMMENT ON TABLE user_member IS '会员等级与有效期；无行表示默认免费档';
COMMENT ON COLUMN user_member.status IS '1有效 0暂停等，由业务定义';

-- ---------------------------------------------------------------------------
-- 4) 登录审计日志
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_login_log (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    login_type VARCHAR(32) NOT NULL,
    ip INET NULL,
    device VARCHAR(256),
    status SMALLINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_login_log_user_created ON user_login_log(user_id, created_at DESC);

COMMENT ON COLUMN user_login_log.login_type IS 'password|sms|wechat|google';
COMMENT ON COLUMN user_login_log.status IS '1成功 2失败';

-- ---------------------------------------------------------------------------
-- 5) JWT 黑名单（登出、强退；可与 Redis 并存）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS jwt_blacklist (
    id BIGSERIAL PRIMARY KEY,
    token TEXT NOT NULL,
    expired_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_jwt_blacklist_expired_at ON jwt_blacklist(expired_at);

COMMENT ON TABLE jwt_blacklist IS '失效 access/refresh 指纹；定期按 expired_at 清理';

-- ---------------------------------------------------------------------------
-- 6) 只读视图：sys_user / sys_role / sys_user_role（映射现有物理表）
-- 若库中已由 009 等创建过不同列集的同名视图，须先 DROP 再建，勿用 OR REPLACE（会报 cannot drop columns from view）
-- ---------------------------------------------------------------------------
DROP VIEW IF EXISTS sys_user_role CASCADE;
DROP VIEW IF EXISTS sys_user CASCADE;
DROP VIEW IF EXISTS sys_role CASCADE;

CREATE VIEW sys_user AS
SELECT
    u.id,
    u.nickname,
    u.mobile,
    u.email,
    u.password AS password_hash,
    u.salt AS password_salt,
    u.avatar,
    u.user_type::INT AS user_type,
    u.status::INT AS status,
    u.register_ip::TEXT AS register_ip,
    u.last_login_ip::TEXT AS last_login_ip,
    u.last_login_at AS last_login_time,
    u.created_at,
    u.updated_at,
    u.deleted_at
FROM users u;

COMMENT ON VIEW sys_user IS '逻辑用户主表视图；物理表为 public.users（含实名/注销等扩展列未全部展开，直接查 users 可得全量）';

CREATE VIEW sys_role AS
SELECT
    r.id,
    r.name AS role_name,
    COALESCE(r.permissions::TEXT, '')::VARCHAR(512) AS remark,
    1::INT AS status
FROM roles r;

COMMENT ON VIEW sys_role IS '逻辑角色视图；物理表为 public.roles';

CREATE VIEW sys_user_role AS
SELECT
    ur.id,
    ur.user_id,
    ur.role_id
FROM user_role_rel ur;

COMMENT ON VIEW sys_user_role IS '用户-角色关联视图；物理表为 public.user_role_rel';
