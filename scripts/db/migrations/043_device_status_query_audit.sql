SET search_path TO public;

CREATE TABLE IF NOT EXISTS public.device_status_query_audit (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  sn_norm VARCHAR(64) NOT NULL DEFAULT '',
  client_ip VARCHAR(64) NOT NULL DEFAULT '',
  outcome VARCHAR(32) NOT NULL,
  source VARCHAR(16) NOT NULL DEFAULT '',
  error_code INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_status_query_audit_created ON public.device_status_query_audit (created_at);
CREATE INDEX IF NOT EXISTS idx_device_status_query_audit_user ON public.device_status_query_audit (user_id);

COMMENT ON TABLE public.device_status_query_audit IS 'App 查询设备状态审计（成功/失败、Redis/DB 来源）';
