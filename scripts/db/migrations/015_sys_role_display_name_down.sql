-- 回滚 015：恢复 sys_role.role_name = roles.name（与 012 一致）
SET search_path TO public;

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
