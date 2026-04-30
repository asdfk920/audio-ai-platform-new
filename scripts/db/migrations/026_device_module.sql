-- 设备模块表结构（PostgreSQL）
-- 由需求 DDL（MySQL 风格）迁移改写：BIGSERIAL + JSONB + 触发器维护 updated_at

SET search_path TO public;

-- 通用 updated_at 维护函数（若已存在则复用）
CREATE OR REPLACE FUNCTION public.set_updated_at()
RETURNS trigger AS $$
BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 1) device 设备主表
CREATE TABLE IF NOT EXISTS public.device (
  id BIGSERIAL PRIMARY KEY,
  sn VARCHAR(64) NOT NULL UNIQUE,
  product_key VARCHAR(64) NOT NULL,
  device_secret VARCHAR(128) NOT NULL,
  firmware_version VARCHAR(32) NOT NULL DEFAULT '',
  hardware_version VARCHAR(32) NOT NULL DEFAULT '',
  model VARCHAR(64) NOT NULL DEFAULT '',
  mac VARCHAR(32) NOT NULL DEFAULT '',
  ip VARCHAR(45) NOT NULL DEFAULT '',
  online_status SMALLINT NOT NULL DEFAULT 0,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT device_online_status_check CHECK (online_status IN (0, 1)),
  CONSTRAINT device_status_check CHECK (status IN (1, 2, 3))
);

CREATE INDEX IF NOT EXISTS idx_device_sn ON public.device(sn);
CREATE INDEX IF NOT EXISTS idx_device_product_key ON public.device(product_key);

DROP TRIGGER IF EXISTS trg_device_set_updated_at ON public.device;
CREATE TRIGGER trg_device_set_updated_at
BEFORE UPDATE ON public.device
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- 2) user_device_bind 用户 - 设备绑定表
CREATE TABLE IF NOT EXISTS public.user_device_bind (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  sn VARCHAR(64) NOT NULL,
  alias VARCHAR(32) NOT NULL DEFAULT '',
  is_default SMALLINT NOT NULL DEFAULT 0,
  bind_type SMALLINT NOT NULL DEFAULT 1,
  status SMALLINT NOT NULL DEFAULT 1,
  bound_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  unbound_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_user_device_bind UNIQUE(user_id, device_id),
  CONSTRAINT user_device_bind_is_default_check CHECK (is_default IN (0, 1)),
  CONSTRAINT user_device_bind_bind_type_check CHECK (bind_type IN (1, 2)),
  CONSTRAINT user_device_bind_status_check CHECK (status IN (1, 2))
);

CREATE INDEX IF NOT EXISTS idx_user_device_bind_user_id ON public.user_device_bind(user_id);
CREATE INDEX IF NOT EXISTS idx_user_device_bind_sn ON public.user_device_bind(sn);

DROP TRIGGER IF EXISTS trg_user_device_bind_set_updated_at ON public.user_device_bind;
CREATE TRIGGER trg_user_device_bind_set_updated_at
BEFORE UPDATE ON public.user_device_bind
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- 3) device_shadow 设备影子表
CREATE TABLE IF NOT EXISTS public.device_shadow (
  id BIGSERIAL PRIMARY KEY,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  sn VARCHAR(64) NOT NULL,
  reported JSONB NULL,
  desired JSONB NULL,
  last_report_time TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_device_shadow_device_id UNIQUE(device_id)
);

CREATE INDEX IF NOT EXISTS idx_device_shadow_sn ON public.device_shadow(sn);

DROP TRIGGER IF EXISTS trg_device_shadow_set_updated_at ON public.device_shadow;
CREATE TRIGGER trg_device_shadow_set_updated_at
BEFORE UPDATE ON public.device_shadow
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- 4) device_state_log 设备状态上报日志表
CREATE TABLE IF NOT EXISTS public.device_state_log (
  id BIGSERIAL PRIMARY KEY,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  sn VARCHAR(64) NOT NULL,
  battery INT NOT NULL DEFAULT 0,
  volume INT NOT NULL DEFAULT 0,
  online_status SMALLINT NOT NULL DEFAULT 0,
  network VARCHAR(16) NOT NULL DEFAULT '',
  rssi INT NOT NULL DEFAULT 0,
  ip VARCHAR(45) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT device_state_log_online_status_check CHECK (online_status IN (0, 1))
);

CREATE INDEX IF NOT EXISTS idx_device_state_log_device_id ON public.device_state_log(device_id);
CREATE INDEX IF NOT EXISTS idx_device_state_log_sn ON public.device_state_log(sn);

