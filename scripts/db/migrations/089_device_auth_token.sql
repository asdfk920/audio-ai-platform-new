-- 设备注册功能扩展
-- 为 device 表添加 auth_token 字段用于设备认证

SET search_path TO public;

-- 添加 auth_token 字段（如果不存在）
ALTER TABLE public.device
    ADD COLUMN IF NOT EXISTS auth_token VARCHAR(128) NOT NULL DEFAULT '';

-- 添加索引加速 token 查询
CREATE INDEX IF NOT EXISTS idx_device_auth_token ON public.device(auth_token) WHERE auth_token != '';

-- 注释
COMMENT ON COLUMN public.device.auth_token IS '设备认证 token，用于 MQTT 连接认证';
