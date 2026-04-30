-- go-admin sys_config: login page GET /api/v1/app-config (ASCII-only for Windows/psql piping)
SET search_path TO public;

CREATE TABLE IF NOT EXISTS sys_config (
    id BIGSERIAL PRIMARY KEY,
    config_name VARCHAR(128),
    config_key VARCHAR(128),
    config_value VARCHAR(255),
    config_type VARCHAR(64),
    is_frontend VARCHAR(64),
    remark VARCHAR(128),
    create_by INT DEFAULT 0,
    update_by INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

INSERT INTO sys_config (id, config_name, config_key, config_value, config_type, is_frontend, remark, create_by, update_by, created_at, updated_at, deleted_at)
VALUES
(1, 'skin', 'sys_index_skinName', 'skin-green', 'Y', '0', '', 1, 1, NOW(), NOW(), NULL),
(2, 'initPassword', 'sys_user_initPassword', '123456', 'Y', '0', '', 1, 1, NOW(), NOW(), NULL),
(3, 'sideTheme', 'sys_index_sideTheme', 'theme-dark', 'Y', '0', '', 1, 1, NOW(), NOW(), NULL),
(4, 'appName', 'sys_app_name', 'go-admin', 'Y', '1', '', 1, 0, NOW(), NOW(), NULL),
(5, 'appLogo', 'sys_app_logo', 'https://doc-image.zhangwj.com/img/go-admin.png', 'Y', '1', '', 1, 0, NOW(), NOW(), NULL)
ON CONFLICT (id) DO NOTHING;

SELECT setval(pg_get_serial_sequence('sys_config', 'id'), GREATEST((SELECT COALESCE(MAX(id), 1) FROM sys_config), 5));
