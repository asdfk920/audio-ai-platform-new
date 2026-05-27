-- 添加设备名称字段
-- device_name_raw: 设备原始名称（设备出厂或用户首次设置的名称）
-- device_name: 用户备注名称（用户绑定设备时设置）

-- 1. 为 device 表添加设备原始名称字段
ALTER TABLE public.device
    ADD COLUMN IF NOT EXISTS device_name_raw VARCHAR(100) DEFAULT '' COMMENT '设备原始名称（出厂或首次设置）';

-- 2. 为 device 表添加用户备注名称字段
ALTER TABLE public.device
    ADD COLUMN IF NOT EXISTS device_name VARCHAR(100) DEFAULT '' COMMENT '用户备注名称（用户绑定设备时设置）';

-- 3. 为 user_device_bind 表添加用户备注名称字段
ALTER TABLE public.user_device_bind
    ADD COLUMN IF NOT EXISTS device_name VARCHAR(100) DEFAULT '未命名设备' COMMENT '用户备注名称（用户为设备起的别名）';

-- 4. 创建索引以加速按名称查询设备
CREATE INDEX IF NOT EXISTS idx_device_name_raw ON public.device(device_name_raw);
CREATE INDEX IF NOT EXISTS idx_device_name ON public.device(device_name);
CREATE INDEX IF NOT EXISTS idx_user_device_bind_device_name ON public.user_device_bind(device_name);

-- 5. 添加字段注释
COMMENT ON COLUMN public.device.device_name_raw IS '设备原始名称：设备出厂时预设或用户首次注册时设置的名称，如"客厅音响"、"卧室音箱"等';
COMMENT ON COLUMN public.device.device_name IS '用户备注名称：用户绑定设备后为设备起的别名，如"我的音响"、"小明房间的音箱"，可为空表示未设置';
COMMENT ON COLUMN public.user_device_bind.device_name IS '用户备注名称：用户在绑定时为设备起的别名，便于识别和管理多台设备';

-- 6. 更新时间戳
SELECT NOW() AS migration_completed_at;
