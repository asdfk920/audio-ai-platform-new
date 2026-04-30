-- 回滚 081：恢复 sys_user / sys_user_role 基于 public.users（与 012 一致）
SET search_path TO public;

DROP VIEW IF EXISTS sys_user_role CASCADE;
DROP VIEW IF EXISTS sys_user CASCADE;

CREATE VIEW sys_user AS
SELECT
    u.id AS user_id,
    u.username,
    u.password AS password,
    u.salt AS salt,
    COALESCE(u.nickname, '')::VARCHAR(128) AS nick_name,
    COALESCE(u.mobile, '')::VARCHAR(11) AS phone,
    COALESCE(u.email, '')::VARCHAR(128) AS email,
    COALESCE(u.avatar, '')::VARCHAR(255) AS avatar,
    ''::VARCHAR(255) AS sex,
    COALESCE((
        SELECT ur.role_id FROM user_role_rel ur
        WHERE ur.user_id = u.id ORDER BY ur.id LIMIT 1
    ), 0)::INT AS role_id,
    0 AS dept_id,
    0 AS post_id,
    ''::VARCHAR(255) AS remark,
    CASE WHEN u.status = 1 THEN '2' ELSE '1' END::VARCHAR(4) AS status,
    0 AS create_by,
    0 AS update_by,
    u.created_at,
    u.updated_at,
    u.deleted_at::TIMESTAMPTZ AS deleted_at,
    ''::VARCHAR(50) AS login_ip,
    0::BIGINT AS login_count,
    COALESCE(u.updated_at, u.created_at)::TIMESTAMPTZ AS last_login_time
FROM users u;

COMMENT ON VIEW sys_user IS 'go-admin 兼容视图；物理表 public.users + user_role_rel';

CREATE VIEW sys_user_role AS
SELECT
    ur.id,
    ur.user_id,
    ur.role_id
FROM user_role_rel ur;

COMMENT ON VIEW sys_user_role IS '用户-角色关联；物理表 public.user_role_rel';
