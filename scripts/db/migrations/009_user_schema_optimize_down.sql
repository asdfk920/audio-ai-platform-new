-- 回滚 009_user_schema_optimize（按需执行；有依赖时注意顺序）

SET search_path TO public;

DROP VIEW IF EXISTS sys_user CASCADE;
DROP VIEW IF EXISTS sys_role CASCADE;
DROP VIEW IF EXISTS sys_user_role CASCADE;

CREATE OR REPLACE VIEW sys_user AS
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

CREATE OR REPLACE VIEW sys_role AS
SELECT
    r.id,
    r.name AS role_name,
    COALESCE(r.permissions::TEXT, '')::VARCHAR(512) AS remark,
    1::INT AS status
FROM roles r;

CREATE OR REPLACE VIEW sys_user_role AS
SELECT
    ur.id,
    ur.user_id,
    ur.role_id
FROM user_role_rel ur;

DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permission_defs;
DROP TABLE IF EXISTS user_settings;

DROP INDEX IF EXISTS idx_user_auth_expired_at;
DROP INDEX IF EXISTS idx_users_status_created;
DROP INDEX IF EXISTS idx_users_username_lower;
DROP INDEX IF EXISTS idx_roles_slug_lower;

ALTER TABLE users DROP COLUMN IF EXISTS username;
ALTER TABLE roles DROP COLUMN IF EXISTS description;
ALTER TABLE roles DROP COLUMN IF EXISTS slug;
ALTER TABLE roles DROP COLUMN IF EXISTS updated_at;

COMMENT ON VIEW sys_user IS '逻辑用户主表视图；物理表为 public.users（含实名/注销等扩展列未全部展开，直接查 users 可得全量）';
COMMENT ON VIEW sys_role IS '逻辑角色视图；物理表为 public.roles';
COMMENT ON VIEW sys_user_role IS '用户-角色关联视图；物理表为 public.user_role_rel';
