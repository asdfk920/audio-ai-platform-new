-- 会员等级表扩展：获取方式、价格文案、默认有效期、扩展 JSON（后台运营）
SET search_path TO public;

ALTER TABLE public.member_level
  ADD COLUMN IF NOT EXISTS acquire_type SMALLINT NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS default_validity_days INT NULL,
  ADD COLUMN IF NOT EXISTS price_package_hint VARCHAR(512) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS extra_config JSONB NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 1;

COMMENT ON COLUMN public.member_level.acquire_type IS '1注册即有 2付费购买 3活动赠送 4邀请解锁';
COMMENT ON COLUMN public.member_level.default_validity_days IS '默认有效天数（展示/新建用户参考，实际以订单为准）';
COMMENT ON COLUMN public.member_level.price_package_hint IS '价格/套餐运营文案，如月付/年付';
COMMENT ON COLUMN public.member_level.extra_config IS '升级规则、备注等 JSON';
COMMENT ON COLUMN public.member_level.version IS '配置版本号';

-- 与历史「适用」列对齐的粗略默认值（可按运营再改）
UPDATE public.member_level SET acquire_type = 1 WHERE level_code = 'ordinary';
UPDATE public.member_level SET acquire_type = 2 WHERE level_code IN ('vip', 'year_vip');
UPDATE public.member_level SET acquire_type = 2 WHERE level_code = 'svip';
