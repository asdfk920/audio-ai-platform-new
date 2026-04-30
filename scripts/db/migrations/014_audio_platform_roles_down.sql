-- 回滚 014：恢复 admin/user 为简单 JSON（与 001/009 语义接近），删除 014 新增角色行
SET search_path TO public;

UPDATE roles SET
    description = NULL,
    permissions = '{"all": true}'::jsonb,
    updated_at = CURRENT_TIMESTAMP
WHERE name = 'admin';

UPDATE roles SET
    description = NULL,
    permissions = '{"read": true, "write": false}'::jsonb,
    updated_at = CURRENT_TIMESTAMP
WHERE name = 'user';

DELETE FROM user_role_rel WHERE role_id IN (SELECT id FROM roles WHERE name IN ('super_admin', 'operator', 'guest'));
DELETE FROM roles WHERE name IN ('super_admin', 'operator', 'guest');
