-- 财务管理员角色：订单/会员/统计为主，不含设备管理（与需求「财务管理员」一致）
SET search_path TO public;
SET client_encoding TO 'UTF8';

INSERT INTO roles (name, slug, description, permissions) VALUES
(
    'finance',
    'finance',
    '财务管理员：订单与会员/权益、统计与报表；不可管理设备与系统配置',
    '{"modules":{"user_mgmt":"none","member_mgmt":"all","device_mgmt":"none","content_mgmt":"none","ota":"none","stats":"all","audit_log":"view","sys_config":"none"},"scope":"finance"}'::jsonb
)
ON CONFLICT (name) DO UPDATE SET
    slug = EXCLUDED.slug,
    description = EXCLUDED.description,
    permissions = EXCLUDED.permissions,
    updated_at = CURRENT_TIMESTAMP;

COMMENT ON TABLE roles IS '平台角色；permissions.modules：user_mgmt/member_mgmt/device_mgmt/content_mgmt/ota/stats/audit_log/sys_config；值：all|view|self|none';
