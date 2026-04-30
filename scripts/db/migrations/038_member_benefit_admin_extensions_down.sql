SET search_path TO public;

ALTER TABLE public.member_benefit
  DROP COLUMN IF EXISTS extra_config,
  DROP COLUMN IF EXISTS effect_end_at,
  DROP COLUMN IF EXISTS effect_start_at,
  DROP COLUMN IF EXISTS tags,
  DROP COLUMN IF EXISTS version,
  DROP COLUMN IF EXISTS lifecycle_status,
  DROP COLUMN IF EXISTS benefit_type;
