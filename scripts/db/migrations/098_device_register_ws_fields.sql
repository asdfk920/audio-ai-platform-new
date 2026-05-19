-- WebSocket 设备注册凭证：签名与时间戳（设备 HTTP 注册接口写入，WS 认证读取）
SET search_path TO public;

ALTER TABLE public.device ADD COLUMN IF NOT EXISTS register_signature VARCHAR(64) NULL;
ALTER TABLE public.device ADD COLUMN IF NOT EXISTS register_timestamp BIGINT NULL;

COMMENT ON COLUMN public.device.register_signature IS '注册签名 HMAC-SHA256(hex)，明文 sn + register_timestamp(ms)';
COMMENT ON COLUMN public.device.register_timestamp IS '注册毫秒时间戳 Unix，与 register_signature 计算一致';

-- 应用层使用 status=5（未认证，等待 WS）；扩展约束以免注册后更新状态失败
ALTER TABLE public.device DROP CONSTRAINT IF EXISTS device_status_check;
ALTER TABLE public.device
  ADD CONSTRAINT device_status_check CHECK (status IN (0, 1, 2, 3, 4, 5));

COMMENT ON COLUMN public.device.status IS '0=默认 1=正常 2=禁用 3=未激活/停用 4=未注册 5=已注册未 WS 认证';
