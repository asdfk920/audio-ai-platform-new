-- 会员权益表扩展：类型、生命周期、版本、生效时间、扩展 JSON（后台运营）
SET search_path TO public;

ALTER TABLE public.member_benefit
  ADD COLUMN IF NOT EXISTS benefit_type SMALLINT NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS lifecycle_status SMALLINT,
  ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS tags VARCHAR(255) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS effect_start_at TIMESTAMP NULL,
  ADD COLUMN IF NOT EXISTS effect_end_at TIMESTAMP NULL,
  ADD COLUMN IF NOT EXISTS extra_config JSONB NOT NULL DEFAULT '{}'::jsonb;

-- 与旧 status 对齐：启用 -> 已上线，禁用 -> 已下线
UPDATE public.member_benefit
   SET lifecycle_status = CASE WHEN status = 1 THEN 2 ELSE 3 END
 WHERE lifecycle_status IS NULL;

ALTER TABLE public.member_benefit
  ALTER COLUMN lifecycle_status SET NOT NULL,
  ALTER COLUMN lifecycle_status SET DEFAULT 2;

COMMENT ON COLUMN public.member_benefit.benefit_type IS '1基础功能 2增值服务 3限时活动';
COMMENT ON COLUMN public.member_benefit.lifecycle_status IS '0草稿 1待发布 2已上线 3已下线';
COMMENT ON COLUMN public.member_benefit.version IS '权益配置版本号（人工递增）';
COMMENT ON COLUMN public.member_benefit.tags IS '逗号分隔标签，如 核心功能,新用户';
COMMENT ON COLUMN public.member_benefit.effect_start_at IS '生效开始（空表示不限制）';
COMMENT ON COLUMN public.member_benefit.effect_end_at IS '生效结束（空表示不限制）';
COMMENT ON COLUMN public.member_benefit.extra_config IS '互斥/依赖/限额等 JSON，如 mutual_exclusive_codes, dependent_codes';

-- C 端展示沿用 status：已上线 -> 1，其余 -> 0
UPDATE public.member_benefit
   SET status = CASE WHEN lifecycle_status = 2 THEN 1 ELSE 0 END;
