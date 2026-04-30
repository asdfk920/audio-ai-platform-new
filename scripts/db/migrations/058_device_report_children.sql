SET search_path TO public;

CREATE TABLE IF NOT EXISTS public.device_child (
  id BIGSERIAL PRIMARY KEY,
  host_device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  child_key VARCHAR(64) NOT NULL,
  child_sn VARCHAR(64) NOT NULL DEFAULT '',
  child_type VARCHAR(64) NOT NULL DEFAULT '',
  child_name VARCHAR(128) NOT NULL DEFAULT '',
  online_status SMALLINT NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  metadata JSONB,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_device_child_host_key UNIQUE(host_device_id, child_key),
  CONSTRAINT device_child_online_status_check CHECK (online_status IN (0, 1)),
  CONSTRAINT device_child_status_check CHECK (status IN (1, 2, 3))
);

CREATE INDEX IF NOT EXISTS idx_device_child_host_device_id ON public.device_child(host_device_id);
CREATE INDEX IF NOT EXISTS idx_device_child_child_sn ON public.device_child(child_sn);

DROP TRIGGER IF EXISTS trg_device_child_set_updated_at ON public.device_child;
CREATE TRIGGER trg_device_child_set_updated_at
BEFORE UPDATE ON public.device_child
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TABLE IF NOT EXISTS public.device_child_shadow (
  id BIGSERIAL PRIMARY KEY,
  child_id BIGINT NOT NULL REFERENCES public.device_child(id) ON DELETE CASCADE,
  host_device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  child_key VARCHAR(64) NOT NULL,
  reported JSONB,
  desired JSONB,
  metadata JSONB,
  version BIGINT NOT NULL DEFAULT 0,
  last_report_time TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_device_child_shadow_child_id UNIQUE(child_id)
);

CREATE INDEX IF NOT EXISTS idx_device_child_shadow_host_device_id ON public.device_child_shadow(host_device_id);
CREATE INDEX IF NOT EXISTS idx_device_child_shadow_child_key ON public.device_child_shadow(child_key);

DROP TRIGGER IF EXISTS trg_device_child_shadow_set_updated_at ON public.device_child_shadow;
CREATE TRIGGER trg_device_child_shadow_set_updated_at
BEFORE UPDATE ON public.device_child_shadow
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TABLE IF NOT EXISTS public.device_report_batch (
  id BIGSERIAL PRIMARY KEY,
  host_device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  sn VARCHAR(64) NOT NULL,
  report_id VARCHAR(128) NOT NULL,
  source VARCHAR(16) NOT NULL DEFAULT 'http',
  reported_at TIMESTAMP NULL,
  received_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  is_history BOOLEAN NOT NULL DEFAULT FALSE,
  child_count INT NOT NULL DEFAULT 0,
  event_count INT NOT NULL DEFAULT 0,
  payload JSONB,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_device_report_batch_host_report UNIQUE(host_device_id, report_id)
);

CREATE INDEX IF NOT EXISTS idx_device_report_batch_host_device_id ON public.device_report_batch(host_device_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_device_report_batch_sn ON public.device_report_batch(sn, created_at DESC);

CREATE TABLE IF NOT EXISTS public.device_report_event (
  id BIGSERIAL PRIMARY KEY,
  batch_id BIGINT NOT NULL REFERENCES public.device_report_batch(id) ON DELETE CASCADE,
  host_device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  child_id BIGINT NULL REFERENCES public.device_child(id) ON DELETE SET NULL,
  child_key VARCHAR(64) NOT NULL DEFAULT '',
  event_kind VARCHAR(16) NOT NULL DEFAULT 'host',
  event_time TIMESTAMP NULL,
  payload JSONB,
  applied_to_shadow BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_report_event_batch_id ON public.device_report_event(batch_id, id ASC);
CREATE INDEX IF NOT EXISTS idx_device_report_event_host_device_id ON public.device_report_event(host_device_id, created_at DESC);

COMMENT ON TABLE public.device_child IS '主机下属子设备表';
COMMENT ON TABLE public.device_child_shadow IS '子设备影子表';
COMMENT ON TABLE public.device_report_batch IS '设备状态上报批次（含历史补传）';
COMMENT ON TABLE public.device_report_event IS '设备状态上报事件明细（主机/子设备/历史）';
