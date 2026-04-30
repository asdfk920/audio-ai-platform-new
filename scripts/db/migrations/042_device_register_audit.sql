-- 设备注册审计（成功 / 失败 / 限流等），便于安全与合规追溯
SET search_path TO public;

CREATE TABLE IF NOT EXISTS public.device_register_audit (
  id BIGSERIAL PRIMARY KEY,
  sn_norm VARCHAR(64) NOT NULL DEFAULT '',
  product_key_suffix VARCHAR(24) NOT NULL DEFAULT '',
  client_ip VARCHAR(64) NOT NULL DEFAULT '',
  outcome VARCHAR(32) NOT NULL,
  error_code INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_register_audit_created_at ON public.device_register_audit (created_at);
CREATE INDEX IF NOT EXISTS idx_device_register_audit_sn_norm ON public.device_register_audit (sn_norm);

COMMENT ON TABLE public.device_register_audit IS '设备首次注册审计：SN（规范化）、IP、结果、错误码';
