-- 设备影子管理端 API（Casbin / sys_api 同步用，按需执行）
INSERT INTO sys_api (handle, title, path, type, action, created_at, updated_at)
SELECT 'go-admin/app/admin/device/apis.PlatformDevice.ShadowList-fm', '设备影子列表', '/api/v1/platform-device/shadow/list', 'BUS', 'GET', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM sys_api WHERE path = '/api/v1/platform-device/shadow/list' AND action = 'GET');

INSERT INTO sys_api (handle, title, path, type, action, created_at, updated_at)
SELECT 'go-admin/app/admin/device/apis.PlatformDevice.ShadowGet-fm', '设备影子查询', '/api/v1/platform-device/shadow', 'BUS', 'GET', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM sys_api WHERE path = '/api/v1/platform-device/shadow' AND action = 'GET');

INSERT INTO sys_api (handle, title, path, type, action, created_at, updated_at)
SELECT 'go-admin/app/admin/device/apis.PlatformDevice.ShadowPutDesired-fm', '设备影子更新desired', '/api/v1/platform-device/shadow/desired', 'BUS', 'PUT', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM sys_api WHERE path = '/api/v1/platform-device/shadow/desired' AND action = 'PUT');
