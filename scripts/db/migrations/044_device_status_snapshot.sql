-- 设备状态快照表：MQTT/过期离线等异步写入，与 Redis 影子配合审计与历史
SET search_path TO public;

CREATE TABLE IF NOT EXISTS public.device_status (
  id BIGSERIAL PRIMARY KEY,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  sn VARCHAR(64) NOT NULL,
  run_state VARCHAR(32) NOT NULL DEFAULT '',
  battery INT NOT NULL DEFAULT 0,
  firmware_version VARCHAR(64) NOT NULL DEFAULT '',
  online_status SMALLINT NOT NULL DEFAULT 0,
  last_active_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  source VARCHAR(16) NOT NULL DEFAULT 'mqtt',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT device_status_online_check CHECK (online_status IN (0, 1))
);

CREATE INDEX IF NOT EXISTS idx_device_status_sn_created ON public.device_status (sn, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_device_status_device_created ON public.device_status (device_id, created_at DESC);

COMMENT ON TABLE public.device_status IS '设备状态异步落库（MQTT 上报、在线键过期离线等）';
COMMENT ON COLUMN public.device_status.source IS 'mqtt | redis_expire | http 等';
