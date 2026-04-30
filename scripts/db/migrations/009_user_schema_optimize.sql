-- 用户域表结构优化（PostgreSQL；对齐「用户名 / 权限规范化 / 角色说明 / 用户配置」等常见设计）
-- 说明：不引入 MySQL 社区业务表（posts/tags 等），仅增强本项目的用户与 RBAC 相关表。

SET search_path TO public;

-- ---------------------------------------------------------------------------
-- 1) users：可选登录用户名（与 email/mobile 并存，均可空）
-- ---------------------------------------------------------------------------
ALTER TABLE users ADD COLUMN IF NOT EXISTS username VARCHAR(64) NULL;
COMMENT ON COLUMN users.username IS '登录用户名，唯一（忽略大小写）；为空则仅用邮箱/手机登录';

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_lower
    ON users (lower(username))
    WHERE username IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_status_created
    ON users (status, created_at DESC)
    WHERE deleted_at IS NULL;

-- ---------------------------------------------------------------------------
-- 2) roles：补充说明、slug、更新时间（与 JSONB permissions 并存）
-- ---------------------------------------------------------------------------
ALTER TABLE roles ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS slug VARCHAR(64) NULL;
ALTER TABLE roles ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

COMMENT ON COLUMN roles.description IS '角色说明';
COMMENT ON COLUMN roles.slug IS '角色标识，如 admin、user，用于代码与接口';

CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_slug_lower
    ON roles (lower(slug))
    WHERE slug IS NOT NULL;

UPDATE roles SET slug = lower(name) WHERE slug IS NULL AND name IS NOT NULL;

-- ---------------------------------------------------------------------------
-- 3) 权限点规范化：permission_defs + role_permissions（与 roles.permissions JSONB 并存）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS permission_defs (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(128) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE permission_defs IS '权限点定义；细粒度授权用本表，粗粒度仍可用 roles.permissions JSONB';

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES permission_defs(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_perm ON role_permissions(permission_id);

COMMENT ON TABLE role_permissions IS '角色与权限点多对多';

INSERT INTO permission_defs (name, slug, description) VALUES
    ('全部', 'all', '超级管理员（兼容旧 admin 角色）'),
    ('用户读', 'user:read', '查询用户数据'),
    ('用户写', 'user:write', '修改用户数据'),
    ('设备读', 'device:read', '查询设备'),
    ('内容读', 'content:read', '查询内容')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permission_defs p
WHERE r.name = 'admin' AND p.slug = 'all'
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permission_defs p
WHERE r.name = 'user' AND p.slug IN ('user:read', 'device:read', 'content:read')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- 4) 用户级键值配置（偏好、功能开关等，JSON 值）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_settings (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    setting_key VARCHAR(64) NOT NULL,
    setting_value JSONB NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, setting_key)
);

CREATE INDEX IF NOT EXISTS idx_user_settings_key ON user_settings(setting_key);

COMMENT ON TABLE user_settings IS '用户级配置项；setting_key 如 theme、notify_push';

-- ---------------------------------------------------------------------------
-- 5) 辅助索引
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_user_auth_expired_at ON user_auth(expired_at) WHERE expired_at IS NOT NULL;

-- ---------------------------------------------------------------------------
-- 6) 视图 sys_user / sys_role：带上新列（列顺序变化需 DROP 再建，勿用 OR REPLACE）
-- ---------------------------------------------------------------------------
DROP VIEW IF EXISTS sys_user CASCADE;
CREATE VIEW sys_user AS
SELECT
    u.id,
    u.username,
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

COMMENT ON VIEW sys_user IS '逻辑用户主表视图；物理表为 public.users';

DROP VIEW IF EXISTS sys_role CASCADE;
CREATE VIEW sys_role AS
SELECT
    r.id,
    r.name AS role_name,
    r.slug AS role_slug,
    r.description AS role_description,
    COALESCE(r.permissions::TEXT, '')::VARCHAR(1024) AS remark,
    1::INT AS status
FROM roles r;

COMMENT ON VIEW sys_role IS '逻辑角色视图；物理表为 public.roles';
