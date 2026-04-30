-- 添加平台用户管理菜单

-- 插入一级菜单：平台管理
INSERT INTO sys_menu (menu_id, menu_name, title, icon, path, paths, menu_type, action, permission, parent_id, no_cache, breadcrumb, component, sort, visible, is_frame, create_by, update_by, created_at, updated_at, deleted_at)
VALUES (1000, 'platform', '平台管理', 'peoples', '/platform', '/0/1000/', 'M', 'GET', '', 0, false, '', 'Layout', 100, '0', '0', 1, 1, NOW(), NOW(), NULL);

-- 插入二级菜单：平台用户管理
INSERT INTO sys_menu (menu_id, menu_name, title, icon, path, paths, menu_type, action, permission, parent_id, no_cache, breadcrumb, component, sort, visible, is_frame, create_by, update_by, created_at, updated_at, deleted_at)
VALUES (1001, 'platform-user', '平台用户', 'user', '/platform/platform-user', '/0/1000/1001/', 'C', 'GET', '', 1000, false, '', '/admin/platform-user/index', 1, '0', '0', 1, 1, NOW(), NOW(), NULL);

-- 插入按钮权限：查询
INSERT INTO sys_menu (menu_id, menu_name, title, icon, path, paths, menu_type, action, permission, parent_id, no_cache, breadcrumb, component, sort, visible, is_frame, create_by, update_by, created_at, updated_at, deleted_at)
VALUES (1002, '', '平台用户查询', '', '', '/0/1000/1001/1002/', 'F', 'GET', 'admin:platformUser:query', 1001, false, '', '', 1, '0', '0', 1, 1, NOW(), NOW(), NULL);

-- 插入按钮权限：新增
INSERT INTO sys_menu (menu_id, menu_name, title, icon, path, paths, menu_type, action, permission, parent_id, no_cache, breadcrumb, component, sort, visible, is_frame, create_by, update_by, created_at, updated_at, deleted_at)
VALUES (1003, '', '平台用户新增', '', '', '/0/1000/1001/1003/', 'F', 'POST', 'admin:platformUser:add', 1001, false, '', '', 2, '0', '0', 1, 1, NOW(), NOW(), NULL);

-- 为管理员角色分配菜单权限
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT 1, menu_id FROM sys_menu WHERE menu_id IN (1000, 1001, 1002, 1003);
