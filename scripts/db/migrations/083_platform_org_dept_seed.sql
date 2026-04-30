-- 平台控制台组织架构（sys_dept），供管理员 dept_id 分配与 /api/v1/deptTree 下拉。
-- 依赖 public.sys_dept 已存在（go-admin AutoMigrate 或 ensure-admin-tables）。
-- 使用固定 dept_id 100–108，避免与历史自增部门（如 id=1 Default）冲突；执行后请同步序列。
SET search_path TO public;

-- dept_name 使用 UTF-8 十六进制经 convert_from 写入，避免 Windows/psql 管道非 UTF-8 导致中文变 ? 
INSERT INTO sys_dept AS d (dept_id, parent_id, dept_path, dept_name, sort, leader, phone, email, status, create_by, update_by, created_at, updated_at)
VALUES
  (100, 0, '/0/100/', convert_from(decode('e680bbe7bb8fe79086e58a9ee585ace5aea4', 'hex'), 'UTF8'), 10, '', '', '', 2, 1, 1, NOW(), NOW()),
  (101, 0, '/0/101/', convert_from(decode('e68a80e69cafe983a8', 'hex'), 'UTF8'), 20, '', '', '', 2, 1, 1, NOW(), NOW()),
  (102, 101, '/0/101/102/', convert_from(decode('e5898de7abafe7bb84', 'hex'), 'UTF8'), 1, '', '', '', 2, 1, 1, NOW(), NOW()),
  (103, 101, '/0/101/103/', convert_from(decode('e5908ee7abafe7bb84', 'hex'), 'UTF8'), 2, '', '', '', 2, 1, 1, NOW(), NOW()),
  (104, 101, '/0/101/104/', convert_from(decode('e8bf90e7bbb4e7bb84', 'hex'), 'UTF8'), 3, '', '', '', 2, 1, 1, NOW(), NOW()),
  (105, 0, '/0/105/', convert_from(decode('e8bf90e890a5e983a8', 'hex'), 'UTF8'), 30, '', '', '', 2, 1, 1, NOW(), NOW()),
  (106, 105, '/0/105/106/', convert_from(decode('e5b882e59cbae7bb84', 'hex'), 'UTF8'), 1, '', '', '', 2, 1, 1, NOW(), NOW()),
  (107, 105, '/0/105/107/', convert_from(decode('e5aea2e69c8de7bb84', 'hex'), 'UTF8'), 2, '', '', '', 2, 1, 1, NOW(), NOW()),
  (108, 0, '/0/108/', convert_from(decode('e8b4a2e58aa1e983a8', 'hex'), 'UTF8'), 40, '', '', '', 2, 1, 1, NOW(), NOW())
ON CONFLICT (dept_id) DO UPDATE SET
  parent_id  = EXCLUDED.parent_id,
  dept_path  = EXCLUDED.dept_path,
  dept_name  = EXCLUDED.dept_name,
  sort       = EXCLUDED.sort,
  leader     = EXCLUDED.leader,
  phone      = EXCLUDED.phone,
  email      = EXCLUDED.email,
  status     = EXCLUDED.status,
  update_by  = EXCLUDED.update_by,
  updated_at = NOW();

-- 显式主键写入后同步序列，避免后续自增与 100–108 冲突
SELECT setval(
  pg_get_serial_sequence('sys_dept', 'dept_id'),
  GREATEST((SELECT COALESCE(MAX(dept_id), 1) FROM sys_dept), 1)
);
