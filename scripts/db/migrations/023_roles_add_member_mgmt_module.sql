-- 023: roles.permissions.modules 增加 member_mgmt（会员管理）默认值
SET search_path TO public;

-- super_admin / admin 默认全量；operator 默认 view；其它 none
UPDATE roles
SET permissions = jsonb_set(
    COALESCE(permissions, '{}'::jsonb),
    '{modules,member_mgmt}',
    to_jsonb(
        CASE
            WHEN slug IN ('super_admin','admin') THEN 'all'
            WHEN slug IN ('operator') THEN 'view'
            ELSE 'none'
        END
    ),
    true
)
WHERE slug IN ('super_admin','admin','operator','user','guest');

