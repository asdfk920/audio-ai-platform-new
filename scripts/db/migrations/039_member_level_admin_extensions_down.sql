SET search_path TO public;

ALTER TABLE public.member_level
  DROP COLUMN IF EXISTS version,
  DROP COLUMN IF EXISTS extra_config,
  DROP COLUMN IF EXISTS price_package_hint,
  DROP COLUMN IF EXISTS default_validity_days,
  DROP COLUMN IF EXISTS acquire_type;