-- 5) device_instruction 设备指令表
CREATE TABLE IF NOT EXISTS public.device_instruction (
  id BIGSERIAL PRIMARY KEY,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  sn VARCHAR(64) NOT NULL,
  user_id BIGINT NOT NULL DEFAULT 0,
  cmd VARCHAR(64) NOT NULL,
  params JSONB NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  result JSONB NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT device_instruction_status_check CHECK (status IN (1, 2, 3, 4))
);

CREATE INDEX IF NOT EXISTS idx_device_instruction_device_id ON public.device_instruction(device_id);
CREATE INDEX IF NOT EXISTS idx_device_instruction_status ON public.device_instruction(status);

DROP TRIGGER IF EXISTS trg_device_instruction_set_updated_at ON public.device_instruction;
CREATE TRIGGER trg_device_instruction_set_updated_at
BEFORE UPDATE ON public.device_instruction
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- 6) ota_firmware OTA 固件表
CREATE TABLE IF NOT EXISTS public.ota_firmware (
  id BIGSERIAL PRIMARY KEY,
  product_key VARCHAR(64) NOT NULL,
  version VARCHAR(32) NOT NULL,
  file_url VARCHAR(255) NOT NULL,
  file_size BIGINT NOT NULL DEFAULT 0,
  md5 VARCHAR(64) NOT NULL,
  upgrade_type SMALLINT NOT NULL DEFAULT 1,
  publish_status SMALLINT NOT NULL DEFAULT 1,
  release_note TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT ota_firmware_upgrade_type_check CHECK (upgrade_type IN (1, 2)),
  CONSTRAINT ota_firmware_publish_status_check CHECK (publish_status IN (1, 2)),
  CONSTRAINT uk_ota_firmware_product_version UNIQUE(product_key, version)
);

CREATE INDEX IF NOT EXISTS idx_ota_firmware_product_key ON public.ota_firmware(product_key);
CREATE INDEX IF NOT EXISTS idx_ota_firmware_version ON public.ota_firmware(version);

-- 7) ota_upgrade_task OTA 升级任务表
CREATE TABLE IF NOT EXISTS public.ota_upgrade_task (
  id BIGSERIAL PRIMARY KEY,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  sn VARCHAR(64) NOT NULL,
  firmware_id BIGINT NOT NULL REFERENCES public.ota_firmware(id) ON DELETE RESTRICT,
  from_version VARCHAR(32) NOT NULL,
  to_version VARCHAR(32) NOT NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  progress INT NOT NULL DEFAULT 0,
  error_msg VARCHAR(255) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT ota_upgrade_task_status_check CHECK (status IN (1, 2, 3, 4)),
  CONSTRAINT ota_upgrade_task_progress_check CHECK (progress >= 0 AND progress <= 100)
);

CREATE INDEX IF NOT EXISTS idx_ota_upgrade_task_device_id ON public.ota_upgrade_task(device_id);
CREATE INDEX IF NOT EXISTS idx_ota_upgrade_task_status ON public.ota_upgrade_task(status);

DROP TRIGGER IF EXISTS trg_ota_upgrade_task_set_updated_at ON public.ota_upgrade_task;
CREATE TRIGGER trg_ota_upgrade_task_set_updated_at
BEFORE UPDATE ON public.ota_upgrade_task
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- 8) device_certificate 设备证书表
CREATE TABLE IF NOT EXISTS public.device_certificate (
  id BIGSERIAL PRIMARY KEY,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  sn VARCHAR(64) NOT NULL,
  cert TEXT NOT NULL,
  private_key TEXT NOT NULL,
  not_before TIMESTAMP NULL,
  not_after TIMESTAMP NULL,
  status SMALLINT NOT NULL DEFAULT 1,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_device_certificate_device_id UNIQUE(device_id),
  CONSTRAINT device_certificate_status_check CHECK (status IN (1, 2, 3))
);

CREATE INDEX IF NOT EXISTS idx_device_certificate_sn ON public.device_certificate(sn);

-- 证书表没有 updated_at（按需求）；如后续需要可再加

-- 9) device_event_log 设备事件日志表
CREATE TABLE IF NOT EXISTS public.device_event_log (
  id BIGSERIAL PRIMARY KEY,
  device_id BIGINT NOT NULL REFERENCES public.device(id) ON DELETE CASCADE,
  sn VARCHAR(64) NOT NULL,
  event_type VARCHAR(32) NOT NULL,
  content VARCHAR(255) NOT NULL,
  extra JSONB NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_device_event_log_device_id ON public.device_event_log(device_id);
CREATE INDEX IF NOT EXISTS idx_device_event_log_event_type ON public.device_event_log(event_type);

-- 10) device_group 设备分组
CREATE TABLE IF NOT EXISTS public.device_group (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
  name VARCHAR(32) NOT NULL,
  remark VARCHAR(64) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT uk_device_group_user_name UNIQUE(user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_device_group_user_id ON public.device_group(user_id);

