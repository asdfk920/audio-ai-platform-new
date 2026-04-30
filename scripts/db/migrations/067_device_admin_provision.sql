-- 后台预注册设备：展示名称/备注；异步批量导入任务表；非空 MAC 唯一（预注册可为空）
SET search_path TO public;

ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS admin_display_name VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE public.device
  ADD COLUMN IF NOT EXISTS admin_remark TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN public.device.admin_display_name IS '后台展示用设备名称';
COMMENT ON COLUMN public.device.admin_remark IS '管理员备注';

-- 仅非空 MAC 唯一，允许多条预注册设备 mac 为空
CREATE UNIQUE INDEX IF NOT EXISTS idx_device_mac_unique_nonempty
  ON public.device (mac)
  WHERE mac IS NOT NULL AND btrim(mac) <> '';

CREATE TABLE IF NOT EXISTS public.device_import_job (
  id BIGSERIAL PRIMARY KEY,
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  total INT NOT NULL DEFAULT 0,
  processed INT NOT NULL DEFAULT 0,
  success_count INT NOT NULL DEFAULT 0,
  fail_count INT NOT NULL DEFAULT 0,
  error_message TEXT NOT NULL DEFAULT '',
  failure_detail_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  result_file_path TEXT NOT NULL DEFAULT '',
  temp_source_path TEXT NOT NULL DEFAULT '',
  created_by BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  finished_at TIMESTAMP NULL,
  CONSTRAINT device_import_job_status_check CHECK (status IN ('pending', 'running', 'success', 'failed', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_device_import_job_created_by ON public.device_import_job (created_by, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_device_import_job_status ON public.device_import_job (status);

COMMENT ON TABLE public.device_import_job IS '后台批量导入设备异步任务';
COMMENT ON COLUMN public.device_import_job.failure_detail_json IS '失败明细 [{row,reason}]';
COMMENT ON COLUMN public.device_import_job.result_file_path IS '成功行密钥 CSV 临时文件路径（不落库明文）';

DROP TRIGGER IF EXISTS trg_device_import_job_set_updated_at ON public.device_import_job;
CREATE TRIGGER trg_device_import_job_set_updated_at
BEFORE UPDATE ON public.device_import_job
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();
