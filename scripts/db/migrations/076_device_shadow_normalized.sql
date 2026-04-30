-- 规范化设备影子：主表 + 电量/位置/配置子表（与 device_shadow JSONB 并存）
SET search_path TO public;

-- 1) 影子主表片段（1:1 device）
CREATE TABLE IF NOT EXISTS public.device_shadow_profile (
  device_id BIGINT PRIMARY KEY REFERENCES public.device(id) ON DELETE CASCADE,
  firmware_version VARCHAR(32) NOT NULL DEFAULT '',
  hardware_version VARCHAR(32) NOT NULL DEFAULT '',
  online_status SMALLINT NOT NULL DEFAULT 0,
  offline_at TIMESTAMPTZ NULL,
  last_active_at TIMESTAMPTZ NULL,
  fw_upgraded_at TIMESTAMPTZ NULL,
  network_type VARCHAR(32) NOT NULL DEFAULT '',
  rssi INTEGER NULL,
  product_key VARCHAR(64) NOT NULL DEFAULT '',
  first_online_at TIMESTAMPTZ NULL,
  offline_reason VARCHAR(128) NOT NULL DEFAULT '',
  reported_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT device_shadow_profile_online_check CHECK (online_status IN (0, 1))
);

CREATE INDEX IF NOT EXISTS idx_device_shadow_profile_online ON public.device_shadow_profile(online_status);

DROP TRIGGER IF EXISTS trg_device_shadow_profile_set_updated_at ON public.device_shadow_profile;
CREATE TRIGGER trg_device_shadow_profile_set_updated_at
BEFORE UPDATE ON public.device_shadow_profile
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

COMMENT ON TABLE public.device_shadow_profile IS '设备影子规范化主表（固件/在线/网络等）';

-- 2) 电量快照
CREATE TABLE IF NOT EXISTS public.device_shadow_battery (
  device_id BIGINT PRIMARY KEY REFERENCES public.device(id) ON DELETE CASCADE,
  main_percent SMALLINT NULL,
  speaker_percent SMALLINT NULL,
  charging SMALLINT NOT NULL DEFAULT 0,
  est_remaining_sec BIGINT NULL,
  low_threshold SMALLINT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT device_shadow_battery_charging_check CHECK (charging IN (0, 1))
);

DROP TRIGGER IF EXISTS trg_device_shadow_battery_set_updated_at ON public.device_shadow_battery;
CREATE TRIGGER trg_device_shadow_battery_set_updated_at
BEFORE UPDATE ON public.device_shadow_battery
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

COMMENT ON TABLE public.device_shadow_battery IS '设备影子电量快照';

-- 3) 位置快照
CREATE TABLE IF NOT EXISTS public.device_shadow_location (
  device_id BIGINT PRIMARY KEY REFERENCES public.device(id) ON DELETE CASCADE,
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  location_mode VARCHAR(32) NOT NULL DEFAULT '',
  accuracy_m DOUBLE PRECISION NULL,
  geofence_status VARCHAR(32) NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

DROP TRIGGER IF EXISTS trg_device_shadow_location_set_updated_at ON public.device_shadow_location;
CREATE TRIGGER trg_device_shadow_location_set_updated_at
BEFORE UPDATE ON public.device_shadow_location
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

COMMENT ON TABLE public.device_shadow_location IS '设备影子位置快照';

-- 4) 配置（多行：按 config_type）
CREATE TABLE IF NOT EXISTS public.device_shadow_config (
  id BIGSERIAL PRIMARY KEY,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  config_type VARCHAR(64) NOT NULL,
  desired TEXT NOT NULL DEFAULT '',
  reported TEXT NOT NULL DEFAULT '',
  sync_status SMALLINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uq_device_shadow_config_device_type UNIQUE(device_id, config_type),
  CONSTRAINT device_shadow_config_sync_status_check CHECK (sync_status IN (0, 1))
);

CREATE INDEX IF NOT EXISTS idx_device_shadow_config_device ON public.device_shadow_config(device_id);

DROP TRIGGER IF EXISTS trg_device_shadow_config_set_updated_at ON public.device_shadow_config;
CREATE TRIGGER trg_device_shadow_config_set_updated_at
BEFORE UPDATE ON public.device_shadow_config
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

COMMENT ON TABLE public.device_shadow_config IS '设备影子配置（desired/reported JSON 文本），sync_status: 0待同步 1已同步';
