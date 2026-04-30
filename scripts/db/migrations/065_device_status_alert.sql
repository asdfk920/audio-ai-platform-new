-- 设备状态上报触发的轻量告警记录（与 reportsvc 结构化日志配套，便于运营检索）
SET search_path TO public;

CREATE TABLE IF NOT EXISTS public.device_status_alert (
  id BIGSERIAL PRIMARY KEY,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  alert_type VARCHAR(64) NOT NULL,
  severity VARCHAR(32) NOT NULL DEFAULT 'info',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_status_alert_device_created
  ON public.device_status_alert (device_id, created_at DESC);

COMMENT ON TABLE public.device_status_alert IS '设备状态上报阈值告警（电量/存储等），异步写入';
