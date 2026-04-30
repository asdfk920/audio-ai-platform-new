-- go-admin 左侧菜单（/api/v1/menurole）所需最小表：sys_menu
-- 仅用于 admin 角色读取菜单；其他角色若需菜单权限，请补齐 sys_role/sys_role_menu 体系（当前项目用 sys_* 视图对齐 users/roles）。
SET search_path TO public;
SET client_encoding TO 'UTF8';

CREATE TABLE IF NOT EXISTS sys_menu (
  menu_id BIGSERIAL PRIMARY KEY,
  menu_name VARCHAR(128),
  title VARCHAR(128),
  icon VARCHAR(128),
  path VARCHAR(128),
  paths VARCHAR(128),
  menu_type VARCHAR(1),
  action VARCHAR(16),
  permission VARCHAR(255),
  parent_id BIGINT,
  no_cache BOOLEAN,
  breadcrumb VARCHAR(255),
  component VARCHAR(255),
  sort INT,
  visible VARCHAR(1),
  is_frame VARCHAR(1),
  create_by BIGINT DEFAULT 0,
  update_by BIGINT DEFAULT 0,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_sys_menu_parent_id ON sys_menu(parent_id);
CREATE INDEX IF NOT EXISTS idx_sys_menu_deleted_at ON sys_menu(deleted_at);

-- 初始化最小菜单：平台用户 + 权限管理（矩阵）
INSERT INTO sys_menu (menu_id, menu_name, title, icon, path, paths, menu_type, action, permission, parent_id, no_cache, breadcrumb, component, sort, visible, is_frame, create_by, update_by, created_at, updated_at, deleted_at)
VALUES
  (9000, 'PlatformRoot', '平台管理', 'el-icon-s-grid', '/platform', '/0/9000', 'M', '无', '', 0, true, '', 'Layout', 10, '0', '1', 0, 0, NOW(), NOW(), NULL),
  (9001, 'PlatformUser', '平台用户', 'user', '/platform-user', '/0/9000/9001', 'C', '无', '', 9000, true, '', '/admin/platform-user/index', 10, '0', '1', 0, 0, NOW(), NOW(), NULL),
  (9002, 'PlatformRbac', '权限管理', 'lock', '/platform-rbac', '/0/9000/9002', 'C', '无', '', 9000, true, '', '/admin/platform-rbac/index', 20, '0', '1', 0, 0, NOW(), NOW(), NULL)
ON CONFLICT (menu_id) DO UPDATE SET
  menu_name = EXCLUDED.menu_name,
  title = EXCLUDED.title,
  icon = EXCLUDED.icon,
  path = EXCLUDED.path,
  paths = EXCLUDED.paths,
  menu_type = EXCLUDED.menu_type,
  parent_id = EXCLUDED.parent_id,
  no_cache = EXCLUDED.no_cache,
  breadcrumb = EXCLUDED.breadcrumb,
  component = EXCLUDED.component,
  sort = EXCLUDED.sort,
  visible = EXCLUDED.visible,
  is_frame = EXCLUDED.is_frame,
  updated_at = CURRENT_TIMESTAMP,
  deleted_at = NULL;

