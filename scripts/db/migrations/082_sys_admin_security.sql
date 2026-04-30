-- sys_admin 安全与强制改密扩展字段
-- · allowed_ips                 逗号分隔的 IP / CIDR 白名单；NULL 或 '' 表示不限制
-- · allowed_login_start/end     允许登录时间窗（24h 本地时间）；NULL 表示不限制；若仅一方为 NULL 也视作不限制
-- · must_change_password        true 表示用户下次登录后仍需立即修改密码（首次登录或管理员重置后置 true）
-- · last_password_changed_at    最近一次改密时间（与 password_changed_at 语义相同；此处冗余以避免读写老字段时的歧义）
SET search_path TO public;

ALTER TABLE IF EXISTS sys_admin
    ADD COLUMN IF NOT EXISTS allowed_ips TEXT,
    ADD COLUMN IF NOT EXISTS allowed_login_start VARCHAR(8),
    ADD COLUMN IF NOT EXISTS allowed_login_end   VARCHAR(8),
    ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS last_password_changed_at TIMESTAMPTZ;

COMMENT ON COLUMN sys_admin.allowed_ips IS '允许登录的 IP/CIDR 白名单，逗号分隔；NULL 或空串表示不限制';
COMMENT ON COLUMN sys_admin.allowed_login_start IS '允许登录时间窗起点（本地时间）；NULL 表示不限制';
COMMENT ON COLUMN sys_admin.allowed_login_end   IS '允许登录时间窗终点（本地时间）；NULL 表示不限制';
COMMENT ON COLUMN sys_admin.must_change_password IS '是否要求下次登录后强制修改密码';
COMMENT ON COLUMN sys_admin.last_password_changed_at IS '最近一次密码变更时间';
