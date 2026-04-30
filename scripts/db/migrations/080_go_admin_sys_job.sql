-- go-admin 定时任务表（JobCore 启动时查询 sys_job；缺表会报 SQLSTATE 42P01）
-- 字段与 admin/app/jobs/models/sys_job.go + common/models ControlBy、ModelTime 一致（GORM 默认 snake_case）
SET search_path TO public;

CREATE TABLE IF NOT EXISTS public.sys_job (
  job_id SERIAL PRIMARY KEY,
  job_name VARCHAR(255) NOT NULL DEFAULT '',
  job_group VARCHAR(255) NOT NULL DEFAULT '',
  job_type INT NOT NULL DEFAULT 0,
  cron_expression VARCHAR(255) NOT NULL DEFAULT '',
  invoke_target VARCHAR(255) NOT NULL DEFAULT '',
  args VARCHAR(255) NOT NULL DEFAULT '',
  misfire_policy INT NOT NULL DEFAULT 0,
  concurrent INT NOT NULL DEFAULT 0,
  status INT NOT NULL DEFAULT 0,
  entry_id INT NOT NULL DEFAULT 0,
  create_by INT NOT NULL DEFAULT 0,
  update_by INT NOT NULL DEFAULT 0,
  delete_by INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_sys_job_deleted_at ON public.sys_job (deleted_at);
CREATE INDEX IF NOT EXISTS idx_sys_job_create_by ON public.sys_job (create_by);
CREATE INDEX IF NOT EXISTS idx_sys_job_update_by ON public.sys_job (update_by);
CREATE INDEX IF NOT EXISTS idx_sys_job_delete_by ON public.sys_job (delete_by);

COMMENT ON TABLE public.sys_job IS 'go-admin 定时任务（cron）';
