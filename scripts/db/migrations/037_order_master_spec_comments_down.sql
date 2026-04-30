-- 037 回滚：恢复 036 中的表/列注释（037 新增的列注释清空）
SET search_path TO public;

COMMENT ON TABLE public.order_master IS '订单主表';
COMMENT ON COLUMN public.order_master.order_no IS NULL;
COMMENT ON COLUMN public.order_master.user_id IS NULL;
COMMENT ON COLUMN public.order_master.package_code IS NULL;
COMMENT ON COLUMN public.order_master.amount_cent IS NULL;
COMMENT ON COLUMN public.order_master.pay_type IS '1微信 2支付宝 3余额';
COMMENT ON COLUMN public.order_master.pay_status IS '0待支付 1成功 2关闭';
