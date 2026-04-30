-- Minimal seed after ensure-admin-tables (PostgreSQL). Safe to run once.
INSERT INTO sys_dict_type (dict_name, dict_type, status, remark, create_by, update_by, created_at, updated_at)
SELECT 'User sex', 'sys_user_sex', 2, '', 1, 1, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM sys_dict_type WHERE dict_type = 'sys_user_sex');
INSERT INTO sys_dict_type (dict_name, dict_type, status, remark, create_by, update_by, created_at, updated_at)
SELECT 'Normal disable', 'sys_normal_disable', 2, '', 1, 1, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM sys_dict_type WHERE dict_type = 'sys_normal_disable');

INSERT INTO sys_dict_data (dict_sort, dict_label, dict_value, dict_type, status, create_by, update_by, created_at, updated_at)
SELECT 0, 'OK', '2', 'sys_normal_disable', 2, 1, 1, NOW(), NOW() WHERE NOT EXISTS (SELECT 1 FROM sys_dict_data WHERE dict_type = 'sys_normal_disable' AND dict_value = '2');
INSERT INTO sys_dict_data (dict_sort, dict_label, dict_value, dict_type, status, create_by, update_by, created_at, updated_at)
SELECT 0, 'Off', '1', 'sys_normal_disable', 2, 1, 1, NOW(), NOW() WHERE NOT EXISTS (SELECT 1 FROM sys_dict_data WHERE dict_type = 'sys_normal_disable' AND dict_value = '1');
INSERT INTO sys_dict_data (dict_sort, dict_label, dict_value, dict_type, status, create_by, update_by, created_at, updated_at)
SELECT 0, 'M', '0', 'sys_user_sex', 2, 1, 1, NOW(), NOW() WHERE NOT EXISTS (SELECT 1 FROM sys_dict_data WHERE dict_type = 'sys_user_sex' AND dict_value = '0');
INSERT INTO sys_dict_data (dict_sort, dict_label, dict_value, dict_type, status, create_by, update_by, created_at, updated_at)
SELECT 0, 'F', '1', 'sys_user_sex', 2, 1, 1, NOW(), NOW() WHERE NOT EXISTS (SELECT 1 FROM sys_dict_data WHERE dict_type = 'sys_user_sex' AND dict_value = '1');
INSERT INTO sys_dict_data (dict_sort, dict_label, dict_value, dict_type, status, create_by, update_by, created_at, updated_at)
SELECT 0, 'U', '2', 'sys_user_sex', 2, 1, 1, NOW(), NOW() WHERE NOT EXISTS (SELECT 1 FROM sys_dict_data WHERE dict_type = 'sys_user_sex' AND dict_value = '2');

INSERT INTO sys_dept (parent_id, dept_path, dept_name, sort, leader, phone, email, status, create_by, update_by, created_at, updated_at)
SELECT 0, '/0/1/', 'Default', 0, '', '', '', 2, 1, 1, NOW(), NOW() WHERE NOT EXISTS (SELECT 1 FROM sys_dept WHERE dept_id = 1);

-- 岗位（新增用户弹窗 GET /api/v1/post 依赖 sys_post）
INSERT INTO sys_post (post_name, post_code, sort, status, remark, create_by, update_by, created_at, updated_at)
SELECT 'Default', 'default', 0, 2, '', 1, 1, NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM sys_post WHERE post_code = 'default');
