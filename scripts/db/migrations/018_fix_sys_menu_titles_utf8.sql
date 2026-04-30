-- 修复 sys_menu 中文标题乱码（使用 Unicode 转义）
SET search_path TO public;
SET client_encoding TO 'UTF8';

UPDATE sys_menu SET title = U&'\5e73\53f0\7ba1\7406', updated_at = CURRENT_TIMESTAMP
WHERE menu_id = 9000;

UPDATE sys_menu SET title = U&'\5e73\53f0\7528\6237', updated_at = CURRENT_TIMESTAMP
WHERE menu_id = 9001;

UPDATE sys_menu SET title = U&'\6743\9650\7ba1\7406', updated_at = CURRENT_TIMESTAMP
WHERE menu_id = 9002;

-- action 字段显示用，统一为「无」
UPDATE sys_menu SET action = U&'\65e0', updated_at = CURRENT_TIMESTAMP
WHERE menu_id in (9000,9001,9002);

