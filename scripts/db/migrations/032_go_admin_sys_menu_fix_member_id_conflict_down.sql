-- 回滚：移除会员管理菜单（9004）
SET search_path TO public;

DELETE FROM sys_menu WHERE menu_id = 9004;

