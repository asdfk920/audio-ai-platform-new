-- 修复设备在线状态约束：添加"重启中"状态
-- 原因：设备重启指令下发后需要标记为"重启中"(2)，原约束只允许(0,1)导致报错
-- 错误信息：new value for column "online_status" violates check constraint "device_online_status_check"
--
-- 状态定义：
--   0 = 离线（offline）
--   1 = 在线（online）
--   2 = 重启中（rebooting）

SET search_path TO public;

-- 删除旧约束
ALTER TABLE public.device DROP CONSTRAINT IF EXISTS device_online_status_check;

-- 创建新约束（包含重启中状态）
ALTER TABLE public.device
  ADD CONSTRAINT device_online_status_check CHECK (online_status IN (0, 1, 2));

COMMENT ON COLUMN public.device.online_status IS '设备在线状态：0=离线 1=在线 2=重启中';
