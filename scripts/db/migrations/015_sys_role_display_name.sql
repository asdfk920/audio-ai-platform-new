-- 平台用户/角色管理下拉：sys_role.role_name 展示中文分类短名（取 description 中全角「：」前一段；与 014 五类角色一致）
-- 物理列 roles.name / slug 不变，仅影响 go-admin 列表与 el-select 的 label。
SET search_path TO public;

DROP VIEW IF EXISTS sys_role CASCADE;

CREATE VIEW sys_role AS
SELECT
    r.id AS role_id,
    COALESCE(
        NULLIF(TRIM(split_part(COALESCE(r.description, ''), E'：', 1)), ''),
        r.name
    )::VARCHAR(128) AS role_name,
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

COMMENT ON VIEW sys_role IS 'go-admin 兼容视图；role_name 为中文分类短名（见 015）';
