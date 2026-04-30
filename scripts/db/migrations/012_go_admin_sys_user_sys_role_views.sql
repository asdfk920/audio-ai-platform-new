-- 对齐 go-admin GORM 模型（GetInfo /sys_user 等）：user_id、nick_name、phone、password、role_id 等
-- 原 009 视图用 id/password_hash 等列名，导致登录后 /api/v1/getinfo 查不到用户。
SET search_path TO public;

DROP VIEW IF EXISTS sys_user_role CASCADE;
DROP VIEW IF EXISTS sys_user CASCADE;
DROP VIEW IF EXISTS sys_role CASCADE;

CREATE VIEW sys_role AS
SELECT
    r.id AS role_id,
    r.name AS role_name,
    '2'::VARCHAR(4) AS status,
    COALESCE(r.slug, '')::VARCHAR(128) AS role_key,
    0 AS role_sort,
    ''::VARCHAR(128) AS flag,
    COALESCE(r.permissions::TEXT, '')::VARCHAR(1024) AS remark,
    false AS admin,
    '1'::VARCHAR(128) AS data_scope,
    0 AS create_by,
    0 AS update_by,
    r.created_at,
    r.updated_at,
    NULL::TIMESTAMP WITH TIME ZONE AS deleted_at
FROM roles r;

COMMENT ON VIEW sys_role IS 'go-admin 兼容视图；物理表 public.roles';

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
    u.deleted_at::TIMESTAMP WITH TIME ZONE AS deleted_at
FROM users u;

COMMENT ON VIEW sys_user IS 'go-admin 兼容视图；物理表 public.users + user_role_rel';

CREATE VIEW sys_user_role AS
SELECT
    ur.id,
    ur.user_id,
    ur.role_id
FROM user_role_rel ur;

COMMENT ON VIEW sys_user_role IS '用户-角色关联；物理表 public.user_role_rel';
