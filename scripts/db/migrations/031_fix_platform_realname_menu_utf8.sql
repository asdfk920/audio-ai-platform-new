-- 修复 sys_menu 中「实名审核」菜单标题乱码
SET search_path TO public;
SET client_encoding TO 'UTF8';

UPDATE sys_menu
SET title = U&'\5b9e\540d\5ba1\6838',
    menu_name = 'PlatformRealName',
    icon = 'eye',
    path = '/platform-realname',
    component = '/admin/platform-realname/index',
    visible = '0',
    parent_id = 9000,
    updated_at = CURRENT_TIMESTAMP,
    deleted_at = NULL
WHERE menu_id = 9003;

