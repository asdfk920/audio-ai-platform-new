-- 会员套餐、订单、支付流水、会员变更日志；用户余额（分）
SET search_path TO public;

-- 触发器依赖（若已执行过 026 迁移，此处为幂等覆盖）
CREATE OR REPLACE FUNCTION public.set_updated_at()
RETURNS trigger AS $$
BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 用户余额（分），用于余额支付
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS balance_cent BIGINT NOT NULL DEFAULT 0;

-- 会员套餐（可售卖 SKU）
CREATE TABLE IF NOT EXISTS public.member_package (
    id BIGSERIAL PRIMARY KEY,
    package_code VARCHAR(64) NOT NULL,
    package_name VARCHAR(64) NOT NULL,
    level_code VARCHAR(32) NOT NULL REFERENCES public.member_level(level_code) ON DELETE RESTRICT,
    price_cent BIGINT NOT NULL CHECK (price_cent >= 0),
    duration_days INT NOT NULL CHECK (duration_days > 0),
    status SMALLINT NOT NULL DEFAULT 1,
    sort INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT member_package_code_unique UNIQUE (package_code),
    CONSTRAINT member_package_status_check CHECK (status IN (0, 1))
);

CREATE INDEX IF NOT EXISTS idx_member_package_status ON public.member_package(status);

COMMENT ON TABLE public.member_package IS '会员售卖套餐';
COMMENT ON COLUMN public.member_package.status IS '1上架 0下架';
COMMENT ON COLUMN public.member_package.duration_days IS '单次购买延长天数';

DROP TRIGGER IF EXISTS trg_member_package_set_updated_at ON public.member_package;
CREATE TRIGGER trg_member_package_set_updated_at
BEFORE UPDATE ON public.member_package
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- 订单主表
CREATE TABLE IF NOT EXISTS public.order_master (
    id BIGSERIAL PRIMARY KEY,
    order_no VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    order_type VARCHAR(32) NOT NULL DEFAULT 'member',
    package_code VARCHAR(64) NOT NULL,
    level_code VARCHAR(32) NOT NULL,
    duration_days INT NOT NULL,
    amount_cent BIGINT NOT NULL,
    pay_type SMALLINT NOT NULL,
    pay_status SMALLINT NOT NULL DEFAULT 0,
    channel_trade_no VARCHAR(128),
    paid_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT order_master_order_no_unique UNIQUE (order_no),
    CONSTRAINT order_master_pay_type_check CHECK (pay_type IN (1, 2, 3)),
    CONSTRAINT order_master_pay_status_check CHECK (pay_status IN (0, 1, 2))
);

CREATE INDEX IF NOT EXISTS idx_order_master_user_id ON public.order_master(user_id);
CREATE INDEX IF NOT EXISTS idx_order_master_created ON public.order_master(created_at DESC);

COMMENT ON TABLE public.order_master IS '订单主表';
COMMENT ON COLUMN public.order_master.pay_type IS '1微信 2支付宝 3余额';
COMMENT ON COLUMN public.order_master.pay_status IS '0待支付 1成功 2关闭';

DROP TRIGGER IF EXISTS trg_order_master_set_updated_at ON public.order_master;
CREATE TRIGGER trg_order_master_set_updated_at
BEFORE UPDATE ON public.order_master
FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();

-- 支付流水
CREATE TABLE IF NOT EXISTS public.pay_log (
    id BIGSERIAL PRIMARY KEY,
    order_no VARCHAR(64) NOT NULL,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    pay_type SMALLINT NOT NULL,
    amount_cent BIGINT NOT NULL,
    trade_no VARCHAR(128) NOT NULL DEFAULT '',
    pay_status SMALLINT NOT NULL DEFAULT 1,
    raw_notify TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pay_log_pay_type_check CHECK (pay_type IN (1, 2, 3))
);

CREATE INDEX IF NOT EXISTS idx_pay_log_order_no ON public.pay_log(order_no);

COMMENT ON TABLE public.pay_log IS '支付结果流水';

-- 会员开通/续费日志
CREATE TABLE IF NOT EXISTS public.member_pay_log (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    order_no VARCHAR(64) NOT NULL,
    package_code VARCHAR(64) NOT NULL,
    level_code VARCHAR(32) NOT NULL,
    duration_days INT NOT NULL,
    old_expire_at TIMESTAMP,
    new_expire_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_member_pay_log_user ON public.member_pay_log(user_id, created_at DESC);

COMMENT ON TABLE public.member_pay_log IS '会员付费开通/续费审计日志';

-- 示例套餐（可按运营改价）
INSERT INTO public.member_package (package_code, package_name, level_code, price_cent, duration_days, status, sort)
VALUES
('vip_monthly', 'VIP月卡', 'vip', 990, 30, 1, 1),
('vip_quarterly', 'VIP季卡', 'vip', 2680, 90, 1, 2),
('vip_yearly', 'VIP年卡', 'vip', 9980, 365, 1, 3)
ON CONFLICT (package_code) DO NOTHING;
