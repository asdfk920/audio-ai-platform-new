-- 将已具备控制台角色（users + user_role_rel + roles）的账号补录到 sys_admin（历史注册在 069 落表前不会产生 sys_admin 行）
SET search_path TO public;
SET client_encoding TO 'UTF8';

INSERT INTO sys_admin (
    username,
    password,
    nick_name,
    role_id,
    role_name,
    role_code,
    status,
    login_count,
    password_changed_at,
    created_at,
    updated_at
)
SELECT
    s.username,
    s.password,
    s.nick_name,
    s.role_id,
    s.role_name,
    s.role_code,
    1,
    0,
    s.password_changed_at,
    s.created_at,
    CURRENT_TIMESTAMP
FROM (
    SELECT DISTINCT ON (u.id)
        u.username,
        u.password,
        LEFT(BTRIM(COALESCE(NULLIF(TRIM(u.nickname), ''), u.username)), 64) AS nick_name,
        r.id AS role_id,
        LEFT(BTRIM(COALESCE(NULLIF(TRIM(r.name), ''), r.slug)), 50) AS role_name,
        LEFT(BTRIM(COALESCE(r.slug, '')), 50) AS role_code,
        u.password_changed_at,
        COALESCE(u.created_at::timestamptz, CURRENT_TIMESTAMP) AS created_at
    FROM users u
    INNER JOIN user_role_rel ur ON ur.user_id = u.id
    INNER JOIN roles r ON r.id = ur.role_id
    WHERE u.deleted_at IS NULL
      AND u.username IS NOT NULL
      AND BTRIM(u.username) <> ''
      AND LOWER(BTRIM(COALESCE(r.slug, ''))) IN ('super_admin', 'admin', 'operator', 'finance')
    ORDER BY
        u.id,
        CASE LOWER(BTRIM(COALESCE(r.slug, '')))
            WHEN 'super_admin' THEN 1
            WHEN 'admin' THEN 2
            WHEN 'operator' THEN 3
            WHEN 'finance' THEN 4
            ELSE 5
        END,
        r.id
) AS s
WHERE NOT EXISTS (
    SELECT 1
    FROM sys_admin sa
    WHERE sa.deleted_at IS NULL
      AND LOWER(BTRIM(sa.username)) = LOWER(BTRIM(s.username))
);
