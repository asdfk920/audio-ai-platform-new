-- 回滚：移除「实名审核」菜单项
SET search_path TO public;

DELETE FROM sys_menu WHERE menu_id = 9003;

