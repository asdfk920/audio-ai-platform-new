-- 010_add_device_info_fields.sql
-- 为用户设备绑定表添加设备信息字段（位置、分组、场景）
-- 用于支持设备信息更新接口（PUT /api/v1/user/device/upd）

-- 为 user_device_bind 表添加位置、分组、场景字段
ALTER TABLE public.user_device_bind
    ADD COLUMN IF NOT EXISTS location VARCHAR(100) DEFAULT '';

ALTER TABLE public.user_device_bind
    ADD COLUMN IF NOT EXISTS group_name VARCHAR(100) DEFAULT '';

ALTER TABLE public.user_device_bind
    ADD COLUMN IF NOT EXISTS scene VARCHAR(500) DEFAULT '';

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_user_device_bind_location ON public.user_device_bind(location);
CREATE INDEX IF NOT EXISTS idx_user_device_bind_group_name ON public.user_device_bind(group_name);
CREATE INDEX IF NOT EXISTS idx_user_device_bind_scene ON public.user_device_bind(scene);

-- 添加字段注释（PG 必须单独写 COMMENT ON COLUMN）
COMMENT ON COLUMN public.user_device_bind.location IS '设备位置信息，用户可自定义设置';
COMMENT ON COLUMN public.user_device_bind.group_name IS '设备分组名称，用于设备分类管理';
COMMENT ON COLUMN public.user_device_bind.scene IS '设备使用场景，支持多场景，用逗号分隔';

-- 验证字段是否添加成功
SELECT column_name, data_type, column_default, is_nullable
FROM information_schema.columns
WHERE table_name = 'user_device_bind'
  AND column_name IN ('location', 'group_name', 'scene')
ORDER BY ordinal_position;