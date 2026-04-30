-- 024: go-admin 左侧菜单补齐「会员管理」
SET search_path TO public;
SET client_encoding TO 'UTF8';

-- 复用平台根菜单 9000，新增子菜单 9004（避免与实名审核 9003 冲突）
INSERT INTO sys_menu (menu_id, menu_name, title, icon, path, paths, menu_type, action, permission, parent_id, no_cache, breadcrumb, component, sort, visible, is_frame, create_by, update_by, created_at, updated_at, deleted_at)
VALUES
  (9004, 'PlatformMember', '会员管理', 'peoples', '/platform-member', '/0/9000/9004', 'C', '无', '', 9000, true, '', '/admin/platform-member/index', 30, '0', '1', 0, 0, NOW(), NOW(), NULL)
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

