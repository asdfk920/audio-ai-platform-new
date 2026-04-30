-- Audio AI 平台后台角色矩阵：super_admin / admin / operator / user / guest
-- 权限粒度：modules 下各业务域取值 all | view | self | none（与产品矩阵一致）
SET search_path TO public;
SET client_encoding TO 'UTF8';

-- 已有 admin、user（001）；补充 super_admin、operator、guest，并统一 permissions + description
INSERT INTO roles (name, slug, description, permissions) VALUES
(
    'super_admin',
    'super_admin',
    '超级管理员：全模块；仅系统内置唯一账号，不对外分配',
    '{"modules":{"user_mgmt":"all","device_mgmt":"all","content_mgmt":"all","ota":"all","stats":"all","audit_log":"all","sys_config":"all"},"scope":"super_admin"}'::jsonb
),
(
    'operator',
    'operator',
    '运营人员：内容全权限与数据统计；OTA/日志仅查看；不涉及用户安全与系统配置',
    '{"modules":{"user_mgmt":"none","device_mgmt":"none","content_mgmt":"all","ota":"view","stats":"all","audit_log":"view","sys_config":"none"},"scope":"operator"}'::jsonb
),
(
    'guest',
    'guest',
    '游客：无后台能力；仅公开端浏览（业务侧自行限制）',
    '{"modules":{"user_mgmt":"none","device_mgmt":"none","content_mgmt":"none","ota":"none","stats":"none","audit_log":"none","sys_config":"none"},"scope":"guest"}'::jsonb
)
ON CONFLICT (name) DO UPDATE SET
    slug = EXCLUDED.slug,
    description = EXCLUDED.description,
    permissions = EXCLUDED.permissions,
    updated_at = CURRENT_TIMESTAMP;

-- 系统管理员：除「系统配置」外全模块（矩阵：无系统配置）
UPDATE roles SET
    slug = 'admin',
    description = '系统管理员：用户/设备/内容/OTA/统计/日志全权限；不含系统配置',
    permissions = '{"modules":{"user_mgmt":"all","device_mgmt":"all","content_mgmt":"all","ota":"all","stats":"all","audit_log":"all","sys_config":"none"},"scope":"admin"}'::jsonb,
    updated_at = CURRENT_TIMESTAMP
WHERE name = 'admin';

-- 普通用户：C 端；仅本人设备与内容（矩阵：设备/内容为「自己」）
UPDATE roles SET
    slug = 'user',
    description = '普通用户：仅本人账号、设备与内容；无后台管理权限',
    permissions = '{"modules":{"user_mgmt":"none","device_mgmt":"self","content_mgmt":"self","ota":"none","stats":"none","audit_log":"none","sys_config":"none"},"scope":"end_user"}'::jsonb,
    updated_at = CURRENT_TIMESTAMP
WHERE name = 'user';

COMMENT ON TABLE roles IS '平台角色；permissions.modules 键：user_mgmt/device_mgmt/content_mgmt/ota/stats/audit_log/sys_config；值：all|view|self|none';
