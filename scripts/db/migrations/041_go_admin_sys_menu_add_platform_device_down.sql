SET search_path TO public;

UPDATE sys_menu SET deleted_at = NOW() WHERE menu_id = 9005;
