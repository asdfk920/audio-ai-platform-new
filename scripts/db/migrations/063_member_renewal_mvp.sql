-- 063: 会员续费 MVP — 订单金额明细、biz_scene、套餐标价、自动续费占位
SET search_path TO public;

-- 订单：原价、优惠、业务场景（服务端推断，不信任客户端）
ALTER TABLE public.order_master
    ADD COLUMN IF NOT EXISTS original_amount_cent BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS discount_cent BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS biz_scene VARCHAR(32) NOT NULL DEFAULT 'new';

UPDATE public.order_master
SET original_amount_cent = amount_cent
WHERE original_amount_cent = 0;

COMMENT ON COLUMN public.order_master.original_amount_cent IS '下单时套餐原价（分），含未优惠标价';
COMMENT ON COLUMN public.order_master.discount_cent IS '优惠抵扣（分）';
COMMENT ON COLUMN public.order_master.biz_scene IS 'new 首购 renew_active 有效期内续费 renew_expired 过期后续';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'order_master_biz_scene_check'
    ) THEN
        ALTER TABLE public.order_master
        ADD CONSTRAINT order_master_biz_scene_check CHECK (biz_scene IN ('new', 'renew_active', 'renew_expired'));
    END IF;
END $$;

-- 套餐：标价（可与实付 price_cent 不同，运营配置）
ALTER TABLE public.member_package
    ADD COLUMN IF NOT EXISTS list_price_cent BIGINT;

UPDATE public.member_package SET list_price_cent = price_cent WHERE list_price_cent IS NULL;

COMMENT ON COLUMN public.member_package.list_price_cent IS '原价/标价（分）；实付为 price_cent';

-- 用户会员：自动续费占位（真实代扣后续对接支付签约）
ALTER TABLE public.user_member
    ADD COLUMN IF NOT EXISTS auto_renew SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS auto_renew_package_code VARCHAR(64),
    ADD COLUMN IF NOT EXISTS auto_renew_pay_type SMALLINT,
    ADD COLUMN IF NOT EXISTS auto_renew_updated_at TIMESTAMP;

COMMENT ON COLUMN public.user_member.auto_renew IS '1 开启自动续费意图 0 关闭（非真实扣款，仅 MVP 占位）';
COMMENT ON COLUMN public.user_member.auto_renew_package_code IS '自动续费使用的套餐编码';
COMMENT ON COLUMN public.user_member.auto_renew_pay_type IS '1微信 2支付宝 3余额，与 order_master.pay_type 一致';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'user_member_auto_renew_pay_type_check'
    ) THEN
        ALTER TABLE public.user_member
        ADD CONSTRAINT user_member_auto_renew_pay_type_check
        CHECK (auto_renew_pay_type IS NULL OR auto_renew_pay_type IN (1, 2, 3));
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'user_member_auto_renew_check'
    ) THEN
        ALTER TABLE public.user_member
        ADD CONSTRAINT user_member_auto_renew_check CHECK (auto_renew IN (0, 1));
    END IF;
END $$;
