SET search_path TO public;

ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS activated_at TIMESTAMP NULL;
ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS activated_ip VARCHAR(45) NOT NULL DEFAULT '';
ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS activated_firmware_version VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS last_auth_at TIMESTAMP NULL;

COMMENT ON COLUMN public.device.activated_at IS '首次激活时间';
COMMENT ON COLUMN public.device.activated_ip IS '首次激活来源 IP';
COMMENT ON COLUMN public.device.activated_firmware_version IS '首次激活时固件版本';
COMMENT ON COLUMN public.device.last_auth_at IS '最后一次设备认证时间';

CREATE TABLE IF NOT EXISTS public.device_auth_failures (
  id BIGSERIAL PRIMARY KEY,
  sn VARCHAR(64) NOT NULL,
  reason VARCHAR(50) NOT NULL,
  client_ip VARCHAR(64) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_auth_failures_sn_time
  ON public.device_auth_failures(sn, created_at DESC);

CREATE TABLE IF NOT EXISTS public.device_auth_audit (
  id BIGSERIAL PRIMARY KEY,
  sn VARCHAR(64) NOT NULL,
  auth_type VARCHAR(32) NOT NULL DEFAULT '',
  client_ip VARCHAR(64) NOT NULL DEFAULT '',
  outcome VARCHAR(32) NOT NULL DEFAULT '',
  error_code INT NOT NULL DEFAULT 0,
  detail JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_auth_audit_sn_time
  ON public.device_auth_audit(sn, created_at DESC);

COMMENT ON TABLE public.device_auth_failures IS '设备认证失败记录表';
COMMENT ON COLUMN public.device_auth_failures.reason IS '失败原因，如 device_not_found/secret_mismatch/signature_invalid';
COMMENT ON TABLE public.device_auth_audit IS '设备认证审计日志表';
