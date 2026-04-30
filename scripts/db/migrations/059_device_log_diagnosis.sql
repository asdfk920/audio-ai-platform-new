SET search_path TO public;

CREATE TABLE IF NOT EXISTS public.device_log_batch (
  id BIGSERIAL PRIMARY KEY,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  sn VARCHAR(64) NOT NULL,
  upload_id VARCHAR(128) NOT NULL,
  source VARCHAR(16) NOT NULL DEFAULT 'http',
  trigger_type VARCHAR(32) NOT NULL DEFAULT 'periodic',
  report_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  log_count INT NOT NULL DEFAULT 0,
  summary JSONB,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_device_log_batch_device_upload UNIQUE(device_id, upload_id)
);

CREATE INDEX IF NOT EXISTS idx_device_log_batch_device_time
  ON public.device_log_batch(device_id, report_time DESC);

CREATE TABLE IF NOT EXISTS public.device_log (
  id BIGSERIAL PRIMARY KEY,
  batch_id BIGINT NULL REFERENCES public.device_log_batch(id) ON DELETE SET NULL,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  sn VARCHAR(64) NOT NULL,
  log_type VARCHAR(32) NOT NULL,
  log_level VARCHAR(16) NOT NULL DEFAULT 'info',
  module VARCHAR(64) NOT NULL DEFAULT '',
  content TEXT NOT NULL,
  error_code INT NULL,
  extra JSONB,
  report_time TIMESTAMP NOT NULL,
  report_source VARCHAR(32) NOT NULL DEFAULT 'device',
  ip_address VARCHAR(64) NOT NULL DEFAULT '',
  processed SMALLINT NOT NULL DEFAULT 0,
  alert_sent SMALLINT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at TIMESTAMP NULL,
  CONSTRAINT device_log_level_check CHECK (log_level IN ('debug','info','warn','error','fatal')),
  CONSTRAINT device_log_processed_check CHECK (processed IN (0, 1, 2)),
  CONSTRAINT device_log_alert_sent_check CHECK (alert_sent IN (0, 1))
);

CREATE INDEX IF NOT EXISTS idx_device_log_device_time
  ON public.device_log(device_id, report_time DESC);
CREATE INDEX IF NOT EXISTS idx_device_log_device_level
  ON public.device_log(device_id, log_level, report_time DESC);
CREATE INDEX IF NOT EXISTS idx_device_log_device_type
  ON public.device_log(device_id, log_type, report_time DESC);

DROP TRIGGER IF EXISTS trg_device_log_set_updated_at ON public.device_log;
CREATE TRIGGER trg_device_log_set_updated_at
BEFORE UPDATE ON public.device_log
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

CREATE TABLE IF NOT EXISTS public.device_diagnosis (
  id BIGSERIAL PRIMARY KEY,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  sn VARCHAR(64) NOT NULL,
  instruction_id BIGINT NULL REFERENCES public.device_instruction(id) ON DELETE SET NULL,
  diag_type VARCHAR(32) NOT NULL DEFAULT 'full',
  status SMALLINT NOT NULL DEFAULT 0,
  params JSONB,
  result JSONB,
  summary TEXT NOT NULL DEFAULT '',
  failure_reason TEXT NOT NULL DEFAULT '',
  total_items INT NOT NULL DEFAULT 0,
  normal_items INT NOT NULL DEFAULT 0,
  abnormal_items INT NOT NULL DEFAULT 0,
  health_score SMALLINT NOT NULL DEFAULT 0,
  timeout_seconds INT NOT NULL DEFAULT 300,
  operator VARCHAR(64) NOT NULL DEFAULT '',
  ip_address VARCHAR(64) NOT NULL DEFAULT '',
  report_time TIMESTAMP NULL,
  completed_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at TIMESTAMP NULL,
  CONSTRAINT device_diagnosis_status_check CHECK (status IN (0, 1, 2, 3)),
  CONSTRAINT device_diagnosis_health_score_check CHECK (health_score >= 0 AND health_score <= 100)
);

CREATE INDEX IF NOT EXISTS idx_device_diagnosis_device_created
  ON public.device_diagnosis(device_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_device_diagnosis_instruction_id
  ON public.device_diagnosis(instruction_id);
CREATE INDEX IF NOT EXISTS idx_device_diagnosis_status
  ON public.device_diagnosis(status, created_at DESC);

DROP TRIGGER IF EXISTS trg_device_diagnosis_set_updated_at ON public.device_diagnosis;
CREATE TRIGGER trg_device_diagnosis_set_updated_at
BEFORE UPDATE ON public.device_diagnosis
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

COMMENT ON TABLE public.device_log_batch IS '设备日志上传批次';
COMMENT ON TABLE public.device_log IS '设备运行日志';
COMMENT ON TABLE public.device_diagnosis IS '设备远程诊断记录';
