-- 设备定时状态上报日志（电量/存储/UWB/声学等）
SET search_path TO public;

CREATE TABLE IF NOT EXISTS public.device_status_logs (
  id BIGSERIAL PRIMARY KEY,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  sn VARCHAR(64) NOT NULL,
  battery_level INT NOT NULL DEFAULT 0 CHECK (battery_level >= 0 AND battery_level <= 100),
  storage_used BIGINT NOT NULL DEFAULT 0 CHECK (storage_used >= 0),
  storage_total BIGINT NOT NULL DEFAULT 0 CHECK (storage_total >= 0),
  speaker_count INT NOT NULL DEFAULT 0 CHECK (speaker_count >= 0),
  uwb_x DOUBLE PRECISION NULL,
  uwb_y DOUBLE PRECISION NULL,
  uwb_z DOUBLE PRECISION NULL,
  acoustic_calibrated SMALLINT NOT NULL DEFAULT 0 CHECK (acoustic_calibrated IN (0, 1)),
  acoustic_offset DOUBLE PRECISION NULL,
  reported_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_status_logs_sn_created ON public.device_status_logs (sn, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_device_status_logs_device_created ON public.device_status_logs (device_id, created_at DESC);

COMMENT ON TABLE public.device_status_logs IS '设备定时状态上报历史（HTTP 等）';
COMMENT ON COLUMN public.device_status_logs.reported_at IS '设备本地采集时间';
COMMENT ON COLUMN public.device_status_logs.created_at IS '服务端接收时间';
